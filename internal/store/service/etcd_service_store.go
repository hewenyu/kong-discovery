package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/hewenyu/kong-discovery/internal/core/model"
	"github.com/hewenyu/kong-discovery/internal/store/etcd"
)

const (
	// 服务存储的前缀
	servicePrefix = "/services/"
	// 服务名称索引的前缀
	serviceNameIndexPrefix = "/service-names/"
)

// EtcdServiceStore 实现基于etcd的服务存储
type EtcdServiceStore struct {
	client    *etcd.Client
	namespace string // 默认命名空间
}

// NewEtcdServiceStore 创建一个新的基于etcd的服务存储
func NewEtcdServiceStore(client *etcd.Client, defaultNamespace string) *EtcdServiceStore {
	return &EtcdServiceStore{
		client:    client,
		namespace: defaultNamespace,
	}
}

// getServiceKey 获取服务的存储键
func getServiceKey(serviceID string) string {
	return servicePrefix + serviceID
}

// getServiceNameIndexKey 获取服务名称索引的存储键
func getServiceNameIndexKey(name, namespace string) string {
	return fmt.Sprintf("%s%s/%s", serviceNameIndexPrefix, namespace, name)
}

// Register 注册服务
func (s *EtcdServiceStore) Register(ctx context.Context, service *model.Service) error {
	// 确保服务有ID
	if service.ID == "" {
		service.ID = uuid.New().String()
	}

	// 确保服务有命名空间
	if service.Namespace == "" {
		service.Namespace = s.namespace
	}

	// 设置注册时间和心跳时间
	now := time.Now()
	service.RegisteredAt = now
	service.LastHeartbeat = now
	service.Health = model.HealthStatusHealthy

	// 序列化服务信息
	data, err := json.Marshal(service)
	if err != nil {
		return fmt.Errorf("序列化服务信息失败: %w", err)
	}

	// 存储服务信息
	serviceKey := getServiceKey(service.ID)
	if service.TTL > 0 {
		err = s.client.PutWithLease(ctx, serviceKey, data, service.TTL)
	} else {
		err = s.client.Put(ctx, serviceKey, data)
	}
	if err != nil {
		return fmt.Errorf("存储服务信息失败: %w", err)
	}

	// 存储服务名称索引
	nameIndexKey := getServiceNameIndexKey(service.Name, service.Namespace)
	serviceIDsData, err := s.client.Get(ctx, nameIndexKey)

	var serviceIDs []string
	if err != nil {
		return fmt.Errorf("获取服务名称索引失败: %w", err)
	}

	if serviceIDsData != nil {
		if err := json.Unmarshal(serviceIDsData, &serviceIDs); err != nil {
			return fmt.Errorf("解析服务名称索引失败: %w", err)
		}
	}

	// 检查服务ID是否已存在
	exists := false
	for _, id := range serviceIDs {
		if id == service.ID {
			exists = true
			break
		}
	}

	// 如果不存在，添加服务ID到索引
	if !exists {
		serviceIDs = append(serviceIDs, service.ID)
		serviceIDsData, err = json.Marshal(serviceIDs)
		if err != nil {
			return fmt.Errorf("序列化服务名称索引失败: %w", err)
		}

		if err := s.client.Put(ctx, nameIndexKey, serviceIDsData); err != nil {
			return fmt.Errorf("存储服务名称索引失败: %w", err)
		}
	}

	return nil
}

