package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/hewenyu/kong-discovery/pkg/api/handler"
	"github.com/hewenyu/kong-discovery/pkg/api/router"
	"github.com/hewenyu/kong-discovery/pkg/config"
	"github.com/hewenyu/kong-discovery/pkg/dns"
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

	// 创建Echo实例 - 服务注册API (8080端口)
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

	// 创建上下文用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动后台任务：清理过期服务
	go startCleanupTask(ctx, serviceStorage, cfg.Heartbeat.Timeout)

	// 初始化并启动DNS服务器
	dnsServer, err := dns.NewServer(cfg, serviceStorage)
	if err != nil {
		log.Fatalf("创建DNS服务器失败: %v", err)
	}

	// 启动DNS服务器
	if err := dnsServer.Start(ctx); err != nil {
		log.Fatalf("启动DNS服务器失败: %v", err)
	}
	log.Printf("DNS服务器启动成功，监听端口: %d", cfg.Server.DNSPort)

	// 创建管理API Echo实例 (9090端口)
	adminAPI := echo.New()
	adminAPI.Validator = &CustomValidator{validator: validate}

	// 管理API中间件
	adminAPI.Use(middleware.Logger())
	adminAPI.Use(middleware.Recover())
	adminAPI.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

	// 创建管理API处理器
	adminServiceHandler := handler.NewAdminServiceHandler(serviceStorage)
	metricsHandler := handler.NewMetricsHandler(serviceStorage)

	// 注册管理API路由
	router.RegisterAdminRoutes(adminAPI, adminServiceHandler, healthHandler, metricsHandler)

	// 管理API基础路由
	adminAPI.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Kong DNS Discovery Admin API 运行正常",
			"status":  "running",
			"version": "1.0.0",
		})
	})

	// 创建服务注册HTTP服务器 (8080端口)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.RegisterPort),
		Handler: e,
	}

	// 创建管理API HTTP服务器 (9090端口)
	adminServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.AdminPort),
		Handler: adminAPI,
	}

	// 启动服务注册HTTP服务器
	go func() {
		log.Printf("启动 Kong DNS Discovery 服务注册API，监听端口: %d", cfg.Server.RegisterPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务注册HTTP服务器启动失败: %v", err)
		}
	}()

	// 启动管理API HTTP服务器
	go func() {
		log.Printf("启动 Kong DNS Discovery 管理API，监听端口: %d", cfg.Server.AdminPort)
		if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("管理API HTTP服务器启动失败: %v", err)
		}
	}()

	// 监听系统信号以优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务器...")

	// 优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// 关闭HTTP服务器
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("服务注册HTTP服务器关闭失败: %v", err)
	}

	// 关闭管理API HTTP服务器
	if err := adminServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("管理API HTTP服务器关闭失败: %v", err)
	}

	// 关闭DNS服务器
	if err := dnsServer.Stop(); err != nil {
		log.Printf("DNS服务器关闭失败: %v", err)
	}

	log.Println("服务器已关闭")
}

// startCleanupTask 启动清理过期服务的定时任务
func startCleanupTask(ctx context.Context, storage *etcd.ServiceStorage, timeoutSeconds int) {
	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			timeout := time.Duration(timeoutSeconds) * time.Second
			if err := storage.CleanupStaleServices(ctx, timeout); err != nil {
				log.Printf("清理过期服务失败: %v", err)
			} else {
				log.Println("清理过期服务完成")
			}
		case <-ctx.Done():
			log.Println("停止清理过期服务任务")
			return
		}
	}
}
