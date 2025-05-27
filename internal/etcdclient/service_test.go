package etcdclient

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServiceToDNSRecords 测试将服务实例转换为DNS记录的功能
func TestServiceToDNSRecords(t *testing.T) {
	// 创建一个模拟的EtcdClient
	client := &mockEtcdClient{
		services: map[string][]*ServiceInstance{
			"nginx": {
				{
					ServiceName: "nginx",
					InstanceID:  "instance-1",
					IPAddress:   "192.168.1.100",
					Port:        8080,
					TTL:         60,
				},
				{
					ServiceName: "nginx",
					InstanceID:  "instance-2",
					IPAddress:   "192.168.1.101",
					Port:        8080,
					TTL:         60,
				},
			},
		},
	}

	// 测试转换服务到DNS记录
	ctx := context.Background()
	records, err := client.ServiceToDNSRecords(ctx, "nginx.default.svc.cluster.local")
	require.NoError(t, err)
	require.NotNil(t, records)

	// 验证A记录
	aRecord, ok := records["A"]
	require.True(t, ok, "应该存在A记录")
	assert.Equal(t, "A", aRecord.Type)
	assert.Equal(t, "192.168.1.100", aRecord.Value) // 应该使用第一个实例的IP

	// 验证SRV记录
	srv1, ok := records["SRV-0"]
	require.True(t, ok, "应该存在SRV-0记录")
	assert.Equal(t, "SRV", srv1.Type)
	assert.Contains(t, srv1.Value, "8080") // 端口号应该包含在SRV记录中

	srv2, ok := records["SRV-1"]
	require.True(t, ok, "应该存在SRV-1记录")
	assert.Equal(t, "SRV", srv2.Type)
	assert.Contains(t, srv2.Value, "8080") // 端口号应该包含在SRV记录中
}

// mockEtcdClient 模拟etcd客户端，用于测试
type mockEtcdClient struct {
	services map[string][]*ServiceInstance
}

func (m *mockEtcdClient) Connect() error                                      { return nil }
func (m *mockEtcdClient) Close() error                                        { return nil }
func (m *mockEtcdClient) Ping(ctx context.Context) error                      { return nil }
func (m *mockEtcdClient) Get(ctx context.Context, key string) (string, error) { return "", nil }
func (m *mockEtcdClient) GetWithPrefix(ctx context.Context, prefix string) (map[string]string, error) {
	return nil, nil
}
func (m *mockEtcdClient) GetDNSRecord(ctx context.Context, domain string, recordType string) (*DNSRecord, error) {
	return nil, nil
}
func (m *mockEtcdClient) PutDNSRecord(ctx context.Context, domain string, record *DNSRecord) error {
	return nil
}
func (m *mockEtcdClient) GetDNSRecordsForDomain(ctx context.Context, domain string) (map[string]*DNSRecord, error) {
	return nil, nil
}

// RegisterService 模拟服务注册
func (m *mockEtcdClient) RegisterService(ctx context.Context, instance *ServiceInstance) error {
	if m.services == nil {
		m.services = make(map[string][]*ServiceInstance)
	}
	m.services[instance.ServiceName] = append(m.services[instance.ServiceName], instance)
	return nil
}

// DeregisterService 模拟服务注销
func (m *mockEtcdClient) DeregisterService(ctx context.Context, serviceName, instanceID string) error {
	instances, ok := m.services[serviceName]
	if !ok {
		return nil
	}

	newInstances := make([]*ServiceInstance, 0, len(instances))
	for _, inst := range instances {
		if inst.InstanceID != instanceID {
			newInstances = append(newInstances, inst)
		}
	}
	m.services[serviceName] = newInstances
	return nil
}

// GetServiceInstances 模拟获取服务实例
func (m *mockEtcdClient) GetServiceInstances(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	instances, ok := m.services[serviceName]
	if !ok {
		return nil, nil
	}
	return instances, nil
}

// ServiceToDNSRecords 实现从服务实例到DNS记录的转换
func (m *mockEtcdClient) ServiceToDNSRecords(ctx context.Context, domain string) (map[string]*DNSRecord, error) {
	// 提取服务名（假设domain格式为service.namespace.svc.cluster.local）
	parts := strings.Split(domain, ".")
	if len(parts) < 1 {
		return nil, fmt.Errorf("无效的域名格式: %s", domain)
	}

	serviceName := parts[0]

	// 获取服务实例
	instances, err := m.GetServiceInstances(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("获取服务实例失败: %w", err)
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("未找到服务实例: %s", serviceName)
	}

	// 创建DNS记录
	records := make(map[string]*DNSRecord)

	// A记录 - 使用第一个实例的IP
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

// RefreshServiceLease 模拟刷新服务租约
func (m *mockEtcdClient) RefreshServiceLease(ctx context.Context, serviceName, instanceID string, ttl int) error {
	instances, ok := m.services[serviceName]
	if !ok {
		return fmt.Errorf("服务不存在: %s", serviceName)
	}

	var found bool
	for i, instance := range instances {
		if instance.InstanceID == instanceID {
			// 如果提供了TTL，则更新TTL
			if ttl > 0 {
				instances[i].TTL = ttl
			}
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("实例不存在: %s/%s", serviceName, instanceID)
	}

	return nil
}
