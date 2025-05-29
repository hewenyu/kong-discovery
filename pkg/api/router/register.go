package router

import (
	"github.com/hewenyu/kong-discovery/pkg/api/handler"
	"github.com/labstack/echo/v4"
)

// RegisterRoutes 配置服务注册相关路由（8080端口）
func RegisterRoutes(e *echo.Echo, serviceHandler *handler.ServiceHandler, healthHandler *handler.HealthHandler) {
	// API分组，版本v1
	api := e.Group("/api/v1")

	// 服务注册相关路由
	services := api.Group("/services")
	services.POST("", serviceHandler.RegisterService)                     // 注册服务
	services.DELETE("/:serviceId", serviceHandler.DeregisterService)      // 注销服务
	services.PUT("/:serviceId/heartbeat", serviceHandler.UpdateHeartbeat) // 心跳更新

	// 健康检查
	api.GET("/health", healthHandler.HealthCheck)
}
