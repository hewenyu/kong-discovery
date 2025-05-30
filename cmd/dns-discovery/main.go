package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hewenyu/kong-discovery/internal/core/config"
	"github.com/hewenyu/kong-discovery/internal/dns"
	"github.com/hewenyu/kong-discovery/internal/registration"
	"github.com/hewenyu/kong-discovery/internal/store/etcd"
	"github.com/hewenyu/kong-discovery/internal/store/service"
)

var (
	configFile string
)

func init() {
	flag.StringVar(&configFile, "config", "configs/config.yaml", "配置文件路径")
}

func main() {
	flag.Parse()

	// 从配置文件加载配置
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	log.Printf("配置加载成功，DNS域名: %s, etcd端点: %v",
		cfg.Server.DNS.Domain, cfg.Etcd.Endpoints)

	// 创建上下文，用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 设置信号处理，以便优雅关闭
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// 初始化etcd客户端
	etcdClient, err := etcd.NewClient(&cfg.Etcd)
	if err != nil {
		log.Fatalf("初始化etcd客户端失败: %v", err)
	}
	defer func() {
		if err := etcdClient.Close(); err != nil {
			log.Printf("关闭etcd客户端失败: %v", err)
		}
	}()

	// 测试etcd连接
	testKey := "/test/connection"
	testValue := []byte("test-connection")
	err = etcdClient.Put(ctx, testKey, testValue)
	if err != nil {
		log.Fatalf("etcd连接测试失败: %v", err)
	}
	value, err := etcdClient.Get(ctx, testKey)
	if err != nil {
		log.Fatalf("etcd读取测试失败: %v", err)
	}
	if string(value) != string(testValue) {
		log.Fatalf("etcd测试值不一致，期望 %s，实际 %s", testValue, value)
	}
	err = etcdClient.Delete(ctx, testKey)
	if err != nil {
		log.Fatalf("etcd删除测试失败: %v", err)
	}
	log.Println("etcd连接测试成功")

	// 创建服务存储
	serviceStore := service.NewEtcdServiceStore(etcdClient, cfg.Namespace.Default)

	// 启动服务注册API (8080端口)
	registrationServer := registration.NewServer(etcdClient, cfg)
	if err := registrationServer.Start(); err != nil {
		log.Fatalf("启动服务注册API失败: %v", err)
	}

	// TODO: 启动管理API (9090端口)

	// 启动DNS服务 (53端口)
	dnsConfig := &dns.Config{
		DNSAddr:      fmt.Sprintf(":%d", cfg.Server.DNS.Port),
		Domain:       cfg.Server.DNS.Domain,
		TTL:          cfg.Server.DNS.TTL,
		Timeout:      5 * time.Second,
		UpstreamDNS:  cfg.Server.DNS.Upstream.Servers,
		EnableTCP:    cfg.Server.DNS.TCPEnabled,
		EnableUDP:    cfg.Server.DNS.UDPEnabled,
		ServiceStore: serviceStore, // 传递服务存储
	}

	dnsServer := dns.NewServer(dnsConfig)
	if err := dnsServer.Start(ctx); err != nil {
		log.Fatalf("启动DNS服务失败: %v", err)
	}

	fmt.Printf("服务已启动，DNS服务(端口%d)，服务注册API(端口%d)，管理API(端口%d)\n",
		cfg.Server.DNS.Port, cfg.Server.Registration.Port, cfg.Server.Admin.Port)

	// 等待终止信号
	sig := <-signalChan
	log.Printf("接收到信号: %v，准备关闭服务", sig)

	// 执行优雅关闭
	cancel()

	// 使用ctx设置一个超时，等待服务关闭
	const shutdownTimeout = 5 * time.Second
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	// 等待各服务关闭完成
	var wg sync.WaitGroup

	// 关闭服务注册API
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := registrationServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("关闭服务注册API失败: %v", err)
		}
	}()

	// 关闭DNS服务
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := dnsServer.Stop(); err != nil {
			log.Printf("关闭DNS服务失败: %v", err)
		}
	}()

	// 等待所有服务关闭
	wg.Wait()

	log.Println("服务已关闭")
}
