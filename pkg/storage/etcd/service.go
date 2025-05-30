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
	client           *Client
	namespaceStorage *NamespaceStorage
}

// NewServiceStorage 创建etcd服务存储
func NewServiceStorage(client *Client) *ServiceStorage {
	return &ServiceStorage{
		client:           client,
		namespaceStorage: NewNamespaceStorage(client),
	}
}

// RegisterService 注册服务实例
func (s *ServiceStorage) RegisterService(ctx context.Context, service *storage.Service) error {
	if service.ID == "" || service.Name == "" || service.IP == "" || service.Port <= 0 {
		return storage.NewInvalidArgumentError("服务ID、名称、IP和端口不能为空")
	}

	// 如果没有指定命名空间，使用默认命名空间
	if service.Namespace == "" {
		service.Namespace = "default"
	}

	// 检查命名空间是否存在
	_, err := s.namespaceStorage.GetNamespace(ctx, service.Namespace)
	if err != nil {
		if se, ok := err.(*storage.StorageError); ok && se.Code == storage.ErrNotFound {
			// 如果是default命名空间不存在，则自动创建
			if service.Namespace == "default" {
				defaultNs := &storage.Namespace{
					Name:        "default",
					Description: "默认命名空间",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				if err := s.namespaceStorage.CreateNamespace(ctx, defaultNs); err != nil {
					return storage.NewInternalError(fmt.Sprintf("创建默认命名空间失败: %v", err))
				}
			} else {
				return storage.NewNotFoundError(fmt.Sprintf("命名空间不存在: %s", service.Namespace))
			}
		} else {
			return err
		}
	}

	// 设置注册时间和最后心跳时间
	now := time.Now()
	if service.RegisteredAt.IsZero() {
		service.RegisteredAt = now
	}

	// 只有在LastHeartbeat为零值时才设置为当前时间
	// 这允许调用者提供自定义的心跳时间
	if service.LastHeartbeat.IsZero() {
		service.LastHeartbeat = now
	}

	// 如果未设置健康状态，默认为健康
	if service.Health == "" {
		service.Health = "healthy"
	}

	// 序列化服务数据
	data, err := json.Marshal(service)
	if err != nil {
		return storage.NewInternalError(fmt.Sprintf("序列化服务数据失败: %v", err))
	}

	// 存储到etcd，使用命名空间前缀
	key := s.client.GetNamespacedServiceKey(service.Namespace, service.ID)

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

	// 更新命名空间服务计数
	err = s.namespaceStorage.UpdateNamespaceServiceCount(ctx, service.Namespace, 1)
	if err != nil {
		// 仅记录错误，不影响服务注册
		fmt.Printf("更新命名空间服务计数失败: %v\n", err)
	}

	return nil
}

// DeregisterService 注销服务实例
func (s *ServiceStorage) DeregisterService(ctx context.Context, serviceID string) error {
	if serviceID == "" {
		return storage.NewInvalidArgumentError("服务ID不能为空")
	}

	// 获取服务信息，以便知道服务所在的命名空间
	service, err := s.GetService(ctx, serviceID)
	if err != nil {
		return err
	}

	// 从etcd中删除服务
	key := s.client.GetNamespacedServiceKey(service.Namespace, serviceID)
	resp, err := s.client.GetClient().Delete(ctx, key)
	if err != nil {
		return storage.NewInternalError(fmt.Sprintf("从etcd删除失败: %v", err))
	}

	if resp.Deleted == 0 {
		return storage.NewNotFoundError(fmt.Sprintf("服务不存在: %s", serviceID))
	}

	// 更新命名空间服务计数
	err = s.namespaceStorage.UpdateNamespaceServiceCount(ctx, service.Namespace, -1)
	if err != nil {
		// 仅记录错误，不影响服务注销
		fmt.Printf("更新命名空间服务计数失败: %v\n", err)
	}

	return nil
}

// GetService 获取服务实例详情
func (s *ServiceStorage) GetService(ctx context.Context, serviceID string) (*storage.Service, error) {
	if serviceID == "" {
		return nil, storage.NewInvalidArgumentError("服务ID不能为空")
	}

	// 由于不知道服务所在的命名空间，需要查询所有命名空间
	namespaces, err := s.namespaceStorage.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}

	// 如果没有命名空间，尝试在默认命名空间中查找
	if len(namespaces) == 0 {
		key := s.client.GetNamespacedServiceKey("default", serviceID)
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

	// 在每个命名空间中查找服务
	for _, ns := range namespaces {
		key := s.client.GetNamespacedServiceKey(ns.Name, serviceID)
		resp, err := s.client.GetClient().Get(ctx, key)
		if err != nil {
			continue
		}

		if len(resp.Kvs) > 0 {
			var service storage.Service
			if err := json.Unmarshal(resp.Kvs[0].Value, &service); err != nil {
				continue
			}
			return &service, nil
		}
	}

	// 尝试兼容旧格式（无命名空间）
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

	// 如果找到了旧格式的服务，设置默认命名空间
	if service.Namespace == "" {
		service.Namespace = "default"
	}

	return &service, nil
}

// ListServices 获取所有服务实例列表
func (s *ServiceStorage) ListServices(ctx context.Context) ([]*storage.Service, error) {
	// 获取所有命名空间
	namespaces, err := s.namespaceStorage.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}

	services := make([]*storage.Service, 0)

	// 如果没有命名空间，尝试使用旧的前缀获取服务
	if len(namespaces) == 0 {
		// 兼容旧格式，无命名空间
		prefix := s.client.GetServicesPrefix()
		resp, err := s.client.GetClient().Get(ctx, prefix, clientv3.WithPrefix())
		if err != nil {
			return nil, storage.NewInternalError(fmt.Sprintf("从etcd读取失败: %v", err))
		}

		for _, kv := range resp.Kvs {
			var service storage.Service
			if err := json.Unmarshal(kv.Value, &service); err != nil {
				// 忽略无法解析的数据，继续处理其他数据
				continue
			}
			if service.Namespace == "" {
				service.Namespace = "default"
			}
			services = append(services, &service)
		}

		return services, nil
	}

	// 遍历每个命名空间，获取其中的服务
	for _, ns := range namespaces {
		prefix := s.client.GetNamespaceServicesPrefix(ns.Name)
		resp, err := s.client.GetClient().Get(ctx, prefix, clientv3.WithPrefix())
		if err != nil {
			continue
		}

		for _, kv := range resp.Kvs {
			var service storage.Service
			if err := json.Unmarshal(kv.Value, &service); err != nil {
				// 忽略无法解析的数据，继续处理其他数据
				continue
			}
			services = append(services, &service)
		}
	}

	// 兼容旧格式，无命名空间
	prefix := s.client.GetServicesPrefix()
	resp, err := s.client.GetClient().Get(ctx, prefix, clientv3.WithPrefix())
	if err == nil {
		for _, kv := range resp.Kvs {
			var service storage.Service
			if err := json.Unmarshal(kv.Value, &service); err != nil {
				// 忽略无法解析的数据，继续处理其他数据
				continue
			}
			if service.Namespace == "" {
				service.Namespace = "default"
			}
			services = append(services, &service)
		}
	}

	return services, nil
}

