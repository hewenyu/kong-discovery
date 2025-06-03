package namespace

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hewenyu/kong-discovery/internal/core/model"
	"github.com/hewenyu/kong-discovery/internal/store/etcd"
	"github.com/hewenyu/kong-discovery/internal/store/service"
)

const (
	// 命名空间存储的前缀
	namespacePrefix = "/namespaces/"
)

// EtcdNamespaceStore 实现基于etcd的命名空间存储
type EtcdNamespaceStore struct {
	client       *etcd.Client
	serviceStore service.ServiceStore
}

// NewEtcdNamespaceStore 创建一个新的基于etcd的命名空间存储
func NewEtcdNamespaceStore(client *etcd.Client, serviceStore service.ServiceStore) *EtcdNamespaceStore {
	return &EtcdNamespaceStore{
		client:       client,
		serviceStore: serviceStore,
	}
}

// getNamespaceKey 获取命名空间的存储键
func getNamespaceKey(name string) string {
	return namespacePrefix + name
}

// CreateNamespace 创建命名空间
func (s *EtcdNamespaceStore) CreateNamespace(ctx context.Context, namespace *model.Namespace) error {
	// 检查命名空间是否已存在
	existing, err := s.GetNamespaceByName(ctx, namespace.Name)
	if err != nil {
		return fmt.Errorf("检查命名空间是否存在失败: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("命名空间已存在: %s", namespace.Name)
	}

	// 设置创建时间和更新时间
	now := time.Now()
	namespace.CreatedAt = now
	namespace.UpdatedAt = now
	namespace.ServiceCount = 0

	// 序列化命名空间信息
	data, err := json.Marshal(namespace)
	if err != nil {
		return fmt.Errorf("序列化命名空间信息失败: %w", err)
	}

	// 存储命名空间信息
	namespaceKey := getNamespaceKey(namespace.Name)
	if err := s.client.Put(ctx, namespaceKey, data); err != nil {
		return fmt.Errorf("存储命名空间信息失败: %w", err)
	}

	return nil
}

// GetNamespaceByName 根据名称获取命名空间
func (s *EtcdNamespaceStore) GetNamespaceByName(ctx context.Context, name string) (*model.Namespace, error) {
	namespaceKey := getNamespaceKey(name)
	data, err := s.client.Get(ctx, namespaceKey)
	if err != nil {
		return nil, fmt.Errorf("获取命名空间信息失败: %w", err)
	}

	if data == nil {
		return nil, nil // 命名空间不存在
	}

	var namespace model.Namespace
	if err := json.Unmarshal(data, &namespace); err != nil {
		return nil, fmt.Errorf("解析命名空间信息失败: %w", err)
	}

	// 获取命名空间的服务数量
	count, err := s.GetNamespaceServiceCount(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("获取命名空间服务数量失败: %w", err)
	}
	namespace.ServiceCount = count

	return &namespace, nil
}

// ListNamespaces 获取所有命名空间
func (s *EtcdNamespaceStore) ListNamespaces(ctx context.Context) ([]*model.Namespace, error) {
	// 获取所有命名空间数据
	namespaces := make([]*model.Namespace, 0)
	data, err := s.client.GetWithPrefix(ctx, namespacePrefix)
	if err != nil {
		return nil, fmt.Errorf("获取命名空间列表失败: %w", err)
	}

	for _, v := range data {
		var namespace model.Namespace
		if err := json.Unmarshal(v, &namespace); err != nil {
			return nil, fmt.Errorf("解析命名空间信息失败: %w", err)
		}

		// 获取命名空间的服务数量
		count, err := s.GetNamespaceServiceCount(ctx, namespace.Name)
		if err != nil {
			return nil, fmt.Errorf("获取命名空间服务数量失败: %w", err)
		}
		namespace.ServiceCount = count

		namespaces = append(namespaces, &namespace)
	}

	return namespaces, nil
}

// DeleteNamespace 删除命名空间
func (s *EtcdNamespaceStore) DeleteNamespace(ctx context.Context, name string) error {
	// 检查命名空间是否存在
	namespace, err := s.GetNamespaceByName(ctx, name)
	if err != nil {
		return fmt.Errorf("检查命名空间是否存在失败: %w", err)
	}
	if namespace == nil {
		return fmt.Errorf("命名空间不存在: %s", name)
	}

	// 检查命名空间是否有服务
	count, err := s.GetNamespaceServiceCount(ctx, name)
	if err != nil {
		return fmt.Errorf("获取命名空间服务数量失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("命名空间包含服务，无法删除: %s (服务数量: %d)", name, count)
	}

	// 删除命名空间
	namespaceKey := getNamespaceKey(name)
	if err := s.client.Delete(ctx, namespaceKey); err != nil {
		return fmt.Errorf("删除命名空间失败: %w", err)
	}

	return nil
}

// GetNamespaceServiceCount 获取命名空间下的服务数量
func (s *EtcdNamespaceStore) GetNamespaceServiceCount(ctx context.Context, name string) (int, error) {
	services, err := s.serviceStore.ListServices(ctx, name)
	if err != nil {
		return 0, fmt.Errorf("获取命名空间服务列表失败: %w", err)
	}
	return len(services), nil
}
