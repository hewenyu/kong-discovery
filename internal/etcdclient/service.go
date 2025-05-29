package etcdclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

// ServiceInstance 表示一个服务实例
type ServiceInstance struct {
	ServiceName   string            `json:"service_name"`       // 服务名称
	InstanceID    string            `json:"instance_id"`        // 实例ID（UUID）
	IPAddress     string            `json:"ip_address"`         // IP地址
	Port          int               `json:"port"`               // 端口
	Metadata      map[string]string `json:"metadata,omitempty"` // 可选元数据（版本、区域等）
	TTL           int               `json:"ttl"`                // 租约TTL（秒）
	LastHeartbeat string            `json:"last_heartbeat"`     // 最后心跳时间
}

// RegisterService 将服务实例注册到etcd
func (e *EtcdClient) RegisterService(ctx context.Context, instance *ServiceInstance) error {
	if e.client == nil {
		return fmt.Errorf("etcd客户端未连接")
	}

	// 设置初始心跳时间
	instance.LastHeartbeat = time.Now().Format(time.RFC3339)

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

	// 自动创建或更新相关的DNS记录
	if err := e.createOrUpdateServiceDNSRecords(ctx, instance); err != nil {
		e.logger.Warn("创建服务DNS记录失败，服务注册成功但DNS记录可能不完整",
			zap.String("service", instance.ServiceName),
			zap.String("id", instance.InstanceID),
			zap.Error(err))
		// 我们不因为DNS记录创建失败而阻止服务注册
	}

	return nil
}

// DeregisterService 从etcd注销服务实例
func (e *EtcdClient) DeregisterService(ctx context.Context, serviceName, instanceID string) error {
	if e.client == nil {
		return fmt.Errorf("etcd客户端未连接")
	}

	// 在删除服务前，先获取服务实例数据
	serviceInstances, err := e.GetServiceInstances(ctx, serviceName)
	if err != nil {
		e.logger.Warn("获取服务实例列表失败，无法检查是否需要清理DNS记录",
			zap.String("service", serviceName),
			zap.Error(err))
		// 继续执行删除操作
	}

	// 找到要删除的实例
	var targetInstance *ServiceInstance
	var remainingCount int
	for _, instance := range serviceInstances {
		if instance.InstanceID == instanceID {
			targetInstance = instance
		} else {
			remainingCount++
		}
	}

	// 生成服务实例键
	key := getServiceInstanceKey(serviceName, instanceID)

	ctx, cancel := context.WithTimeout(ctx, etcdTimeout)
	defer cancel()

	// 删除键
	_, err = e.client.Delete(ctx, key)
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

	// 如果没有剩余实例，则清理DNS记录
	if remainingCount == 0 && targetInstance != nil {
		if err := e.cleanupServiceDNSRecords(ctx, targetInstance); err != nil {
			e.logger.Warn("清理服务DNS记录失败，服务已注销但DNS记录可能仍然存在",
				zap.String("service", serviceName),
				zap.String("id", instanceID),
				zap.Error(err))
			// 我们不因为DNS记录清理失败而阻止服务注销
		}
	}

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

	// 更新最后心跳时间
	instance.LastHeartbeat = time.Now().Format(time.RFC3339)

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

// IsServiceExpired 检查服务实例是否已过期
func IsServiceExpired(instance *ServiceInstance, maxHeartbeatAge time.Duration) bool {
	if instance == nil || instance.LastHeartbeat == "" {
		return true // 没有心跳记录的服务视为过期
	}

	lastHeartbeat, err := time.Parse(time.RFC3339, instance.LastHeartbeat)
	if err != nil {
		return true // 无法解析心跳时间的服务视为过期
	}

	// 检查心跳是否超过最大允许时间
	return time.Since(lastHeartbeat) > maxHeartbeatAge
}

// StartCleanupExpiredServices 启动过期服务清理定时任务
func (e *EtcdClient) StartCleanupExpiredServices(ctx context.Context, interval time.Duration, maxHeartbeatAge time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				e.cleanupExpiredServices(ctx, maxHeartbeatAge)
			}
		}
	}()
}

// cleanupExpiredServices 清理过期的服务实例
func (e *EtcdClient) cleanupExpiredServices(ctx context.Context, maxHeartbeatAge time.Duration) {
	e.logger.Info("开始清理过期服务实例", zap.Duration("max_age", maxHeartbeatAge))

	// 获取所有服务名称
	serviceNames, err := e.GetAllServiceNames(ctx)
	if err != nil {
		e.logger.Error("获取服务列表失败", zap.Error(err))
		return
	}

	for _, serviceName := range serviceNames {
		// 获取该服务的所有实例
		instances, err := e.GetServiceInstances(ctx, serviceName)
		if err != nil {
			e.logger.Error("获取服务实例失败",
				zap.String("service", serviceName),
				zap.Error(err))
			continue
		}

		for _, instance := range instances {
			if IsServiceExpired(instance, maxHeartbeatAge) {
				e.logger.Info("检测到过期服务实例",
					zap.String("service", instance.ServiceName),
					zap.String("id", instance.InstanceID),
					zap.String("last_heartbeat", instance.LastHeartbeat))

				// 尝试注销过期的服务
				deregisterCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := e.DeregisterService(deregisterCtx, instance.ServiceName, instance.InstanceID)
				cancel()

				if err != nil {
					e.logger.Error("注销过期服务失败",
						zap.String("service", instance.ServiceName),
						zap.String("id", instance.InstanceID),
						zap.Error(err))
				} else {
					e.logger.Info("已自动注销过期服务",
						zap.String("service", instance.ServiceName),
						zap.String("id", instance.InstanceID))
				}
			}
		}
	}
}

