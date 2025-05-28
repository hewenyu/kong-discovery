package etcdclient

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEtcdClient_Connect(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建配置和日志记录器
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建etcd客户端
	client := NewEtcdClient(cfg, logger)

	// 测试连接
	err := client.Connect()
	assert.NoError(t, err, "连接etcd应该成功")

	// 确保在测试结束时关闭连接
	defer func() {
		err := client.Close()
		assert.NoError(t, err, "关闭etcd连接应该成功")
	}()
}

func TestEtcdClient_Ping(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建配置和日志记录器
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建etcd客户端并连接
	client := NewEtcdClient(cfg, logger)
	err := client.Connect()
	require.NoError(t, err, "连接etcd应该成功")

	// 确保在测试结束时关闭连接
	defer func() {
		err := client.Close()
		assert.NoError(t, err, "关闭etcd连接应该成功")
	}()

	// 测试Ping
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.Ping(ctx)
	assert.NoError(t, err, "Ping etcd应该成功")
}

func TestEtcdClient_GetAndGetWithPrefix(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建配置和日志记录器
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建etcd客户端并连接
	client := NewEtcdClient(cfg, logger)
	err := client.Connect()
	require.NoError(t, err, "连接etcd应该成功")

	// 确保在测试结束时关闭连接
	defer func() {
		err := client.Close()
		assert.NoError(t, err, "关闭etcd连接应该成功")
	}()

	// 测试Get - 这将在实际环境中进行，所以可能会失败
	// 如果key不存在，这是预期的行为
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Get(ctx, "test-key")
	// 我们不断言错误，因为key可能不存在
	t.Logf("Get结果: %v", err)

	// 测试GetWithPrefix
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	result, err := client.GetWithPrefix(ctx2, "test")
	// 我们不断言错误，因为前缀可能没有匹配项
	t.Logf("GetWithPrefix结果: %v, %v", result, err)
}

