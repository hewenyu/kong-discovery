package dnsserver

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hewenyu/kong-discovery/internal/config"
	"github.com/hewenyu/kong-discovery/internal/etcdclient"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

// MockLogger 实现config.Logger接口，用于测试
type MockLogger struct{}

func (l *MockLogger) Debug(msg string, fields ...zapcore.Field) {}
func (l *MockLogger) Info(msg string, fields ...zapcore.Field)  {}
func (l *MockLogger) Warn(msg string, fields ...zapcore.Field)  {}
func (l *MockLogger) Error(msg string, fields ...zapcore.Field) {}
func (l *MockLogger) Fatal(msg string, fields ...zapcore.Field) {}

// MockEtcdClient 实现etcdclient.Client接口，用于测试
type MockEtcdClient struct{}

func (m *MockEtcdClient) Connect() error                                      { return nil }
func (m *MockEtcdClient) Close() error                                        { return nil }
func (m *MockEtcdClient) Ping(ctx context.Context) error                      { return nil }
func (m *MockEtcdClient) Get(ctx context.Context, key string) (string, error) { return "", nil }
func (m *MockEtcdClient) GetWithPrefix(ctx context.Context, prefix string) (map[string]string, error) {
	return nil, nil
}

func (m *MockEtcdClient) GetDNSRecord(ctx context.Context, domain string, recordType string) (*etcdclient.DNSRecord, error) {
	// 模拟test.etcd.local的A记录
	if domain == "test.etcd.local" && recordType == "A" {
		return &etcdclient.DNSRecord{
			Type:  "A",
			Value: "5.6.7.8",
			TTL:   300,
		}, nil
	}
	return nil, fmt.Errorf("记录不存在")
}

func (m *MockEtcdClient) PutDNSRecord(ctx context.Context, domain string, record *etcdclient.DNSRecord) error {
	return nil
}

func (m *MockEtcdClient) GetDNSRecordsForDomain(ctx context.Context, domain string) (map[string]*etcdclient.DNSRecord, error) {
	return nil, nil
}

// 实现剩余的接口方法
func (m *MockEtcdClient) RegisterService(ctx context.Context, instance *etcdclient.ServiceInstance) error {
	return nil
}

func (m *MockEtcdClient) DeregisterService(ctx context.Context, serviceName, instanceID string) error {
	return nil
}

func (m *MockEtcdClient) GetServiceInstances(ctx context.Context, serviceName string) ([]*etcdclient.ServiceInstance, error) {
	return nil, nil
}

func (m *MockEtcdClient) ServiceToDNSRecords(ctx context.Context, domain string) (map[string]*etcdclient.DNSRecord, error) {
	// 为test.etcd.local域名返回模拟的DNS记录
	if domain == "test.etcd.local" {
		records := make(map[string]*etcdclient.DNSRecord)
		records["A"] = &etcdclient.DNSRecord{
			Type:  "A",
			Value: "5.6.7.8",
			TTL:   300,
		}
		return records, nil
	}
	return nil, fmt.Errorf("服务不存在")
}

// RefreshServiceLease 模拟刷新服务租约的功能
func (m *MockEtcdClient) RefreshServiceLease(ctx context.Context, serviceName, instanceID string, ttl int) error {
	// 简单返回nil，因为测试不需要真正刷新租约
	return nil
}

// 创建一个测试用的配置，使用环境变量中的etcd地址
func createTestConfig(t *testing.T) *config.Config {
	t.Helper()

	// 从环境变量中获取etcd地址
	etcdEndpoints := os.Getenv("KONG_DISCOVERY_ETCD_ENDPOINTS")
	require.NotEmpty(t, etcdEndpoints, "环境变量KONG_DISCOVERY_ETCD_ENDPOINTS必须设置")

	// 创建配置
	cfg := &config.Config{}
	cfg.Etcd.Endpoints = []string{etcdEndpoints}
	cfg.Etcd.Username = "" // 如果需要认证，设置相应的值
	cfg.Etcd.Password = "" // 如果需要认证，设置相应的值
	cfg.DNS.ListenAddress = "127.0.0.1"
	cfg.DNS.Port = 15353 // 使用非标准端口避免冲突
	cfg.DNS.Protocol = "udp"

	return cfg
}

// 创建测试用的日志记录器
func createTestLogger(t *testing.T) config.Logger {
	t.Helper()

	logger, err := config.NewLogger(true)
	require.NoError(t, err, "创建测试日志记录器失败")

	return logger
}

// 准备测试DNS记录
func prepareTestDNSRecord(t *testing.T, client etcdclient.Client) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建测试DNS记录
	testRecord := &etcdclient.DNSRecord{
		Type:  "A",
		Value: "5.6.7.8",
		TTL:   300,
	}

	err := client.PutDNSRecord(ctx, "test.etcd.local", testRecord)
	require.NoError(t, err, "创建测试DNS记录失败")
}

// 准备测试服务实例
func prepareTestService(t *testing.T, client etcdclient.Client) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建测试服务实例
	testService := &etcdclient.ServiceInstance{
		ServiceName: "test-service",
		InstanceID:  "instance-001",
		IPAddress:   "10.0.0.1",
		Port:        8080,
		TTL:         60,
	}

	err := client.RegisterService(ctx, testService)
	require.NoError(t, err, "注册测试服务实例失败")
}