// createOrUpdateServiceDNSRecords 为服务创建或更新DNS记录
func (e *EtcdClient) createOrUpdateServiceDNSRecords(ctx context.Context, instance *ServiceInstance) error {
	// 确定服务域名
	var domain string
	// 尝试从元数据中获取域名
	if instance.Metadata != nil {
		if d, ok := instance.Metadata["domain"]; ok && d != "" {
			domain = d
		}
	}

	// 如果元数据中没有域名，使用默认命名规则
	if domain == "" {
		domain = fmt.Sprintf("%s.default.svc.cluster.local", instance.ServiceName)
	}

	// 判断是否需要创建A记录
	// 获取当前实例列表
	instances, err := e.GetServiceInstances(ctx, instance.ServiceName)
	if err != nil {
		return fmt.Errorf("获取服务实例列表失败: %w", err)
	}

	// 检查是否是第一个实例
	isFirstInstance := len(instances) <= 1

	// 如果是第一个实例或唯一的实例，创建A记录
	if isFirstInstance {
		// 创建A记录
		aRecord := &DNSRecord{
			Type:  "A",
			Value: instance.IPAddress,
			TTL:   60,
		}

		if err := e.PutDNSRecord(ctx, domain, aRecord); err != nil {
			return fmt.Errorf("创建A记录失败: %w", err)
		}

		e.logger.Info("为服务创建A记录",
			zap.String("service", instance.ServiceName),
			zap.String("domain", domain),
			zap.String("ip", instance.IPAddress))
	}

	// 创建SRV记录
	// 确保目标域名以点号结尾，符合DNS格式规范
	targetDomain := domain
	if !strings.HasSuffix(targetDomain, ".") {
		targetDomain = targetDomain + "."
	}

	// SRV记录域名
	srvDomain := fmt.Sprintf("_%s._tcp.default.svc.cluster.local", instance.ServiceName)

	// SRV记录值，格式为: "priority weight port target"
	srvValue := fmt.Sprintf("10 10 %d %s", instance.Port, targetDomain)

	srvRecord := &DNSRecord{
		Type:  "SRV",
		Value: srvValue,
		TTL:   60,
	}

	if err := e.PutDNSRecord(ctx, srvDomain, srvRecord); err != nil {
		return fmt.Errorf("创建SRV记录失败: %w", err)
	}

	e.logger.Info("为服务创建SRV记录",
		zap.String("service", instance.ServiceName),
		zap.String("domain", srvDomain),
		zap.String("value", srvValue))

	return nil
}

// cleanupServiceDNSRecords 清理服务的DNS记录
func (e *EtcdClient) cleanupServiceDNSRecords(ctx context.Context, instance *ServiceInstance) error {
	// 确定服务域名
	var domain string
	// 尝试从元数据中获取域名
	if instance.Metadata != nil {
		if d, ok := instance.Metadata["domain"]; ok && d != "" {
			domain = d
		}
	}

	// 如果元数据中没有域名，使用默认命名规则
	if domain == "" {
		domain = fmt.Sprintf("%s.default.svc.cluster.local", instance.ServiceName)
	}

	var errors []error

	// 删除A记录
	if err := e.DeleteDNSRecord(ctx, domain, "A"); err != nil {
		errors = append(errors, fmt.Errorf("删除A记录失败: %w", err))
	} else {
		e.logger.Info("删除服务A记录",
			zap.String("service", instance.ServiceName),
			zap.String("domain", domain))
	}

	// 删除SRV记录
	srvDomain := fmt.Sprintf("_%s._tcp.default.svc.cluster.local", instance.ServiceName)
	if err := e.DeleteDNSRecord(ctx, srvDomain, "SRV"); err != nil {
		errors = append(errors, fmt.Errorf("删除SRV记录失败: %w", err))
	} else {
		e.logger.Info("删除服务SRV记录",
			zap.String("service", instance.ServiceName),
			zap.String("domain", srvDomain))
	}

	// 如果有错误，返回组合的错误信息
	if len(errors) > 0 {
		errMsg := "清理DNS记录时发生错误: "
		for i, err := range errors {
			if i > 0 {
				errMsg += "; "
			}
			errMsg += err.Error()
		}
		return fmt.Errorf(errMsg)
	}

	return nil
}