// Deregister 注销服务
func (s *EtcdServiceStore) Deregister(ctx context.Context, serviceID string) error {
	// 获取服务信息
	service, err := s.GetService(ctx, serviceID)
	if err != nil {
		return fmt.Errorf("获取服务信息失败: %w", err)
	}

	if service == nil {
		return fmt.Errorf("服务不存在: %s", serviceID)
	}

	// 删除服务信息
	serviceKey := getServiceKey(serviceID)
	if err := s.client.Delete(ctx, serviceKey); err != nil {
		return fmt.Errorf("删除服务信息失败: %w", err)
	}

	// 更新服务名称索引
	nameIndexKey := getServiceNameIndexKey(service.Name, service.Namespace)
	serviceIDsData, err := s.client.Get(ctx, nameIndexKey)
	if err != nil {
		return fmt.Errorf("获取服务名称索引失败: %w", err)
	}

	if serviceIDsData == nil {
		// 索引不存在，无需更新
		return nil
	}

	var serviceIDs []string
	if err := json.Unmarshal(serviceIDsData, &serviceIDs); err != nil {
		return fmt.Errorf("解析服务名称索引失败: %w", err)
	}

	// 从索引中移除服务ID
	var newServiceIDs []string
	for _, id := range serviceIDs {
		if id != serviceID {
			newServiceIDs = append(newServiceIDs, id)
		}
	}

	// 如果索引为空，删除索引
	if len(newServiceIDs) == 0 {
		if err := s.client.Delete(ctx, nameIndexKey); err != nil {
			return fmt.Errorf("删除服务名称索引失败: %w", err)
		}
	} else {
		// 更新索引
		newServiceIDsData, err := json.Marshal(newServiceIDs)
		if err != nil {
			return fmt.Errorf("序列化服务名称索引失败: %w", err)
		}

		if err := s.client.Put(ctx, nameIndexKey, newServiceIDsData); err != nil {
			return fmt.Errorf("更新服务名称索引失败: %w", err)
		}
	}

	return nil
}

// UpdateHeartbeat 更新服务心跳
func (s *EtcdServiceStore) UpdateHeartbeat(ctx context.Context, serviceID string) error {
	// 获取服务信息
	service, err := s.GetService(ctx, serviceID)
	if err != nil {
		return fmt.Errorf("获取服务信息失败: %w", err)
	}

	if service == nil {
		return fmt.Errorf("服务不存在: %s", serviceID)
	}

	// 更新心跳时间
	service.LastHeartbeat = time.Now()
	service.Health = model.HealthStatusHealthy

	// 序列化服务信息
	data, err := json.Marshal(service)
	if err != nil {
		return fmt.Errorf("序列化服务信息失败: %w", err)
	}

	// 存储服务信息，更新lease
	serviceKey := getServiceKey(serviceID)
	if service.TTL > 0 {
		// 使用PutWithLease重新创建lease并更新服务信息
		// 这会使TTL从当前时间重新计时
		err = s.client.PutWithLease(ctx, serviceKey, data, service.TTL)
	} else {
		err = s.client.Put(ctx, serviceKey, data)
	}
	if err != nil {
		return fmt.Errorf("存储服务信息失败: %w", err)
	}

	return nil
}

// GetService 获取服务信息
func (s *EtcdServiceStore) GetService(ctx context.Context, serviceID string) (*model.Service, error) {
	serviceKey := getServiceKey(serviceID)
	data, err := s.client.Get(ctx, serviceKey)
	if err != nil {
		return nil, fmt.Errorf("获取服务信息失败: %w", err)
	}

	if data == nil {
		return nil, nil // 服务不存在
	}

	var service model.Service
	if err := json.Unmarshal(data, &service); err != nil {
		return nil, fmt.Errorf("解析服务信息失败: %w", err)
	}

	return &service, nil
}

// GetServiceByName 根据服务名和命名空间获取服务列表
func (s *EtcdServiceStore) GetServiceByName(ctx context.Context, name, namespace string) ([]*model.Service, error) {
	if namespace == "" {
		namespace = s.namespace
	}

	// 获取服务名称索引
	nameIndexKey := getServiceNameIndexKey(name, namespace)
	serviceIDsData, err := s.client.Get(ctx, nameIndexKey)
	if err != nil {
		return nil, fmt.Errorf("获取服务名称索引失败: %w", err)
	}

	if serviceIDsData == nil {
		return []*model.Service{}, nil // 没有对应的服务
	}

	var serviceIDs []string
	if err := json.Unmarshal(serviceIDsData, &serviceIDs); err != nil {
		return nil, fmt.Errorf("解析服务名称索引失败: %w", err)
	}

	// 获取服务信息
	services := make([]*model.Service, 0, len(serviceIDs))
	for _, id := range serviceIDs {
		service, err := s.GetService(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("获取服务信息失败: %w", err)
		}

		// 服务可能已经被删除
		if service != nil {
			services = append(services, service)
		}
	}

	return services, nil
}

