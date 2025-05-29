package memory

import (
	"context"
	"sync"
	"time"

	"github.com/hewenyu/kong-discovery/pkg/storage"
)

// MemoryStorage 是基于内存的服务存储实现，主要用于测试
type MemoryStorage struct {
	services map[string]*storage.Service
	mutex    sync.RWMutex
}

// NewMemoryStorage 创建新的内存存储
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		services: make(map[string]*storage.Service),
	}
}

// RegisterService 注册服务实例
func (m *MemoryStorage) RegisterService(ctx context.Context, service *storage.Service) error {
	if service.ID == "" || service.Name == "" || service.IP == "" || service.Port <= 0 {
		return storage.NewInvalidArgumentError("服务ID、名称、IP和端口不能为空")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 设置注册时间和最后心跳时间
	now := time.Now()
	if service.RegisteredAt.IsZero() {
		service.RegisteredAt = now
	}
	if service.LastHeartbeat.IsZero() {
		service.LastHeartbeat = now
	}
	if service.Health == "" {
		service.Health = "healthy"
	}

	// 保存服务
	m.services[service.ID] = service
	return nil
}

// DeregisterService 注销服务实例
func (m *MemoryStorage) DeregisterService(ctx context.Context, serviceID string) error {
	if serviceID == "" {
		return storage.NewInvalidArgumentError("服务ID不能为空")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.services[serviceID]; !exists {
		return storage.NewNotFoundError("服务不存在: " + serviceID)
	}

	delete(m.services, serviceID)
	return nil
}

// GetService 获取服务实例详情
func (m *MemoryStorage) GetService(ctx context.Context, serviceID string) (*storage.Service, error) {
	if serviceID == "" {
		return nil, storage.NewInvalidArgumentError("服务ID不能为空")
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	service, exists := m.services[serviceID]
	if !exists {
		return nil, storage.NewNotFoundError("服务不存在: " + serviceID)
	}

	return service, nil
}

// ListServices 获取所有服务实例列表
func (m *MemoryStorage) ListServices(ctx context.Context) ([]*storage.Service, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	services := make([]*storage.Service, 0, len(m.services))
	for _, service := range m.services {
		services = append(services, service)
	}

	return services, nil
}

// ListServicesByName 获取指定名称的服务实例列表
func (m *MemoryStorage) ListServicesByName(ctx context.Context, serviceName string) ([]*storage.Service, error) {
	if serviceName == "" {
		return nil, storage.NewInvalidArgumentError("服务名称不能为空")
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var services []*storage.Service
	for _, service := range m.services {
		if service.Name == serviceName {
			services = append(services, service)
		}
	}

	return services, nil
}

// UpdateServiceHeartbeat 更新服务心跳时间
func (m *MemoryStorage) UpdateServiceHeartbeat(ctx context.Context, serviceID string) error {
	if serviceID == "" {
		return storage.NewInvalidArgumentError("服务ID不能为空")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	service, exists := m.services[serviceID]
	if !exists {
		return storage.NewNotFoundError("服务不存在: " + serviceID)
	}

	service.LastHeartbeat = time.Now()
	return nil
}

// CleanupStaleServices 清理过期的服务实例
func (m *MemoryStorage) CleanupStaleServices(ctx context.Context, timeout time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	staleServices := make([]string, 0)

	for id, service := range m.services {
		if now.Sub(service.LastHeartbeat) > timeout {
			staleServices = append(staleServices, id)
		}
	}

	for _, id := range staleServices {
		delete(m.services, id)
	}

	return nil
}
