package service

import (
	"context"
	"fmt"

	"github.com/hewenyu/kong-discovery/internal/core/model"
	namespaceStore "github.com/hewenyu/kong-discovery/internal/store/namespace"
	serviceStore "github.com/hewenyu/kong-discovery/internal/store/service"
)

// AdminServiceImpl 实现AdminService接口
type AdminServiceImpl struct {
	serviceStore   serviceStore.ServiceStore
	namespaceStore namespaceStore.NamespaceStore
}

// NewAdminService 创建一个新的AdminService实例
func NewAdminService(serviceStore serviceStore.ServiceStore, namespaceStore namespaceStore.NamespaceStore) AdminService {
	return &AdminServiceImpl{
		serviceStore:   serviceStore,
		namespaceStore: namespaceStore,
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

// CreateNamespace 创建命名空间
func (s *AdminServiceImpl) CreateNamespace(ctx context.Context, namespace *model.Namespace) error {
	if err := s.namespaceStore.CreateNamespace(ctx, namespace); err != nil {
		return fmt.Errorf("创建命名空间失败: %w", err)
	}
	return nil
}

// GetNamespaceByName 根据名称获取命名空间
func (s *AdminServiceImpl) GetNamespaceByName(ctx context.Context, name string) (*model.Namespace, error) {
	namespace, err := s.namespaceStore.GetNamespaceByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("获取命名空间失败: %w", err)
	}
	if namespace == nil {
		return nil, fmt.Errorf("命名空间不存在: %s", name)
	}
	return namespace, nil
}

// ListNamespaces 获取所有命名空间
func (s *AdminServiceImpl) ListNamespaces(ctx context.Context) ([]*model.Namespace, error) {
	namespaces, err := s.namespaceStore.ListNamespaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取命名空间列表失败: %w", err)
	}
	return namespaces, nil
}

// DeleteNamespace 删除命名空间
func (s *AdminServiceImpl) DeleteNamespace(ctx context.Context, name string) error {
	if err := s.namespaceStore.DeleteNamespace(ctx, name); err != nil {
		return fmt.Errorf("删除命名空间失败: %w", err)
	}
	return nil
}
