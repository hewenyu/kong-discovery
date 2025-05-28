package etcdclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

// ServiceInstance 表示一个服务实例
type ServiceInstance struct {
	ServiceName string            `json:"service_name"`       // 服务名称
	InstanceID  string            `json:"instance_id"`        // 实例ID（UUID）
	IPAddress   string            `json:"ip_address"`         // IP地址
	Port        int               `json:"port"`               // 端口
	Metadata    map[string]string `json:"metadata,omitempty"` // 可选元数据（版本、区域等）
	TTL         int               `json:"ttl"`                // 租约TTL（秒）
}

// RegisterService 将服务实例注册到etcd
func (e *EtcdClient) RegisterService(ctx context.Context, instance *ServiceInstance) error {
	if e.client == nil {
		return fmt.Errorf("etcd客户端未连接")
	}

	// 生成服务实例键
	key := getServiceInstanceKey(instance.ServiceName, instance.InstanceID)

	// 序列化服务实例
	data, err := json.Marshal(instance)
	if err != nil {
		e.logger.Error("序列化服务实例失败",
			zap.String("service", instance.ServiceName),
			zap.String("id", instance.InstanceID),
			zap.Error(err))
		return fmt.Errorf("序列化服务实例失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, etcdTimeout)
	defer cancel()

	// 创建租约
	lease, err := e.client.Grant(ctx, int64(instance.TTL))
	if err != nil {
		e.logger.Error("创建etcd租约失败", zap.Error(err))
		return fmt.Errorf("创建etcd租约失败: %w", err)
	}

	// 写入带租约的键值
	_, err = e.client.Put(ctx, key, string(data), clientv3.WithLease(lease.ID))
	if err != nil {
		e.logger.Error("注册服务实例失败", zap.Error(err))
		return fmt.Errorf("注册服务实例失败: %w", err)
	}

	e.logger.Info("服务实例注册成功",
		zap.String("service", instance.ServiceName),
		zap.String("id", instance.InstanceID),
		zap.String("ip", instance.IPAddress),
		zap.Int("port", instance.Port))

	return nil
}

// DeregisterService 从etcd注销服务实例
func (e *EtcdClient) DeregisterService(ctx context.Context, serviceName, instanceID string) error {
	if e.client == nil {
		return fmt.Errorf("etcd客户端未连接")
	}

	// 生成服务实例键
	key := getServiceInstanceKey(serviceName, instanceID)

	ctx, cancel := context.WithTimeout(ctx, etcdTimeout)
	defer cancel()

	// 删除键
	_, err := e.client.Delete(ctx, key)
	if err != nil {
		e.logger.Error("注销服务实例失败",
			zap.String("service", serviceName),
			zap.String("id", instanceID),
			zap.Error(err))
		return fmt.Errorf("注销服务实例失败: %w", err)
	}

	e.logger.Info("服务实例注销成功",
		zap.String("service", serviceName),
		zap.String("id", instanceID))

	return nil
}

// GetServiceInstances 获取指定服务的所有实例
func (e *EtcdClient) GetServiceInstances(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	if e.client == nil {
		return nil, fmt.Errorf("etcd客户端未连接")
	}

	// 生成服务前缀
	prefix := getServicePrefix(serviceName)

	ctx, cancel := context.WithTimeout(ctx, etcdTimeout)
	defer cancel()

	// 查询前缀
	resp, err := e.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		e.logger.Error("获取服务实例列表失败",
			zap.String("service", serviceName),
			zap.Error(err))
		return nil, fmt.Errorf("获取服务实例列表失败: %w", err)
	}

	instances := make([]*ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var instance ServiceInstance
		if err := json.Unmarshal(kv.Value, &instance); err != nil {
			e.logger.Warn("解析服务实例数据失败",
				zap.String("key", string(kv.Key)),
				zap.Error(err))
			continue
		}
		instances = append(instances, &instance)
	}

	return instances, nil
}

// ServiceToDNSRecords 将服务实例转换为DNS记录
func (e *EtcdClient) ServiceToDNSRecords(ctx context.Context, domain string) (map[string]*DNSRecord, error) {
	// 提取服务名（假设domain格式为service.namespace.svc.cluster.local）
	parts := strings.Split(domain, ".")
	if len(parts) < 1 {
		return nil, fmt.Errorf("无效的域名格式: %s", domain)
	}

	serviceName := parts[0]

	// 获取服务实例
	instances, err := e.GetServiceInstances(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("获取服务实例失败: %w", err)
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("未找到服务实例: %s", serviceName)
	}

	// 创建DNS记录
	records := make(map[string]*DNSRecord)

	// A记录 - 使用第一个实例的IP（简单负载均衡可以在DNS层之上实现）
	records["A"] = &DNSRecord{
		Type:  "A",
		Value: instances[0].IPAddress,
		TTL:   60,
	}

	// SRV记录 - 列出所有实例的IP:Port
	for i, instance := range instances {
		// SRV记录格式：priority weight port target
		srvValue := fmt.Sprintf("10 10 %d %s.%s", instance.Port, instance.InstanceID, domain)
		records[fmt.Sprintf("SRV-%d", i)] = &DNSRecord{
			Type:  "SRV",
			Value: srvValue,
			TTL:   60,
		}
	}

	return records, nil
}

// RefreshServiceLease 刷新服务实例的租约
func (e *EtcdClient) RefreshServiceLease(ctx context.Context, serviceName, instanceID string, ttl int) error {
	if e.client == nil {
		return fmt.Errorf("etcd客户端未连接")
	}

	// 生成服务实例键
	key := getServiceInstanceKey(serviceName, instanceID)

	// 首先获取当前服务实例数据
	ctx, cancel := context.WithTimeout(ctx, etcdTimeout)
	defer cancel()

	resp, err := e.client.Get(ctx, key)
	if err != nil {
		e.logger.Error("获取服务实例数据失败",
			zap.String("service", serviceName),
			zap.String("id", instanceID),
			zap.Error(err))
		return fmt.Errorf("获取服务实例数据失败: %w", err)
	}

	if len(resp.Kvs) == 0 {
		e.logger.Warn("服务实例不存在，无法刷新租约",
			zap.String("service", serviceName),
			zap.String("id", instanceID))
		return fmt.Errorf("服务实例不存在: %s/%s", serviceName, instanceID)
	}

	// 解析服务实例数据
	var instance ServiceInstance
	if err := json.Unmarshal(resp.Kvs[0].Value, &instance); err != nil {
		e.logger.Error("解析服务实例数据失败",
			zap.String("service", serviceName),
			zap.String("id", instanceID),
			zap.Error(err))
		return fmt.Errorf("解析服务实例数据失败: %w", err)
	}

	// 如果提供了TTL，则更新实例的TTL
	if ttl > 0 {
		instance.TTL = ttl
	}

	// 创建新的租约
	lease, err := e.client.Grant(ctx, int64(instance.TTL))
	if err != nil {
		e.logger.Error("创建etcd租约失败", zap.Error(err))
		return fmt.Errorf("创建etcd租约失败: %w", err)
	}

	// 序列化更新后的服务实例
	data, err := json.Marshal(&instance)
	if err != nil {
		e.logger.Error("序列化服务实例失败",
			zap.String("service", serviceName),
			zap.String("id", instanceID),
			zap.Error(err))
		return fmt.Errorf("序列化服务实例失败: %w", err)
	}

	// 使用新租约写入服务实例数据
	_, err = e.client.Put(ctx, key, string(data), clientv3.WithLease(lease.ID))
	if err != nil {
		e.logger.Error("刷新服务实例租约失败",
			zap.String("service", serviceName),
			zap.String("id", instanceID),
			zap.Error(err))
		return fmt.Errorf("刷新服务实例租约失败: %w", err)
	}

	e.logger.Info("服务实例租约刷新成功",
		zap.String("service", serviceName),
		zap.String("id", instanceID),
		zap.Int("ttl", instance.TTL))

	return nil
}

// getServiceInstanceKey 生成服务实例在etcd中的键
func getServiceInstanceKey(serviceName, instanceID string) string {
	return fmt.Sprintf("/services/%s/%s", serviceName, instanceID)
}

// getServicePrefix 生成服务在etcd中的键前缀
func getServicePrefix(serviceName string) string {
	return fmt.Sprintf("/services/%s/", serviceName)
}

// GetAllServiceNames 获取所有已注册服务的名称列表
func (e *EtcdClient) GetAllServiceNames(ctx context.Context) ([]string, error) {
	if e.client == nil {
		return nil, fmt.Errorf("etcd客户端未连接")
	}

	// 服务根路径前缀
	servicesPrefix := "/services/"

	ctx, cancel := context.WithTimeout(ctx, etcdTimeout)
	defer cancel()

	// 获取所有服务键
	resp, err := e.client.Get(ctx, servicesPrefix, clientv3.WithPrefix(), clientv3.WithKeysOnly())
	if err != nil {
		e.logger.Error("获取服务列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取服务列表失败: %w", err)
	}

	// 使用map去重服务名称
	serviceMap := make(map[string]struct{})
	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		// 从键中提取服务名
		// 格式: /services/服务名/实例ID
		parts := strings.Split(key, "/")
		if len(parts) >= 3 {
			serviceName := parts[2]
			serviceMap[serviceName] = struct{}{}
		}
	}

	// 转换为字符串切片
	serviceNames := make([]string, 0, len(serviceMap))
	for serviceName := range serviceMap {
		serviceNames = append(serviceNames, serviceName)
	}

	return serviceNames, nil
}
