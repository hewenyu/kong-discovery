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
	// 创建命名空间，不依赖于类型断言
	ctx := context.Background()

	// 创建测试命名空间
	nsStorage := etcd.NewNamespaceStorage(getEtcdClient(t))

	// 创建默认命名空间
	defaultNs := &storage.Namespace{
		Name:        "default",
		Description: "默认命名空间",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := nsStorage.CreateNamespace(ctx, defaultNs)
	// 忽略已存在的命名空间错误
	if err != nil {
		if se, ok := err.(*storage.StorageError); !ok || se.Code != storage.ErrAlreadyExists {
			require.NoError(t, err, "创建默认命名空间失败")
		}
	}

	// 创建测试命名空间
	namespace := &storage.Namespace{
		Name:        "test-ns",
		Description: "测试命名空间",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err = nsStorage.CreateNamespace(ctx, namespace)
	// 忽略已存在的命名空间错误
	if err != nil {
		if se, ok := err.(*storage.StorageError); !ok || se.Code != storage.ErrAlreadyExists {
			require.NoError(t, err, "创建测试命名空间失败")
		}
	}

	// 创建测试服务
	service := &storage.Service{
		ID:            "test-service-1",
		Name:          "app",
		Namespace:     "test-ns", // 添加命名空间字段
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
	err = serviceStorage.RegisterService(ctx, service)
	require.NoError(t, err, "注册测试服务失败")

	// 同时在默认命名空间注册相同服务用于测试
	defaultService := &storage.Service{
		ID:            "test-service-default",
		Name:          "app",
		Namespace:     "default", // 使用默认命名空间
		IP:            "192.168.1.20",
		Port:          8080,
		Tags:          []string{"test", "dns"},
		Metadata:      map[string]string{"version": "1.0"},
		Health:        "healthy",
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		TTL:           60,
	}

	// 注册默认命名空间服务
	err = serviceStorage.RegisterService(ctx, defaultService)
	require.NoError(t, err, "注册默认命名空间服务失败")

	return service
}

// 获取etcd客户端
func getEtcdClient(t *testing.T) *etcd.Client {
	etcdConfig := getEtcdConfig()
	etcdConfig.DialTimeout = "10s" // 使用更长的超时时间
	client, err := etcd.NewClient(etcdConfig)
	require.NoError(t, err, "创建etcd客户端失败")
	return client
}

// 清理测试服务数据
func cleanupTestService(t *testing.T, serviceStorage storage.ServiceStorage, serviceID string) {
	ctx := context.Background()
	// 清理测试命名空间的服务
	err := serviceStorage.DeregisterService(ctx, serviceID)
	require.NoError(t, err, "清理测试服务失败")

	// 清理默认命名空间的服务
	err = serviceStorage.DeregisterService(ctx, "test-service-default")
	// 忽略可能的服务不存在错误
	if err != nil {
		if se, ok := err.(*storage.StorageError); !ok || se.Code != storage.ErrNotFound {
			require.NoError(t, err, "清理默认命名空间服务失败")
		}
	}
}

func TestDNSResolutionWithRealServer(t *testing.T) {
	// 跳过CI环境测试
	if os.Getenv("CI") == "true" {
		t.Skip("在CI环境中跳过etcd集成测试")
	}

	// 创建测试配置
	conf := &config.Config{
		Server: config.ServerConfig{
			RegisterPort: 8080,
			AdminPort:    9090,
			DNSPort:      15353, // 使用非标准端口避免冲突
		},
		DNS: config.DNSConfig{
			Domain:   "service.test",
			Upstream: []string{"8.8.8.8:53", "114.114.114.114:53"},
			CacheTTL: 60,
		},
		Heartbeat: config.HeartbeatConfig{
			Interval: 30,
			Timeout:  90,
		},
	}

	// 获取etcd配置
	etcdConfig := getEtcdConfig()
	client, err := etcd.NewClient(etcdConfig)
	require.NoError(t, err, "连接etcd失败")

	// 创建服务存储
	serviceStorage := etcd.NewServiceStorage(client)
	// 创建命名空间存储
	namespaceStorage := etcd.NewNamespaceStorage(client)

	// 准备测试数据
	service := prepareTestService(t, serviceStorage)
	defer cleanupTestService(t, serviceStorage, service.ID)

	// 创建DNS服务器
	server, err := NewServer(conf, serviceStorage, namespaceStorage)
	require.NoError(t, err, "创建DNS服务器失败")

	// 启动服务器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = server.Start(ctx)
	require.NoError(t, err, "启动DNS服务器失败")
	defer server.Stop()

	// 等待服务器完全启动
	time.Sleep(1 * time.Second)

	// 使用真实的DNS客户端测试A记录查询
	t.Run("A记录解析测试", func(t *testing.T) {
		dnsClient := new(dns.Client)
		msg := new(dns.Msg)
		msg.SetQuestion("app.service.test.", dns.TypeA)
		msg.RecursionDesired = true

		response, _, err := dnsClient.Exchange(msg, "127.0.0.1:15353")
		require.NoError(t, err, "DNS A记录查询失败")
		require.NotNil(t, response, "没有收到DNS响应")

		assert.Equal(t, dns.RcodeSuccess, response.Rcode, "DNS响应码不正确")
		assert.True(t, response.Authoritative, "DNS响应不是权威响应")
		assert.GreaterOrEqual(t, len(response.Answer), 1, "DNS响应没有包含回答部分")

		// 验证A记录
		if len(response.Answer) > 0 {
			aRecord, ok := response.Answer[0].(*dns.A)
			if assert.True(t, ok, "响应不是A记录") {
				assert.Equal(t, net.ParseIP("192.168.1.20").String(), aRecord.A.String(), "IP地址不匹配")
			}
		}
	})

	// 使用真实的DNS客户端测试SRV记录查询
	t.Run("SRV记录解析测试", func(t *testing.T) {
		dnsClient := new(dns.Client)
		msg := new(dns.Msg)
		// 修改为使用默认命名空间的查询格式
		msg.SetQuestion("_app._tcp.default.service.test.", dns.TypeSRV)
		msg.RecursionDesired = true

		response, _, err := dnsClient.Exchange(msg, "127.0.0.1:15353")
		require.NoError(t, err, "DNS SRV记录查询失败")
		require.NotNil(t, response, "没有收到DNS响应")

		assert.Equal(t, dns.RcodeSuccess, response.Rcode, "DNS响应码不正确")
		assert.True(t, response.Authoritative, "DNS响应不是权威响应")
		assert.GreaterOrEqual(t, len(response.Answer), 1, "DNS响应没有包含回答部分")

		// 验证SRV记录
		if len(response.Answer) > 0 {
			srvRecord, ok := response.Answer[0].(*dns.SRV)
			if assert.True(t, ok, "响应不是SRV记录") {
				assert.Equal(t, uint16(8080), srvRecord.Port, "端口不匹配")
				assert.Equal(t, "app.default.service.test.", srvRecord.Target, "目标不匹配")
			}
		}
	})

	// 添加命名空间相关的A记录测试
	t.Run("带命名空间的A记录解析测试", func(t *testing.T) {
		dnsClient := new(dns.Client)
		msg := new(dns.Msg)
		msg.SetQuestion("app.test-ns.service.test.", dns.TypeA)
		msg.RecursionDesired = true

		response, _, err := dnsClient.Exchange(msg, "127.0.0.1:15353")
		require.NoError(t, err, "带命名空间的DNS A记录查询失败")
		require.NotNil(t, response, "没有收到DNS响应")

		assert.Equal(t, dns.RcodeSuccess, response.Rcode, "DNS响应码不正确")
		assert.True(t, response.Authoritative, "DNS响应不是权威响应")
		assert.GreaterOrEqual(t, len(response.Answer), 1, "DNS响应没有包含回答部分")

		// 验证A记录
		if len(response.Answer) > 0 {
			aRecord, ok := response.Answer[0].(*dns.A)
			if assert.True(t, ok, "响应不是A记录") {
				assert.Equal(t, net.ParseIP("192.168.1.10").String(), aRecord.A.String(), "IP地址不匹配")
			}
		}
	})

	// 添加命名空间相关的SRV记录测试
	t.Run("带命名空间的SRV记录解析测试", func(t *testing.T) {
		dnsClient := new(dns.Client)
		msg := new(dns.Msg)
		msg.SetQuestion("_app._tcp.test-ns.service.test.", dns.TypeSRV)
		msg.RecursionDesired = true

		response, _, err := dnsClient.Exchange(msg, "127.0.0.1:15353")
		require.NoError(t, err, "带命名空间的DNS SRV记录查询失败")
		require.NotNil(t, response, "没有收到DNS响应")

		assert.Equal(t, dns.RcodeSuccess, response.Rcode, "DNS响应码不正确")
		assert.True(t, response.Authoritative, "DNS响应不是权威响应")
		assert.GreaterOrEqual(t, len(response.Answer), 1, "DNS响应没有包含回答部分")

		// 验证SRV记录
		if len(response.Answer) > 0 {
			srvRecord, ok := response.Answer[0].(*dns.SRV)
			if assert.True(t, ok, "响应不是SRV记录") {
				assert.Equal(t, uint16(8080), srvRecord.Port, "端口不匹配")
				assert.Equal(t, "app.test-ns.service.test.", srvRecord.Target, "目标不匹配")
			}
		}
	})

	// 添加明确指定默认命名空间的A记录测试
	t.Run("默认命名空间A记录解析测试", func(t *testing.T) {
		dnsClient := new(dns.Client)
		msg := new(dns.Msg)
		msg.SetQuestion("app.default.service.test.", dns.TypeA)
		msg.RecursionDesired = true

		response, _, err := dnsClient.Exchange(msg, "127.0.0.1:15353")
		require.NoError(t, err, "默认命名空间的DNS A记录查询失败")
		require.NotNil(t, response, "没有收到DNS响应")

		assert.Equal(t, dns.RcodeSuccess, response.Rcode, "DNS响应码不正确")
		assert.True(t, response.Authoritative, "DNS响应不是权威响应")
		assert.GreaterOrEqual(t, len(response.Answer), 1, "DNS响应没有包含回答部分")

		// 验证A记录
		if len(response.Answer) > 0 {
			aRecord, ok := response.Answer[0].(*dns.A)
			if assert.True(t, ok, "响应不是A记录") {
				assert.Equal(t, net.ParseIP("192.168.1.20").String(), aRecord.A.String(), "IP地址不匹配")
			}
		}
	})
}