// ListServicesByNamespace 获取指定命名空间的服务实例列表
func (s *ServiceStorage) ListServicesByNamespace(ctx context.Context, namespace string) ([]*storage.Service, error) {
	if namespace == "" {
		return nil, storage.NewInvalidArgumentError("命名空间不能为空")
	}

	// 检查命名空间是否存在
	_, err := s.namespaceStorage.GetNamespace(ctx, namespace)
	if err != nil {
		return nil, err
	}

	// 获取指定命名空间的服务
	prefix := s.client.GetNamespaceServicesPrefix(namespace)
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

// ListServicesByNameAndNamespace 获取指定命名空间和名称的服务实例列表
func (s *ServiceStorage) ListServicesByNameAndNamespace(ctx context.Context, namespace, serviceName string) ([]*storage.Service, error) {
	if namespace == "" {
		return nil, storage.NewInvalidArgumentError("命名空间不能为空")
	}
	if serviceName == "" {
		return nil, storage.NewInvalidArgumentError("服务名称不能为空")
	}

	// 获取指定命名空间的服务
	services, err := s.ListServicesByNamespace(ctx, namespace)
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
	fmt.Printf("当前时间: %v, 清理超时阈值: %v\n", now, timeout)

	for _, service := range services {
		timeSinceLastHeartbeat := now.Sub(service.LastHeartbeat)
		fmt.Printf("服务 %s 上次心跳: %v, 时间差: %v\n", service.ID, service.LastHeartbeat, timeSinceLastHeartbeat)

		if timeSinceLastHeartbeat > timeout {
			fmt.Printf("服务 %s 超时，准备清理\n", service.ID)
			if err := s.DeregisterService(ctx, service.ID); err != nil {
				fmt.Printf("清理过期服务 %s 失败: %v\n", service.ID, err)
				continue
			}
			fmt.Printf("清理过期服务 %s 成功\n", service.ID)
		}
	}

	return nil
}

// GetClient 获取底层的etcd Client
func (s *ServiceStorage) GetClient() *Client {
	return s.client
}
