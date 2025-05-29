package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hewenyu/kong-discovery/pkg/storage"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// ServiceStorage 实现基于etcd的服务存储
type ServiceStorage struct {
	client *Client
}

// NewServiceStorage 创建etcd服务存储
func NewServiceStorage(client *Client) *ServiceStorage {
	return &ServiceStorage{
		client: client,
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

	// 序列化服务数据
	data, err := json.Marshal(service)
	if err != nil {
		return storage.NewInternalError(fmt.Sprintf("序列化服务数据失败: %v", err))
	}

	// 存储到etcd
	key := s.client.GetServiceKey(service.ID)

	// 如果设置了TTL，创建带有租约的key
	if service.TTL > 0 {
		// 创建租约
		lease, err := s.client.GetClient().Grant(ctx, int64(service.TTL))
		if err != nil {
			return storage.NewInternalError(fmt.Sprintf("创建etcd租约失败: %v", err))
		}

		// 带租约写入
		_, err = s.client.GetClient().Put(ctx, key, string(data), clientv3.WithLease(lease.ID))
		if err != nil {
			return storage.NewInternalError(fmt.Sprintf("写入etcd失败: %v", err))
		}
	} else {
		// 无租约写入
		_, err = s.client.GetClient().Put(ctx, key, string(data))
		if err != nil {
			return storage.NewInternalError(fmt.Sprintf("写入etcd失败: %v", err))
		}
	}

	return nil
}

// DeregisterService 注销服务实例
func (s *ServiceStorage) DeregisterService(ctx context.Context, serviceID string) error {
	if serviceID == "" {
		return storage.NewInvalidArgumentError("服务ID不能为空")
	}

	key := s.client.GetServiceKey(serviceID)
	resp, err := s.client.GetClient().Delete(ctx, key)
	if err != nil {
		return storage.NewInternalError(fmt.Sprintf("从etcd删除失败: %v", err))
	}

	if resp.Deleted == 0 {
		return storage.NewNotFoundError(fmt.Sprintf("服务不存在: %s", serviceID))
	}

	return nil
}

// GetService 获取服务实例详情
func (s *ServiceStorage) GetService(ctx context.Context, serviceID string) (*storage.Service, error) {
	if serviceID == "" {
		return nil, storage.NewInvalidArgumentError("服务ID不能为空")
	}

	key := s.client.GetServiceKey(serviceID)
	resp, err := s.client.GetClient().Get(ctx, key)
	if err != nil {
		return nil, storage.NewInternalError(fmt.Sprintf("从etcd读取失败: %v", err))
	}

	if len(resp.Kvs) == 0 {
		return nil, storage.NewNotFoundError(fmt.Sprintf("服务不存在: %s", serviceID))
	}

	var service storage.Service
	if err := json.Unmarshal(resp.Kvs[0].Value, &service); err != nil {
		return nil, storage.NewInternalError(fmt.Sprintf("解析服务数据失败: %v", err))
	}

	return &service, nil
}

// ListServices 获取所有服务实例列表
func (s *ServiceStorage) ListServices(ctx context.Context) ([]*storage.Service, error) {
	prefix := s.client.GetServicesPrefix()
	resp, err := s.client.GetClient().Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, storage.NewInternalError(fmt.Sprintf("从etcd读取失败: %v", err))
	}

	services := make([]*storage.Service, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var service storage.Service
		if err := json.Unmarshal(kv.Value, &service); err != nil {
			// 忽略无法解析的数据，继续处理其他数据
			continue
		}
		services = append(services, &service)
	}

	return services, nil
}

// ListServicesByName 获取指定名称的服务实例列表
func (s *ServiceStorage) ListServicesByName(ctx context.Context, serviceName string) ([]*storage.Service, error) {
	if serviceName == "" {
		return nil, storage.NewInvalidArgumentError("服务名称不能为空")
	}

	// 获取所有服务
	services, err := s.ListServices(ctx)
	if err != nil {
		return nil, err
	}

	// 过滤出指定名称的服务
	result := make([]*storage.Service, 0)
	for _, service := range services {
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

	// 获取服务
	service, err := s.GetService(ctx, serviceID)
	if err != nil {
		return err
	}

	// 更新心跳时间
	service.LastHeartbeat = time.Now()

	// 更新服务
	return s.RegisterService(ctx, service)
}

// CleanupStaleServices 清理过期的服务实例
func (s *ServiceStorage) CleanupStaleServices(ctx context.Context, timeout time.Duration) error {
	// 获取所有服务
	services, err := s.ListServices(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, service := range services {
		// 如果心跳超时，注销服务
		if now.Sub(service.LastHeartbeat) > timeout {
			if err := s.DeregisterService(ctx, service.ID); err != nil {
				// 忽略单个服务注销失败，继续处理其他服务
				continue
			}
		}
	}

	return nil
}
