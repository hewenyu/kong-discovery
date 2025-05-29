package handler

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/hewenyu/kong-discovery/pkg/storage"
	"github.com/labstack/echo/v4"
)

// MetricsHandler 指标处理器
type MetricsHandler struct {
	storage        storage.ServiceStorage
	metrics        *Metrics
	metricsLock    sync.RWMutex
	lastUpdateTime time.Time
}

// Metrics 系统指标
type Metrics struct {
	ServiceCount      int                    `json:"service_count"`
	DNSQueryCount     int64                  `json:"dns_query_count"`
	DNSCacheHitRate   float64                `json:"dns_cache_hit_rate"`
	APIRequestCount   int64                  `json:"api_request_count"`
	AvgResponseTime   float64                `json:"avg_response_time"`
	ResourceUsage     map[string]interface{} `json:"resource_usage"`
	LastCollectedTime time.Time              `json:"last_collected_time"`
}

// NewMetricsHandler 创建指标处理器
func NewMetricsHandler(storage storage.ServiceStorage) *MetricsHandler {
	handler := &MetricsHandler{
		storage:        storage,
		metrics:        &Metrics{},
		lastUpdateTime: time.Now(),
	}

	// 初始化指标数据
	handler.updateMetrics()

	// 启动指标收集协程
	go handler.startMetricsCollector()

	return handler
}

// GetMetrics 获取系统指标
func (h *MetricsHandler) GetMetrics(c echo.Context) error {
	// 如果距离上次更新时间超过5秒，则更新指标
	if time.Since(h.lastUpdateTime) > 5*time.Second {
		h.updateMetrics()
	}

	// 读取指标数据
	h.metricsLock.RLock()
	metrics := h.metrics
	h.metricsLock.RUnlock()

	return c.JSON(http.StatusOK, ServiceResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    metrics,
	})
}

// 更新指标数据
func (h *MetricsHandler) updateMetrics() {
	// 获取服务数量
	services, err := h.storage.ListServices(context.Background())
	if err == nil {
		h.metricsLock.Lock()
		h.metrics.ServiceCount = len(services)
		h.metrics.ResourceUsage = getResourceUsage()
		h.metrics.LastCollectedTime = time.Now()
		h.lastUpdateTime = time.Now()
		h.metricsLock.Unlock()
	}
}

// 启动指标收集协程，定期更新指标
func (h *MetricsHandler) startMetricsCollector() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.updateMetrics()
	}
}

// IncrementDNSQueryCount 增加DNS查询计数
func (h *MetricsHandler) IncrementDNSQueryCount() {
	h.metricsLock.Lock()
	defer h.metricsLock.Unlock()
	h.metrics.DNSQueryCount++
}

// UpdateDNSCacheHitRate 更新DNS缓存命中率
func (h *MetricsHandler) UpdateDNSCacheHitRate(hitRate float64) {
	h.metricsLock.Lock()
	defer h.metricsLock.Unlock()
	h.metrics.DNSCacheHitRate = hitRate
}

// IncrementAPIRequestCount 增加API请求计数
func (h *MetricsHandler) IncrementAPIRequestCount() {
	h.metricsLock.Lock()
	defer h.metricsLock.Unlock()
	h.metrics.APIRequestCount++
}

// UpdateAvgResponseTime 更新平均响应时间
func (h *MetricsHandler) UpdateAvgResponseTime(responseTime float64) {
	h.metricsLock.Lock()
	defer h.metricsLock.Unlock()

	// 简单的移动平均值计算
	if h.metrics.AvgResponseTime == 0 {
		h.metrics.AvgResponseTime = responseTime
	} else {
		h.metrics.AvgResponseTime = (h.metrics.AvgResponseTime*9 + responseTime) / 10
	}
}
