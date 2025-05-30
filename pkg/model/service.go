package model

import "time"

// HealthStatus 表示服务健康状态
type HealthStatus string

const (
	// HealthStatusHealthy 健康状态
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusUnhealthy 不健康状态
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	// HealthStatusUnknown 未知状态
	HealthStatusUnknown HealthStatus = "unknown"
)

// Service 表示一个服务实例
type Service struct {
	ID            string            `json:"id"`             // 服务实例唯一ID
	Namespace     string            `json:"namespace"`      // 服务所属命名空间
	Name          string            `json:"name"`           // 服务名称
	IP            string            `json:"ip"`             // 服务IP地址
	Port          int               `json:"port"`           // 服务端口
	Tags          []string          `json:"tags"`           // 服务标签
	Metadata      map[string]string `json:"metadata"`       // 服务元数据
	Health        HealthStatus      `json:"health"`         // 服务健康状态
	RegisteredAt  time.Time         `json:"registered_at"`  // 注册时间
	LastHeartbeat time.Time         `json:"last_heartbeat"` // 最后心跳时间
	TTL           int               `json:"ttl"`            // 生存时间(秒)
}
