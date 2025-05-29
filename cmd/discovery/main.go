package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/hewenyu/kong-discovery/pkg/api/handler"
	"github.com/hewenyu/kong-discovery/pkg/api/router"
	"github.com/hewenyu/kong-discovery/pkg/config"
	"github.com/hewenyu/kong-discovery/pkg/storage/etcd"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// 自定义验证器
type CustomValidator struct {
	validator *validator.Validate
}

// Validate 实现echo.Validator接口
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func main() {
	// 命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建Etcd客户端
	etcdClient, err := etcd.NewClient(&cfg.Etcd)
	if err != nil {
		log.Fatalf("连接Etcd失败: %v", err)
	}

	// 创建服务存储
	serviceStorage := etcd.NewServiceStorage(etcdClient)

	// 创建Echo实例
	e := echo.New()

	// 设置验证器
	validate := validator.New()
	e.Validator = &CustomValidator{validator: validate}

	// 中间件
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

	// 创建服务处理器
	serviceHandler := handler.NewServiceHandler(serviceStorage)

	// 创建健康检查处理器
	healthHandler := handler.NewHealthHandler(serviceStorage)

	// 注册路由
	router.RegisterRoutes(e, serviceHandler, healthHandler)

	// 基础路由
	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Kong DNS Discovery Service 运行正常",
			"status":  "running",
			"version": "1.0.0",
		})
	})

	// 启动后台任务：清理过期服务
	go startCleanupTask(serviceStorage, cfg.Heartbeat.Timeout)

	// 启动服务器
	log.Printf("启动 Kong DNS Discovery 服务，监听端口: %d", cfg.Server.RegisterPort)
	if err := e.Start(fmt.Sprintf(":%d", cfg.Server.RegisterPort)); err != nil && err != http.ErrServerClosed {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// startCleanupTask 启动清理过期服务的定时任务
func startCleanupTask(storage *etcd.ServiceStorage, timeoutSeconds int) {
	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		timeout := time.Duration(timeoutSeconds) * time.Second
		if err := storage.CleanupStaleServices(ctx, timeout); err != nil {
			log.Printf("清理过期服务失败: %v", err)
		} else {
			log.Println("清理过期服务完成")
		}
	}
}
