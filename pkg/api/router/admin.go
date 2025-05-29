package router

import (
	"github.com/hewenyu/kong-discovery/pkg/api/handler"
	"github.com/labstack/echo/v4"
)

// RegisterAdminRoutes 配置管理API相关路由（9090端口）
func RegisterAdminRoutes(e *echo.Echo, serviceHandler *handler.AdminServiceHandler, healthHandler *handler.HealthHandler, metricsHandler *handler.MetricsHandler) {
	// API分组，版本v1
	api := e.Group("/api/v1")

	// 服务管理相关路由
	services := api.Group("/services")
	services.GET("", serviceHandler.ListServices)          // 查询服务列表
	services.GET("/:serviceId", serviceHandler.GetService) // 查询服务详情

	// 系统状态相关路由
	api.GET("/status", serviceHandler.GetSystemStatus) // 系统状态
	api.GET("/health", healthHandler.HealthCheck)      // 健康检查

	// 统计指标相关路由
	api.GET("/metrics", metricsHandler.GetMetrics) // 获取系统指标
}
