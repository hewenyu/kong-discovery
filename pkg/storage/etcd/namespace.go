package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hewenyu/kong-discovery/pkg/storage"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// NamespaceStorage 实现基于etcd的命名空间存储
type NamespaceStorage struct {
	client *Client
}

// NewNamespaceStorage 创建etcd命名空间存储
func NewNamespaceStorage(client *Client) *NamespaceStorage {
	return &NamespaceStorage{
		client: client,
	}
}

// CreateNamespace 创建命名空间
func (s *NamespaceStorage) CreateNamespace(ctx context.Context, namespace *storage.Namespace) error {
	if namespace.Name == "" {
		return storage.NewInvalidArgumentError("命名空间名称不能为空")
	}

	// 检查命名空间是否已存在
	key := s.client.GetNamespaceKey(namespace.Name)
	resp, err := s.client.GetClient().Get(ctx, key)
	if err != nil {
		return storage.NewInternalError(fmt.Sprintf("检查命名空间是否存在失败: %v", err))
	}

	if len(resp.Kvs) > 0 {
		return storage.NewAlreadyExistsError(fmt.Sprintf("命名空间已存在: %s", namespace.Name))
	}

	// 设置创建和更新时间
	now := time.Now()
	if namespace.CreatedAt.IsZero() {
		namespace.CreatedAt = now
	}
	if namespace.UpdatedAt.IsZero() {
		namespace.UpdatedAt = now
	}

	// 序列化命名空间数据
	data, err := json.Marshal(namespace)
	if err != nil {
		return storage.NewInternalError(fmt.Sprintf("序列化命名空间数据失败: %v", err))
	}

	// 存储到etcd
	_, err = s.client.GetClient().Put(ctx, key, string(data))
	if err != nil {
		return storage.NewInternalError(fmt.Sprintf("写入etcd失败: %v", err))
	}

	return nil
}

// DeleteNamespace 删除命名空间
func (s *NamespaceStorage) DeleteNamespace(ctx context.Context, name string) error {
	if name == "" {
		return storage.NewInvalidArgumentError("命名空间名称不能为空")
	}

	// 检查命名空间是否存在
	namespace, err := s.GetNamespace(ctx, name)
	if err != nil {
		return err
	}

	// 检查命名空间是否为空（不含服务）
	if namespace.ServiceCount > 0 {
		return storage.NewNamespaceNotEmptyError(fmt.Sprintf("命名空间非空，包含 %d 个服务: %s", namespace.ServiceCount, name))
	}

	// 从etcd删除命名空间
	key := s.client.GetNamespaceKey(name)
	resp, err := s.client.GetClient().Delete(ctx, key)
	if err != nil {
		return storage.NewInternalError(fmt.Sprintf("从etcd删除失败: %v", err))
	}

	if resp.Deleted == 0 {
		return storage.NewNotFoundError(fmt.Sprintf("命名空间不存在: %s", name))
	}

	return nil
}

// GetNamespace 获取命名空间详情
func (s *NamespaceStorage) GetNamespace(ctx context.Context, name string) (*storage.Namespace, error) {
	if name == "" {
		return nil, storage.NewInvalidArgumentError("命名空间名称不能为空")
	}

	key := s.client.GetNamespaceKey(name)
	resp, err := s.client.GetClient().Get(ctx, key)
	if err != nil {
		return nil, storage.NewInternalError(fmt.Sprintf("从etcd读取失败: %v", err))
	}

	if len(resp.Kvs) == 0 {
		return nil, storage.NewNotFoundError(fmt.Sprintf("命名空间不存在: %s", name))
	}

	var namespace storage.Namespace
	if err := json.Unmarshal(resp.Kvs[0].Value, &namespace); err != nil {
		return nil, storage.NewInternalError(fmt.Sprintf("解析命名空间数据失败: %v", err))
	}

	return &namespace, nil
}

// ListNamespaces 获取所有命名空间列表
func (s *NamespaceStorage) ListNamespaces(ctx context.Context) ([]*storage.Namespace, error) {
	prefix := s.client.GetNamespacesPrefix()
	resp, err := s.client.GetClient().Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, storage.NewInternalError(fmt.Sprintf("从etcd读取失败: %v", err))
	}

	namespaces := make([]*storage.Namespace, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var namespace storage.Namespace
		if err := json.Unmarshal(kv.Value, &namespace); err != nil {
			// 忽略无法解析的数据，继续处理其他数据
			continue
		}
		namespaces = append(namespaces, &namespace)
	}

	return namespaces, nil
}

// UpdateNamespaceServiceCount 更新命名空间服务数量
func (s *NamespaceStorage) UpdateNamespaceServiceCount(ctx context.Context, name string, delta int) error {
	if name == "" {
		return storage.NewInvalidArgumentError("命名空间名称不能为空")
	}

	// 获取命名空间
	namespace, err := s.GetNamespace(ctx, name)
	if err != nil {
		return err
	}

	// 更新服务数量
	namespace.ServiceCount += delta
	if namespace.ServiceCount < 0 {
		namespace.ServiceCount = 0
	}
	namespace.UpdatedAt = time.Now()

	// 序列化命名空间数据
	data, err := json.Marshal(namespace)
	if err != nil {
		return storage.NewInternalError(fmt.Sprintf("序列化命名空间数据失败: %v", err))
	}

	// 更新到etcd
	key := s.client.GetNamespaceKey(name)
	_, err = s.client.GetClient().Put(ctx, key, string(data))
	if err != nil {
		return storage.NewInternalError(fmt.Sprintf("写入etcd失败: %v", err))
	}

	return nil
}
