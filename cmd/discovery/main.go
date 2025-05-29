package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/hewenyu/kong-discovery/pkg/config"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// 命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建Echo实例
	e := echo.New()

	// 中间件
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// 路由
	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Kong DNS Discovery Service 运行正常",
			"status":  "running",
		})
	})

	// 健康检查
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "healthy",
		})
	})

	// 启动服务器
	log.Printf("启动 Kong DNS Discovery 服务，监听端口: %d", cfg.Server.RegisterPort)
	if err := e.Start(fmt.Sprintf(":%d", cfg.Server.RegisterPort)); err != nil && err != http.ErrServerClosed {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
