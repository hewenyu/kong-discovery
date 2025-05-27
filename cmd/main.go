package main

import (
	"fmt"
	"os"

	"github.com/hewenyu/kong-discovery/internal/config"
	"go.uber.org/zap"
)

var logger config.Logger

func main() {
	// 初始化日志
	var err error
	logger, err = config.NewLogger(true) // 开发环境设置为true
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Kong Discovery Service Starting...", zap.String("version", "0.1.0"))
}
