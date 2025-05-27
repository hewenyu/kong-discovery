package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hewenyu/kong-discovery/internal/config"
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

	// 等待信号以优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("接收到关闭信号，正在优雅关闭...")
}
