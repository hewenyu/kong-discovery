package dns

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hewenyu/kong-discovery/pkg/config"
	"github.com/hewenyu/kong-discovery/pkg/storage"
	"github.com/hewenyu/kong-discovery/pkg/storage/etcd"
)

// 从环境变量获取etcd配置
func getEtcdConfig() *config.EtcdConfig {
	endpoints := os.Getenv("ETCD_ENDPOINTS")
	if endpoints == "" {
		endpoints = "localhost:2379" // 默认值
	}

	return &config.EtcdConfig{
		Endpoints:   strings.Split(endpoints, ","),
		DialTimeout: "5s",
		Username:    os.Getenv("ETCD_USERNAME"),
		Password:    os.Getenv("ETCD_PASSWORD"),
	}
}

// 准备测试服务数据
func prepareTestService(t *testing.T, serviceStorage storage.ServiceStorage) *storage.Service {
	// 创建测试服务
	service := &storage.Service{
		ID:            "test-service-1",
		Name:          "app",
		IP:            "192.168.1.10",
		Port:          8080,
		Tags:          []string{"test", "dns"},
		Metadata:      map[string]string{"version": "1.0"},
		Health:        "healthy",
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		TTL:           60,
	}

	// 注册服务
	ctx := context.Background()
	err := serviceStorage.RegisterService(ctx, service)
	require.NoError(t, err, "注册测试服务失败")

	return service
}

// 清理测试服务数据
func cleanupTestService(t *testing.T, serviceStorage storage.ServiceStorage, serviceID string) {
	ctx := context.Background()
	err := serviceStorage.DeregisterService(ctx, serviceID)
	require.NoError(t, err, "清理测试服务失败")
}

func TestHandlerServeDNS(t *testing.T) {
	// 跳过CI环境测试
	if os.Getenv("CI") == "true" {
		t.Skip("在CI环境中跳过etcd集成测试")
	}

	// 创建etcd客户端
	etcdConfig := getEtcdConfig()
	client, err := etcd.NewClient(etcdConfig)
	require.NoError(t, err, "连接etcd失败")

	// 创建服务存储
	serviceStorage := etcd.NewServiceStorage(client)

	// 准备测试数据
	service := prepareTestService(t, serviceStorage)
	defer cleanupTestService(t, serviceStorage, service.ID)

	// 创建记录管理器
	recordManager := NewRecordManager(serviceStorage, "service.local", 60)

	// 创建缓存
	cache := NewDNSCache(60)

	// 创建上游解析器
	upstreamResolver := NewUpstreamResolver([]string{"8.8.8.8:53"}, cache)

	// 创建处理器
	handler := NewHandler(recordManager, upstreamResolver, cache, "service.local")

	// 创建DNS请求
	req := new(dns.Msg)
	req.SetQuestion("app.service.local.", dns.TypeA)
	req.Id = 1234

	// 创建响应编写器
	w := &mockResponseWriter{}

	// 调用处理函数
	handler.ServeDNS(w, req)

	// 验证结果
	require.NotNil(t, w.msg, "没有收到DNS响应")
	assert.Equal(t, dns.RcodeSuccess, w.msg.Rcode, "DNS响应码不正确")
	assert.True(t, w.msg.Authoritative, "DNS响应不是权威响应")
	assert.Equal(t, req.Id, w.msg.Id, "DNS响应ID不匹配")
	assert.GreaterOrEqual(t, len(w.msg.Answer), 1, "DNS响应没有包含回答部分")

	// 测试SRV记录
	reqSRV := new(dns.Msg)
	reqSRV.SetQuestion("_app._tcp.service.local.", dns.TypeSRV)
	reqSRV.Id = 5678

	wSRV := &mockResponseWriter{}
	handler.ServeDNS(wSRV, reqSRV)

	require.NotNil(t, wSRV.msg, "没有收到SRV记录DNS响应")
	if wSRV.msg != nil && wSRV.msg.Rcode == dns.RcodeSuccess {
		assert.GreaterOrEqual(t, len(wSRV.msg.Answer), 1, "SRV记录DNS响应没有包含回答部分")
	}
}

// mockResponseWriter 模拟DNS响应编写器
type mockResponseWriter struct {
	msg *dns.Msg
}

func (m *mockResponseWriter) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 53}
}

func (m *mockResponseWriter) RemoteAddr() net.Addr {
	return &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1234}
}

func (m *mockResponseWriter) WriteMsg(msg *dns.Msg) error {
	m.msg = msg
	return nil
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (m *mockResponseWriter) Close() error {
	return nil
}

func (m *mockResponseWriter) TsigStatus() error {
	return nil
}

func (m *mockResponseWriter) TsigTimersOnly(bool) {}

func (m *mockResponseWriter) Hijack() {}
