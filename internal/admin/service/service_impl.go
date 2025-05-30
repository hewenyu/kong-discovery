package service

import (
	"context"
	"fmt"

	"github.com/hewenyu/kong-discovery/internal/core/model"
	serviceStore "github.com/hewenyu/kong-discovery/internal/store/service"
)

// AdminServiceImpl 实现AdminService接口
type AdminServiceImpl struct {
	serviceStore serviceStore.ServiceStore
}

// NewAdminService 创建一个新的AdminService实例
func NewAdminService(serviceStore serviceStore.ServiceStore) AdminService {
	return &AdminServiceImpl{
		serviceStore: serviceStore,
	}
}

// ListServices 查询服务列表，如果namespace为空，则返回所有命名空间的服务
func (s *AdminServiceImpl) ListServices(ctx context.Context, namespace string) ([]*model.Service, error) {
	if namespace == "" {
		// 返回所有命名空间的服务
		return s.serviceStore.ListAllServices(ctx)
	}

	// 返回指定命名空间的服务
	return s.serviceStore.ListServices(ctx, namespace)
}

// GetServiceByID 根据ID获取服务详情
func (s *AdminServiceImpl) GetServiceByID(ctx context.Context, serviceID string) (*model.Service, error) {
	service, err := s.serviceStore.GetService(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("获取服务详情失败: %w", err)
	}

	if service == nil {
		return nil, fmt.Errorf("服务不存在: %s", serviceID)
	}

	return service, nil
}