// 清理测试数据
func cleanupTestData(t *testing.T, client etcdclient.Client) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 清理测试服务实例
	_ = client.DeregisterService(ctx, "test-service", "instance-001")

	// 尝试清理测试DNS记录 (这里没有现成的删除DNS记录的方法，实际项目中可能需要添加)
}

func TestDNSServer_StartAndShutdown(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 准备测试配置
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建服务器
	server := NewDNSServer(cfg, logger)

	// 启动服务器
	err := server.Start()
	require.NoError(t, err)

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestDNSServer_QueryHardcodedRecord(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 准备测试配置
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建并启动服务器
	server := NewDNSServer(cfg, logger)
	err := server.Start()
	require.NoError(t, err)

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 创建DNS客户端
	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion("test.local.", dns.TypeA)
	m.RecursionDesired = true

	// 发送查询
	r, _, err := c.Exchange(m, "127.0.0.1:15353")
	require.NoError(t, err)
	require.NotNil(t, r)

	// 验证响应
	assert.Equal(t, dns.RcodeSuccess, r.Rcode)
	assert.GreaterOrEqual(t, len(r.Answer), 1)

	// 检查A记录
	if len(r.Answer) > 0 {
		if a, ok := r.Answer[0].(*dns.A); ok {
			assert.Equal(t, "1.2.3.4", a.A.String())
		} else {
			t.Errorf("Expected A record, got %T", r.Answer[0])
		}
	}

	// 关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestDNSServer_QueryEtcdRecord(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 准备测试配置
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建真实的etcd客户端
	client := etcdclient.CreateEtcdClientForTest(t)
	defer client.Close()

	// 准备测试数据
	prepareTestDNSRecord(t, client)
	defer cleanupTestData(t, client)

	// 创建并启动服务器
	server := NewDNSServer(cfg, logger)

	// 设置etcd客户端
	server.SetEtcdClient(client)

	err := server.Start()
	require.NoError(t, err)

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 创建DNS客户端
	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion("test.etcd.local.", dns.TypeA)
	m.RecursionDesired = true

	// 发送查询
	r, _, err := c.Exchange(m, "127.0.0.1:15353")
	require.NoError(t, err)
	require.NotNil(t, r)

	// 验证响应
	assert.Equal(t, dns.RcodeSuccess, r.Rcode)
	assert.GreaterOrEqual(t, len(r.Answer), 1)

	// 检查A记录
	if len(r.Answer) > 0 {
		if a, ok := r.Answer[0].(*dns.A); ok {
			assert.Equal(t, "5.6.7.8", a.A.String())
		} else {
			t.Errorf("Expected A record, got %T", r.Answer[0])
		}
	}

	// 关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestDNSServer_ForwardToUpstream(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 准备测试配置
	cfg := createTestConfig(t)
	cfg.DNS.UpstreamDNS = "8.8.8.8:53" // 使用Google的DNS服务器作为上游
	logger := createTestLogger(t)

	// 创建真实的etcd客户端
	client := etcdclient.CreateEtcdClientForTest(t)
	defer client.Close()

	// 创建并启动服务器
	server := NewDNSServer(cfg, logger)

	// 设置etcd客户端
	server.SetEtcdClient(client)

	err := server.Start()
	require.NoError(t, err)

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 创建DNS客户端
	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion("example.com.", dns.TypeA) // 查询example.com，应该被转发
	m.RecursionDesired = true

	// 发送查询
	r, _, err := c.Exchange(m, "127.0.0.1:15353")
	require.NoError(t, err)
	require.NotNil(t, r)

	// 验证响应是成功的
	assert.Equal(t, dns.RcodeSuccess, r.Rcode, "转发查询应该成功")
	assert.True(t, len(r.Answer) > 0, "应该返回至少一个回答")

	// 关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestDNSServer_NoUpstreamDNS(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 准备测试配置，不设置上游DNS
	cfg := createTestConfig(t)
	cfg.DNS.UpstreamDNS = "" // 不设置上游DNS
	logger := createTestLogger(t)

	// 创建真实的etcd客户端
	client := etcdclient.CreateEtcdClientForTest(t)
	defer client.Close()

	// 创建并启动服务器
	server := NewDNSServer(cfg, logger)

	// 设置etcd客户端
	server.SetEtcdClient(client)

	err := server.Start()
	require.NoError(t, err)

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 创建DNS客户端
	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion("unknown.example.", dns.TypeA) // 查询未知域名
	m.RecursionDesired = true

	// 发送查询
	r, _, err := c.Exchange(m, "127.0.0.1:15353")
	require.NoError(t, err)
	require.NotNil(t, r)

	// 验证响应是NXDOMAIN（名称不存在）
	assert.Equal(t, dns.RcodeNameError, r.Rcode, "未知域名查询应该返回NXDOMAIN")
	assert.Equal(t, 0, len(r.Answer), "不应该返回任何答案")

	// 关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.Shutdown(ctx)
	assert.NoError(t, err)
}
