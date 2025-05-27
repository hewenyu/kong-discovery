package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/hewenyu/kong-discovery/internal/apihandler"
	"github.com/hewenyu/kong-discovery/internal/config"
	"github.com/hewenyu/kong-discovery/internal/dnsserver"
	"github.com/hewenyu/kong-discovery/internal/etcdclient"
	"go.uber.org/zap"
)

var (
	logger     config.Logger
	configFile string
	appConfig  *config.Config
)

func init() {
	// 解析命令行参数
	flag.StringVar(&configFile, "config", "", "配置文件路径")
}

func main() {
	flag.Parse()

	// 加载配置
	var err error
	appConfig, err = config.LoadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logger, err = config.NewLogger(appConfig.Log.Development)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}

	// 打印启动信息
	logger.Info("Kong Discovery Service Starting...",
		zap.String("version", "0.1.0"),
		zap.String("etcd_endpoints", fmt.Sprintf("%v", appConfig.Etcd.Endpoints)),
		zap.Int("dns_port", appConfig.DNS.Port),
		zap.Int("management_api_port", appConfig.API.Management.Port),
		zap.Int("registration_api_port", appConfig.API.Registration.Port),
	)

	// 初始化etcd客户端
	etcdClient := etcdclient.NewEtcdClient(appConfig, logger)
	if err := etcdClient.Connect(); err != nil {
		logger.Error("连接etcd失败", zap.Error(err))
		os.Exit(1)
	}
	defer etcdClient.Close()

	// 检查etcd连接状态
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := etcdClient.Ping(ctx); err != nil {
		logger.Error("etcd健康检查失败", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("etcd连接成功并通过健康检查")

	// 初始化并启动API处理器
	apiHandler := apihandler.NewAPIHandler(appConfig, logger)

	// 启动管理API服务
	if err := apiHandler.StartManagementAPI(); err != nil {
		logger.Error("启动管理API服务失败", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("管理API服务启动成功",
		zap.String("address", appConfig.API.Management.ListenAddress),
		zap.Int("port", appConfig.API.Management.Port))

	// 启动服务注册API服务
	if err := apiHandler.StartRegistrationAPI(); err != nil {
		logger.Error("启动服务注册API服务失败", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("服务注册API服务启动成功",
		zap.String("address", appConfig.API.Registration.ListenAddress),
		zap.Int("port", appConfig.API.Registration.Port))

	// 创建测试DNS记录
	testCtx, testCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer testCancel()

	// 1. 创建常规DNS记录
	testRecord := &etcdclient.DNSRecord{
		Type:  "A",
		Value: "192.168.1.100",
		TTL:   300,
	}
	if err := etcdClient.PutDNSRecord(testCtx, "kong.test", testRecord); err != nil {
		logger.Warn("创建测试DNS记录失败", zap.Error(err))
	} else {
		logger.Info("创建测试DNS记录成功", zap.String("domain", "kong.test"))
	}

	// 2. 注册服务实例
	instanceID := uuid.New().String()
	serviceInstance := &etcdclient.ServiceInstance{
		ServiceName: "nginx",
		InstanceID:  instanceID,
		IPAddress:   "192.168.1.200",
		Port:        8080,
		Metadata: map[string]string{
			"version": "1.0.0",
			"env":     "test",
		},
		TTL: 60,
	}

	if err := etcdClient.RegisterService(testCtx, serviceInstance); err != nil {
		logger.Warn("注册测试服务实例失败", zap.Error(err))
	} else {
		logger.Info("注册测试服务实例成功",
			zap.String("service", serviceInstance.ServiceName),
			zap.String("id", serviceInstance.InstanceID))
	}

	// 初始化DNS服务器并注入etcd客户端
	dnsServer := dnsserver.NewDNSServer(appConfig, logger)
	dnsServer.SetEtcdClient(etcdClient)

	// 启动DNS服务器
	if err := dnsServer.Start(); err != nil {
		logger.Error("启动DNS服务器失败", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("DNS服务器启动成功",
		zap.String("address", appConfig.DNS.ListenAddress),
		zap.Int("port", appConfig.DNS.Port),
		zap.String("protocol", appConfig.DNS.Protocol))

	// 等待信号以优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("接收到关闭信号，正在优雅关闭...")

	// 优雅关闭所有服务
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// 关闭DNS服务器
	if err := dnsServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("关闭DNS服务器失败", zap.Error(err))
	}

	// 关闭API服务
	if err := apiHandler.Shutdown(shutdownCtx); err != nil {
		logger.Error("关闭API服务失败", zap.Error(err))
	}
}
