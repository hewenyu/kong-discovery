package service

import (
	"context"

	"github.com/hewenyu/kong-discovery/internal/core/model"
)

// AdminService 定义管理API的服务层接口
type AdminService interface {
	// ListServices 查询服务列表，如果namespace为空，则返回所有命名空间的服务
	ListServices(ctx context.Context, namespace string) ([]*model.Service, error)

	// GetServiceByID 根据ID获取服务详情
	GetServiceByID(ctx context.Context, serviceID string) (*model.Service, error)
}
