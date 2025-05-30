package service

import (
	"context"
	"time"

	"github.com/hewenyu/kong-discovery/internal/core/model"
)

// ServiceStore 表示服务存储接口
type ServiceStore interface {
	// Register 注册服务
	Register(ctx context.Context, service *model.Service) error

	// Deregister 注销服务
	Deregister(ctx context.Context, serviceID string) error

	// UpdateHeartbeat 更新服务心跳
	UpdateHeartbeat(ctx context.Context, serviceID string) error

	// GetService 获取服务信息
	GetService(ctx context.Context, serviceID string) (*model.Service, error)

	// GetServiceByName 根据服务名和命名空间获取服务列表
	GetServiceByName(ctx context.Context, name, namespace string) ([]*model.Service, error)

	// ListServices 获取服务列表
	ListServices(ctx context.Context, namespace string) ([]*model.Service, error)

	// ListAllServices 获取所有服务列表
	ListAllServices(ctx context.Context) ([]*model.Service, error)

	// CleanupStaleServices 清理过期服务
	CleanupStaleServices(ctx context.Context, before time.Time) (int, error)
}
