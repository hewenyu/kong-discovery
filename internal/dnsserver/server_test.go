package dnsserver

import (
	"context"
	"fmt"
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

func TestDNSServer_StartAndShutdown(t *testing.T) {
	// 准备测试配置
	cfg := &config.Config{}
	cfg.DNS.ListenAddress = "127.0.0.1"
	cfg.DNS.Port = 15353 // 使用非标准端口避免冲突
	cfg.DNS.Protocol = "udp"

	// 创建服务器
	server := NewDNSServer(cfg, &MockLogger{})

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
	// 跳过集成测试如果环境变量设置
	// TODO: 实现基于环境变量跳过集成测试的逻辑

	// 准备测试配置
	cfg := &config.Config{}
	cfg.DNS.ListenAddress = "127.0.0.1"
	cfg.DNS.Port = 15353 // 使用非标准端口避免冲突
	cfg.DNS.Protocol = "udp"

	// 创建并启动服务器
	server := NewDNSServer(cfg, &MockLogger{})
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
	// 跳过集成测试如果环境变量设置
	// TODO: 实现基于环境变量跳过集成测试的逻辑

	// 准备测试配置
	cfg := &config.Config{}
	cfg.DNS.ListenAddress = "127.0.0.1"
	cfg.DNS.Port = 15353 // 使用非标准端口避免冲突
	cfg.DNS.Protocol = "udp"

	// 创建并启动服务器
	server := NewDNSServer(cfg, &MockLogger{})

	// 设置模拟的etcd客户端
	mockEtcdClient := &MockEtcdClient{}
	server.SetEtcdClient(mockEtcdClient)

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
