package model

import (
	"time"
)

// 健康状态枚举
type HealthStatus string

const (
	// HealthStatusHealthy 表示服务健康
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusUnhealthy 表示服务不健康
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// Service 表示一个服务实例
type Service struct {
	ID            string            `json:"id"`
	Namespace     string            `json:"namespace"`
	Name          string            `json:"name"`
	IP            string            `json:"ip"`
	Port          int               `json:"port"`
	Tags          []string          `json:"tags,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Health        HealthStatus      `json:"health"`
	RegisteredAt  time.Time         `json:"registered_at"`
	LastHeartbeat time.Time         `json:"last_heartbeat"`
	TTL           time.Duration     `json:"ttl"`
}

// ServiceRegistrationRequest 表示服务注册请求
type ServiceRegistrationRequest struct {
	Name      string            `json:"name" binding:"required"`
	Namespace string            `json:"namespace"`
	IP        string            `json:"ip" binding:"required,ip"`
	Port      int               `json:"port" binding:"required,min=1,max=65535"`
	Tags      []string          `json:"tags,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	TTL       string            `json:"ttl,omitempty"`
}

// ServiceRegistrationResponse 表示服务注册响应
type ServiceRegistrationResponse struct {
	ServiceID    string    `json:"service_id"`
	RegisteredAt time.Time `json:"registered_at"`
}

// ServiceHeartbeatResponse 表示服务心跳响应
type ServiceHeartbeatResponse struct {
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

// ApiResponse 表示通用API响应
type ApiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