func TestEtcdClient_DNSRecordOperations(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建配置和日志记录器
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建etcd客户端并连接
	client := NewEtcdClient(cfg, logger)
	err := client.Connect()
	require.NoError(t, err, "连接etcd应该成功")

	// 确保在测试结束时关闭连接
	defer func() {
		err := client.Close()
		assert.NoError(t, err, "关闭etcd连接应该成功")
	}()

	// 创建测试DNS记录
	testDomain := "test.example.com"
	testRecord := &DNSRecord{
		Type:  "A",
		Value: "192.168.1.100",
		TTL:   300,
		Tags:  []string{"test", "example"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 测试PutDNSRecord
	err = client.PutDNSRecord(ctx, testDomain, testRecord)
	assert.NoError(t, err, "保存DNS记录应该成功")

	// 测试GetDNSRecord
	retrievedRecord, err := client.GetDNSRecord(ctx, testDomain, "A")
	assert.NoError(t, err, "获取DNS记录应该成功")
	assert.Equal(t, testRecord.Type, retrievedRecord.Type)
	assert.Equal(t, testRecord.Value, retrievedRecord.Value)
	assert.Equal(t, testRecord.TTL, retrievedRecord.TTL)

	// 测试GetDNSRecordsForDomain
	records, err := client.GetDNSRecordsForDomain(ctx, testDomain)
	assert.NoError(t, err, "获取域名的所有DNS记录应该成功")
	assert.NotEmpty(t, records, "应该返回至少一条记录")
	assert.Contains(t, records, "A", "返回的记录中应该包含A记录")
}

func TestEtcdClient_ServiceOperations(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建配置和日志记录器
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建etcd客户端并连接
	client := NewEtcdClient(cfg, logger)
	err := client.Connect()
	require.NoError(t, err, "连接etcd应该成功")

	// 确保在测试结束时关闭连接
	defer func() {
		err := client.Close()
		assert.NoError(t, err, "关闭etcd连接应该成功")
	}()

	// 创建测试服务实例
	testService := &ServiceInstance{
		ServiceName: "test-service",
		InstanceID:  "instance-001",
		IPAddress:   "192.168.1.101",
		Port:        8080,
		Metadata: map[string]string{
			"version": "1.0.0",
			"region":  "cn-north",
		},
		TTL: 30,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 测试RegisterService
	err = client.RegisterService(ctx, testService)
	assert.NoError(t, err, "注册服务实例应该成功")

	// 测试GetServiceInstances
	instances, err := client.GetServiceInstances(ctx, testService.ServiceName)
	assert.NoError(t, err, "获取服务实例列表应该成功")
	assert.NotEmpty(t, instances, "应该返回至少一个服务实例")

	var found bool
	for _, instance := range instances {
		if instance.InstanceID == testService.InstanceID {
			found = true
			assert.Equal(t, testService.ServiceName, instance.ServiceName)
			assert.Equal(t, testService.IPAddress, instance.IPAddress)
			assert.Equal(t, testService.Port, instance.Port)
			break
		}
	}
	assert.True(t, found, "应该找到注册的服务实例")

	// 测试ServiceToDNSRecords
	domain := testService.ServiceName + ".svc.cluster.local"
	dnsRecords, err := client.ServiceToDNSRecords(ctx, domain)
	assert.NoError(t, err, "将服务转换为DNS记录应该成功")
	assert.NotEmpty(t, dnsRecords, "应该返回至少一条DNS记录")
	assert.Contains(t, dnsRecords, "A", "返回的记录中应该包含A记录")

	// 测试DeregisterService
	err = client.DeregisterService(ctx, testService.ServiceName, testService.InstanceID)
	assert.NoError(t, err, "注销服务实例应该成功")

	// 验证服务已被注销
	instances, err = client.GetServiceInstances(ctx, testService.ServiceName)
	if err == nil {
		for _, instance := range instances {
			assert.NotEqual(t, testService.InstanceID, instance.InstanceID, "不应该找到已注销的服务实例")
		}
	}
}

func TestEtcdClient_MultipleRecords(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建配置和日志记录器
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建etcd客户端并连接
	client := NewEtcdClient(cfg, logger)
	err := client.Connect()
	require.NoError(t, err, "连接etcd应该成功")

	// 确保在测试结束时关闭连接
	defer func() {
		err := client.Close()
		assert.NoError(t, err, "关闭etcd连接应该成功")
	}()

	// 测试域名
	testDomain := "multi.example.com"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建并存储A记录
	aRecord := &DNSRecord{
		Type:  "A",
		Value: "192.168.1.200",
		TTL:   300,
	}
	err = client.PutDNSRecord(ctx, testDomain, aRecord)
	assert.NoError(t, err, "保存A记录应该成功")

	// 创建并存储AAAA记录
	aaaaRecord := &DNSRecord{
		Type:  "AAAA",
		Value: "2001:db8::1",
		TTL:   300,
	}
	err = client.PutDNSRecord(ctx, testDomain, aaaaRecord)
	assert.NoError(t, err, "保存AAAA记录应该成功")

	// 创建并存储TXT记录
	txtRecord := &DNSRecord{
		Type:  "TXT",
		Value: "v=spf1 -all",
		TTL:   300,
	}
	err = client.PutDNSRecord(ctx, testDomain, txtRecord)
	assert.NoError(t, err, "保存TXT记录应该成功")

	// 获取所有记录
	records, err := client.GetDNSRecordsForDomain(ctx, testDomain)
	assert.NoError(t, err, "获取域名的所有DNS记录应该成功")

	// 验证返回的记录
	assert.Contains(t, records, "A", "应该包含A记录")
	assert.Contains(t, records, "AAAA", "应该包含AAAA记录")
	assert.Contains(t, records, "TXT", "应该包含TXT记录")

	assert.Equal(t, "192.168.1.200", records["A"].Value)
	assert.Equal(t, "2001:db8::1", records["AAAA"].Value)
	assert.Equal(t, "v=spf1 -all", records["TXT"].Value)
}

func TestEtcdClient_RefreshServiceLease(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建配置和日志记录器
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建etcd客户端并连接
	client := NewEtcdClient(cfg, logger)
	err := client.Connect()
	require.NoError(t, err, "连接etcd应该成功")

	// 确保在测试结束时关闭连接
	defer func() {
		err := client.Close()
		assert.NoError(t, err, "关闭etcd连接应该成功")
	}()

	// 创建测试服务实例，使用较短的TTL以便测试刷新功能
	testService := &ServiceInstance{
		ServiceName: "refresh-service",
		InstanceID:  "refresh-instance-001",
		IPAddress:   "192.168.1.101",
		Port:        8080,
		Metadata: map[string]string{
			"version": "1.0.0",
			"region":  "cn-north",
		},
		TTL: 10, // 使用较短的TTL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 首先注册服务实例
	err = client.RegisterService(ctx, testService)
	assert.NoError(t, err, "注册服务实例应该成功")

	// 延迟一小段时间，确保服务已经注册
	time.Sleep(1 * time.Second)

	// 刷新服务实例的租约，将TTL更新为新值
	newTTL := 30
	err = client.RefreshServiceLease(ctx, testService.ServiceName, testService.InstanceID, newTTL)
	assert.NoError(t, err, "刷新服务实例租约应该成功")

	// 获取服务实例，验证TTL已更新
	instances, err := client.GetServiceInstances(ctx, testService.ServiceName)
	assert.NoError(t, err, "获取服务实例列表应该成功")
	assert.NotEmpty(t, instances, "应该返回至少一个服务实例")

	var found bool
	for _, instance := range instances {
		if instance.InstanceID == testService.InstanceID {
			found = true
			assert.Equal(t, testService.ServiceName, instance.ServiceName)
			assert.Equal(t, testService.IPAddress, instance.IPAddress)
			assert.Equal(t, testService.Port, instance.Port)
			assert.Equal(t, newTTL, instance.TTL, "TTL应该已经更新为新值")
			break
		}
	}
	assert.True(t, found, "应该找到注册的服务实例")

	// 测试完成后，注销服务实例
	err = client.DeregisterService(ctx, testService.ServiceName, testService.InstanceID)
	assert.NoError(t, err, "注销服务实例应该成功")
}

func TestEtcdClient_UpdateDNSConfig(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建配置和日志记录器
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建etcd客户端并连接
	client := NewEtcdClient(cfg, logger)
	err := client.Connect()
	require.NoError(t, err, "连接etcd应该成功")

	// 确保在测试结束时关闭连接
	defer func() {
		err := client.Close()
		assert.NoError(t, err, "关闭etcd连接应该成功")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 测试1: 更新字符串数组类型的配置 (上游DNS服务器列表)
	upstreamDNS := []string{"8.8.8.8:53", "1.1.1.1:53"}
	err = client.UpdateDNSConfig(ctx, "upstream_dns", upstreamDNS)
	assert.NoError(t, err, "更新上游DNS配置应该成功")

	// 验证配置是否已保存
	configs, err := client.GetDNSConfig(ctx)
	assert.NoError(t, err, "获取DNS配置应该成功")
	assert.Contains(t, configs, "upstream_dns", "配置中应该包含upstream_dns")

	// 验证保存的内容是否正确 (需要解析JSON)
	var savedUpstreamDNS []string
	err = json.Unmarshal([]byte(configs["upstream_dns"]), &savedUpstreamDNS)
	assert.NoError(t, err, "应该能够解析保存的上游DNS配置")
	assert.ElementsMatch(t, upstreamDNS, savedUpstreamDNS, "保存的上游DNS配置应该与原始配置匹配")

	// 测试2: 更新字符串类型的配置
	testStringKey := "test_string_config"
	testStringValue := "test_value"
	err = client.UpdateDNSConfig(ctx, testStringKey, testStringValue)
	assert.NoError(t, err, "更新字符串配置应该成功")

	// 验证字符串配置是否已保存
	configs, err = client.GetDNSConfig(ctx)
	assert.NoError(t, err, "获取DNS配置应该成功")
	assert.Contains(t, configs, testStringKey, "配置中应该包含测试的字符串键")
	assert.Equal(t, testStringValue, configs[testStringKey], "保存的字符串值应该与原始值匹配")

	// 测试3: 更新结构体类型的配置
	type TestConfig struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
		Count   int    `json:"count"`
	}
	testConfig := TestConfig{
		Name:    "test_config",
		Enabled: true,
		Count:   42,
	}
	err = client.UpdateDNSConfig(ctx, "test_struct_config", testConfig)
	assert.NoError(t, err, "更新结构体配置应该成功")

	// 验证结构体配置是否已保存
	configs, err = client.GetDNSConfig(ctx)
	assert.NoError(t, err, "获取DNS配置应该成功")
	assert.Contains(t, configs, "test_struct_config", "配置中应该包含测试的结构体键")

	// 解析保存的结构体配置
	var savedConfig TestConfig
	err = json.Unmarshal([]byte(configs["test_struct_config"]), &savedConfig)
	assert.NoError(t, err, "应该能够解析保存的结构体配置")
	assert.Equal(t, testConfig.Name, savedConfig.Name, "保存的结构体Name字段应该匹配")
	assert.Equal(t, testConfig.Enabled, savedConfig.Enabled, "保存的结构体Enabled字段应该匹配")
	assert.Equal(t, testConfig.Count, savedConfig.Count, "保存的结构体Count字段应该匹配")

	// 清理测试数据
	etcdClientImpl := client.(*EtcdClient)
	_, _ = etcdClientImpl.Client().Delete(ctx, "/config/dns/upstream_dns")
	_, _ = etcdClientImpl.Client().Delete(ctx, "/config/dns/"+testStringKey)
	_, _ = etcdClientImpl.Client().Delete(ctx, "/config/dns/test_struct_config")
}