// ListServices 获取服务列表
func (s *EtcdServiceStore) ListServices(ctx context.Context, namespace string) ([]*model.Service, error) {
	if namespace == "" {
		namespace = s.namespace
	}

	// 获取服务名称索引
	nameIndexPrefix := serviceNameIndexPrefix + namespace + "/"
	serviceNameIndices, err := s.client.GetWithPrefix(ctx, nameIndexPrefix)
	if err != nil {
		return nil, fmt.Errorf("获取服务名称索引失败: %w", err)
	}

	// 获取所有服务ID
	var allServiceIDs []string
	for _, data := range serviceNameIndices {
		var serviceIDs []string
		if err := json.Unmarshal(data, &serviceIDs); err != nil {
			return nil, fmt.Errorf("解析服务名称索引失败: %w", err)
		}
		allServiceIDs = append(allServiceIDs, serviceIDs...)
	}

	// 获取服务信息
	services := make([]*model.Service, 0, len(allServiceIDs))
	for _, id := range allServiceIDs {
		service, err := s.GetService(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("获取服务信息失败: %w", err)
		}

		// 服务可能已经被删除
		if service != nil {
			services = append(services, service)
		}
	}

	return services, nil
}

// ListAllServices 获取所有服务列表
func (s *EtcdServiceStore) ListAllServices(ctx context.Context) ([]*model.Service, error) {
	// 直接获取所有服务信息
	serviceData, err := s.client.GetWithPrefix(ctx, servicePrefix)
	if err != nil {
		return nil, fmt.Errorf("获取服务信息失败: %w", err)
	}

	services := make([]*model.Service, 0, len(serviceData))
	for _, data := range serviceData {
		var service model.Service
		if err := json.Unmarshal(data, &service); err != nil {
			return nil, fmt.Errorf("解析服务信息失败: %w", err)
		}
		services = append(services, &service)
	}

	return services, nil
}

// CleanupStaleServices 清理过期服务
// 注意：etcd会根据TTL自动清理过期服务，本方法主要作为备份机制
func (s *EtcdServiceStore) CleanupStaleServices(ctx context.Context, before time.Time) (int, error) {
	// 获取所有服务信息
	services, err := s.ListAllServices(ctx)
	if err != nil {
		return 0, fmt.Errorf("获取服务列表失败: %w", err)
	}

	// 找出过期的服务
	staleServices := make([]*model.Service, 0)
	for _, service := range services {
		// 添加日志记录
		log.Printf("检查服务 %s (ID: %s) 的心跳时间: %v, 心跳阈值时间: %v, 是否过期: %v",
			service.Name, service.ID, service.LastHeartbeat, before, service.LastHeartbeat.Before(before))

		// 如果这个服务还存在于etcd中，说明etcd的lease机制没有自动清理它
		// 检查它的心跳时间是否过期，如果过期，手动清理
		if service.LastHeartbeat.Before(before) {
			staleServices = append(staleServices, service)
		}
	}

	log.Printf("找到 %d 个过期服务", len(staleServices))

	// 删除过期的服务
	deletedCount := 0
	for _, service := range staleServices {
		log.Printf("尝试删除过期服务: %s (ID: %s)", service.Name, service.ID)
		if err := s.Deregister(ctx, service.ID); err != nil {
			log.Printf("删除过期服务 %s (ID: %s) 失败: %v", service.Name, service.ID, err)
			continue
		}
		deletedCount++
	}

	return deletedCount, nil
}
