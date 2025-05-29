package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hewenyu/kong-discovery/pkg/storage"
)

// ServiceStorage 实现基于内存的服务存储，用于测试
type ServiceStorage struct {
	services map[string]*storage.Service
	mu       sync.RWMutex
}

// NewServiceStorage 创建内存服务存储
func NewServiceStorage() *ServiceStorage {
	return &ServiceStorage{
		services: make(map[string]*storage.Service),
	}
}

// RegisterService 注册服务实例
func (s *ServiceStorage) RegisterService(ctx context.Context, service *storage.Service) error {
	if service.ID == "" || service.Name == "" || service.IP == "" || service.Port <= 0 {
		return storage.NewInvalidArgumentError("服务ID、名称、IP和端口不能为空")
	}

	// 设置注册时间和最后心跳时间
	now := time.Now()
	if service.RegisteredAt.IsZero() {
		service.RegisteredAt = now
	}
	service.LastHeartbeat = now

	// 如果未设置健康状态，默认为健康
	if service.Health == "" {
		service.Health = "healthy"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 存储服务
	s.services[service.ID] = service

	return nil
}

// DeregisterService 注销服务实例
func (s *ServiceStorage) DeregisterService(ctx context.Context, serviceID string) error {
	if serviceID == "" {
		return storage.NewInvalidArgumentError("服务ID不能为空")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.services[serviceID]; !exists {
		return storage.NewNotFoundError(fmt.Sprintf("服务不存在: %s", serviceID))
	}

	delete(s.services, serviceID)
	return nil
}

// GetService 获取服务实例详情
func (s *ServiceStorage) GetService(ctx context.Context, serviceID string) (*storage.Service, error) {
	if serviceID == "" {
		return nil, storage.NewInvalidArgumentError("服务ID不能为空")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	service, exists := s.services[serviceID]
	if !exists {
		return nil, storage.NewNotFoundError(fmt.Sprintf("服务不存在: %s", serviceID))
	}

	return service, nil
}

// ListServices 获取所有服务实例列表
func (s *ServiceStorage) ListServices(ctx context.Context) ([]*storage.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	services := make([]*storage.Service, 0, len(s.services))
	for _, service := range s.services {
		services = append(services, service)
	}

	return services, nil
}

// ListServicesByName 获取指定名称的服务实例列表
func (s *ServiceStorage) ListServicesByName(ctx context.Context, serviceName string) ([]*storage.Service, error) {
	if serviceName == "" {
		return nil, storage.NewInvalidArgumentError("服务名称不能为空")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*storage.Service, 0)
	for _, service := range s.services {
		if service.Name == serviceName {
			result = append(result, service)
		}
	}

	return result, nil
}

// UpdateServiceHeartbeat 更新服务心跳时间
func (s *ServiceStorage) UpdateServiceHeartbeat(ctx context.Context, serviceID string) error {
	if serviceID == "" {
		return storage.NewInvalidArgumentError("服务ID不能为空")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	service, exists := s.services[serviceID]
	if !exists {
		return storage.NewNotFoundError(fmt.Sprintf("服务不存在: %s", serviceID))
	}

	service.LastHeartbeat = time.Now()
	return nil
}

// CleanupStaleServices 清理过期的服务实例
func (s *ServiceStorage) CleanupStaleServices(ctx context.Context, timeout time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	staleIDs := make([]string, 0)

	// 找出所有过期的服务
	for id, service := range s.services {
		if now.Sub(service.LastHeartbeat) > timeout {
			staleIDs = append(staleIDs, id)
		}
	}

	// 删除过期服务
	for _, id := range staleIDs {
		delete(s.services, id)
	}

	return nil
}
