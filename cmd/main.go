package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hewenyu/kong-discovery/internal/config"
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
}
