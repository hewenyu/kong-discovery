package handler

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/hewenyu/kong-discovery/pkg/storage"
	"github.com/labstack/echo/v4"
)

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// HealthHandler 健康检查处理器
type HealthHandler struct {
	storage storage.ServiceStorage
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(storage storage.ServiceStorage) *HealthHandler {
	return &HealthHandler{
		storage: storage,
	}
}

// HealthCheck 健康检查处理函数
func (h *HealthHandler) HealthCheck(c echo.Context) error {
	// 创建一个带有超时的上下文，确保健康检查响应及时
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// 检查存储层是否正常
	_, err := h.storage.ListServices(ctx)
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, HealthResponse{
			Status:    "unhealthy",
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"error":      err.Error(),
				"component":  "storage",
				"version":    "1.0.0",
				"uptime":     time.Since(startTime).String(),
				"resources":  getResourceUsage(),
				"goroutines": runtime.NumGoroutine(),
			},
		})
	}

	return c.JSON(http.StatusOK, HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"version":    "1.0.0",
			"uptime":     time.Since(startTime).String(),
			"resources":  getResourceUsage(),
			"goroutines": runtime.NumGoroutine(),
		},
	})
}

// 应用启动时间
var startTime = time.Now()

// getResourceUsage 获取资源使用情况
func getResourceUsage() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"memory_alloc":   formatBytes(memStats.Alloc),
		"memory_sys":     formatBytes(memStats.Sys),
		"memory_heap":    formatBytes(memStats.HeapAlloc),
		"num_gc":         memStats.NumGC,
		"num_goroutines": runtime.NumGoroutine(),
	}
}

// formatBytes 将字节数格式化为可读形式
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
