package dns

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hewenyu/kong-discovery/internal/core/config"
	"github.com/hewenyu/kong-discovery/internal/core/model"
	dnssrv "github.com/hewenyu/kong-discovery/internal/dns"
	"github.com/hewenyu/kong-discovery/internal/store/etcd"
	"github.com/hewenyu/kong-discovery/internal/store/service"
	dnspkg "github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试DNS服务器端口
const dnsPort = 15353

// 获取etcd客户端
func getEtcdClient() (*etcd.Client, error) {
	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
	if etcdEndpoints == "" {
		// 默认使用本地etcd
		etcdEndpoints = "localhost:2379"
	}

	return etcd.NewClient(&config.EtcdConfig{
		Endpoints:      []string{etcdEndpoints},
		DialTimeout:    5 * time.Second,
		RequestTimeout: 5 * time.Second,
	})
}

// 跳过测试如果无法连接etcd
func skipIfNoEtcd(t *testing.T) *etcd.Client {
	etcdClient, err := getEtcdClient()
	if err != nil {
		t.Skip("跳过测试：无法连接etcd")
		return nil
	}

	// 测试etcd连接
	ctx := context.Background()
	testKey := "/test/dns-integration-test-connection"
	testValue := []byte("test-connection")
	err = etcdClient.Put(ctx, testKey, testValue)
	if err != nil {
		etcdClient.Close()
		t.Skip("跳过测试：etcd连接测试失败")
		return nil
	}

	value, err := etcdClient.Get(ctx, testKey)
	if err != nil || string(value) != string(testValue) {
		etcdClient.Close()
		t.Skip("跳过测试：etcd读取测试失败")
		return nil
	}

	err = etcdClient.Delete(ctx, testKey)
	if err != nil {
		etcdClient.Close()
		t.Skip("跳过测试：etcd删除测试失败")
		return nil
	}

	return etcdClient
}

// 启动DNS服务器
func startDNSServer(t *testing.T, serviceStore service.ServiceStore) dnssrv.Service {
	// 创建DNS服务配置
	dnsConfig := &dnssrv.Config{
		DNSAddr:      "127.0.0.1:" + strconv.Itoa(dnsPort),
		Domain:       "service.local",
		TTL:          60,
		Timeout:      5 * time.Second,
		UpstreamDNS:  []string{"8.8.8.8:53", "114.114.114.114:53"},
		EnableTCP:    true,
		EnableUDP:    true,
		ServiceStore: serviceStore,
	}

	// 创建DNS服务器
	server := dnssrv.NewServer(dnsConfig)
	require.NotNil(t, server, "创建DNS服务器失败")

	// 启动DNS服务器
	ctx := context.Background()
	err := server.Start(ctx)
	require.NoError(t, err, "启动DNS服务器失败")

	// 等待服务器启动
	time.Sleep(1 * time.Second)

	return server
}

// 注册测试服务
func registerTestService(t *testing.T, ctx context.Context, store service.ServiceStore, id, name, namespace, ip string, port int) {
	service := &model.Service{
		ID:            id,
		Name:          name,
		Namespace:     namespace,
		IP:            ip,
		Port:          port,
		Health:        model.HealthStatusHealthy,
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		TTL:           60 * time.Second,
	}

	// 清理可能存在的服务
	_ = store.Deregister(ctx, id)

	// 注册服务
	err := store.Register(ctx, service)
	require.NoError(t, err, "注册测试服务失败")
}

// 执行DNS查询
func doDNSQuery(t *testing.T, domain string, qType uint16) *dnspkg.Msg {
	c := new(dnspkg.Client)
	// 增加UDP缓冲区大小，解决"buffer size too small"错误
	c.UDPSize = 65535

	m := new(dnspkg.Msg)
	m.SetQuestion(domain, qType)
	m.RecursionDesired = true

	// 设置EDNS0选项以支持更大的UDP包
	o := new(dnspkg.OPT)
	o.Hdr.Name = "."
	o.Hdr.Rrtype = dnspkg.TypeOPT
	o.SetUDPSize(dnspkg.DefaultMsgSize)
	m.Extra = append(m.Extra, o)

	r, _, err := c.Exchange(m, "127.0.0.1:"+strconv.Itoa(dnsPort))
	require.NoError(t, err, "DNS查询失败")
	require.NotNil(t, r, "未收到DNS响应")
	return r
}

// TestDNSIntegration 集成测试DNS服务的功能
func TestDNSIntegration(t *testing.T) {
	// 获取etcd客户端，如果连接失败则跳过测试
	etcdClient := skipIfNoEtcd(t)
	if etcdClient == nil {
		return
	}
	defer etcdClient.Close()

	// 创建上下文
	ctx := context.Background()

	// 创建服务存储
	serviceStore := service.NewEtcdServiceStore(etcdClient, "default")

	// 启动DNS服务器
	dnsServer := startDNSServer(t, serviceStore)
	defer dnsServer.Stop()

	// 测试A记录解析
	t.Run("ARecordLookup", func(t *testing.T) {
		// 注册测试服务
		const (
			serviceID   = "test-a-record-1"
			serviceName = "test-a-record"
			namespace   = "default"
			serviceIP   = "192.168.1.100"
			servicePort = 8080
		)

		registerTestService(t, ctx, serviceStore, serviceID, serviceName, namespace, serviceIP, servicePort)
		defer serviceStore.Deregister(ctx, serviceID)

		// 查询A记录
		domain := serviceName + "." + namespace + ".service.local."
		r := doDNSQuery(t, domain, dnspkg.TypeA)

		// 检查响应代码
		assert.Equal(t, dnspkg.RcodeSuccess, r.Rcode, "DNS响应错误码不正确")

		// 检查响应中是否有A记录
		assert.GreaterOrEqual(t, len(r.Answer), 1, "DNS响应中没有A记录")

		// 检查A记录的IP是否匹配
		found := false
		for _, ans := range r.Answer {
			if a, ok := ans.(*dnspkg.A); ok {
				if a.A.String() == serviceIP {
					found = true
					break
				}
			}
		}
		assert.True(t, found, "DNS响应中没有找到正确的IP地址")
	})

	// 测试SRV记录解析
	t.Run("SRVRecordLookup", func(t *testing.T) {
		// 注册测试服务
		const (
			serviceID   = "test-srv-record-1"
			serviceName = "test-srv-record"
			namespace   = "default"
			serviceIP   = "192.168.1.101"
			servicePort = 8081
		)

		registerTestService(t, ctx, serviceStore, serviceID, serviceName, namespace, serviceIP, servicePort)
		defer serviceStore.Deregister(ctx, serviceID)

		// 查询SRV记录
		domain := serviceName + "." + namespace + ".service.local."
		r := doDNSQuery(t, domain, dnspkg.TypeSRV)

		// 检查响应代码
		assert.Equal(t, dnspkg.RcodeSuccess, r.Rcode, "DNS响应错误码不正确")

		// 检查响应中是否有SRV记录
		assert.GreaterOrEqual(t, len(r.Answer), 1, "DNS响应中没有SRV记录")

		// 检查SRV记录的端口是否匹配
		found := false
		for _, ans := range r.Answer {
			if srv, ok := ans.(*dnspkg.SRV); ok {
				if srv.Port == uint16(servicePort) {
					found = true
					break
				}
			}
		}
		assert.True(t, found, "DNS响应中没有找到正确的端口")

		// 检查额外记录（A记录）
		assert.GreaterOrEqual(t, len(r.Extra), 1, "DNS响应中没有额外记录")

		// 检查额外记录中是否有对应的A记录
		found = false
		for _, extra := range r.Extra {
			if a, ok := extra.(*dnspkg.A); ok {
				if a.A.String() == serviceIP {
					found = true
					break
				}
			}
		}
		assert.True(t, found, "DNS响应中的额外记录没有找到正确的IP地址")
	})

	// 测试多实例服务解析
	t.Run("MultipleInstances", func(t *testing.T) {
		// 注册多个测试服务实例
		const (
			serviceName = "test-multi-instance"
			namespace   = "default"
		)

		instances := []struct {
			id   string
			ip   string
			port int
		}{
			{"test-multi-1", "192.168.1.201", 8091},
			{"test-multi-2", "192.168.1.202", 8092},
			{"test-multi-3", "192.168.1.203", 8093},
		}

		// 注册所有实例
		for _, instance := range instances {
			registerTestService(t, ctx, serviceStore, instance.id, serviceName, namespace, instance.ip, instance.port)
			defer serviceStore.Deregister(ctx, instance.id)
		}

		// 查询A记录
		domain := serviceName + "." + namespace + ".service.local."
		r := doDNSQuery(t, domain, dnspkg.TypeA)

		// 检查响应代码
		assert.Equal(t, dnspkg.RcodeSuccess, r.Rcode, "DNS响应错误码不正确")

		// 检查是否返回了所有实例的A记录
		assert.Equal(t, len(instances), len(r.Answer), "DNS响应中A记录数量不正确")

		// 收集返回的IP地址
		ips := make(map[string]bool)
		for _, ans := range r.Answer {
			if a, ok := ans.(*dnspkg.A); ok {
				ips[a.A.String()] = true
			}
		}

		// 验证所有实例的IP都在结果中
		for _, instance := range instances {
			assert.True(t, ips[instance.ip], "未找到实例IP: "+instance.ip)
		}

		// 查询SRV记录
		r = doDNSQuery(t, domain, dnspkg.TypeSRV)

		// 检查响应代码
		assert.Equal(t, dnspkg.RcodeSuccess, r.Rcode, "DNS响应错误码不正确")

		// 检查是否返回了所有实例的SRV记录
		assert.Equal(t, len(instances), len(r.Answer), "DNS响应中SRV记录数量不正确")

		// 收集返回的端口
		ports := make(map[uint16]bool)
		for _, ans := range r.Answer {
			if srv, ok := ans.(*dnspkg.SRV); ok {
				ports[srv.Port] = true
			}
		}

		// 验证所有实例的端口都在结果中
		for _, instance := range instances {
			assert.True(t, ports[uint16(instance.port)], "未找到实例端口: %d", instance.port)
		}

		// 检查额外记录（A记录）
		assert.Equal(t, len(instances), len(r.Extra), "DNS响应中额外记录数量不正确")

		// 验证所有实例的IP都在额外记录中
		extraIPs := make(map[string]bool)
		for _, extra := range r.Extra {
			if a, ok := extra.(*dnspkg.A); ok {
				extraIPs[a.A.String()] = true
			}
		}

		for _, instance := range instances {
			assert.True(t, extraIPs[instance.ip], "额外记录中未找到实例IP: "+instance.ip)
		}
	})

	// 测试同IP不同端口的服务实例
	t.Run("SameIPDifferentPorts", func(t *testing.T) {
		// 注册多个使用相同IP但不同端口的服务实例
		const (
			serviceName = "test-same-ip"
			namespace   = "default"
			serviceIP   = "192.168.1.50"
		)

		instances := []struct {
			id   string
			port int
		}{
			{"test-same-ip-1", 9001},
			{"test-same-ip-2", 9002},
			{"test-same-ip-3", 9003},
		}

		// 注册所有实例
		for _, instance := range instances {
			registerTestService(t, ctx, serviceStore, instance.id, serviceName, namespace, serviceIP, instance.port)
			defer serviceStore.Deregister(ctx, instance.id)
		}

		// 查询A记录
		domain := serviceName + "." + namespace + ".service.local."
		r := doDNSQuery(t, domain, dnspkg.TypeA)

		// 检查响应代码
		assert.Equal(t, dnspkg.RcodeSuccess, r.Rcode, "DNS响应错误码不正确")

		// 对于A记录，可能每个实例都会返回一个A记录（即使IP相同）
		assert.Equal(t, len(instances), len(r.Answer), "DNS响应中A记录数量不正确")

		// 检查所有A记录都指向相同的IP
		for _, ans := range r.Answer {
			if a, ok := ans.(*dnspkg.A); ok {
				assert.Equal(t, serviceIP, a.A.String(), "DNS响应中的IP不正确")
			}
		}

		// 查询SRV记录
		r = doDNSQuery(t, domain, dnspkg.TypeSRV)

		// 检查响应代码
		assert.Equal(t, dnspkg.RcodeSuccess, r.Rcode, "DNS响应错误码不正确")

		// 检查是否返回了所有实例的SRV记录（每个端口一条）
		assert.Equal(t, len(instances), len(r.Answer), "DNS响应中SRV记录数量不正确")

		// 收集返回的端口
		ports := make(map[uint16]bool)
		for _, ans := range r.Answer {
			if srv, ok := ans.(*dnspkg.SRV); ok {
				ports[srv.Port] = true
			}
		}

		// 验证所有实例的端口都在结果中
		for _, instance := range instances {
			assert.True(t, ports[uint16(instance.port)], "未找到实例端口: %d", instance.port)
		}

		// 检查额外记录（A记录）
		assert.GreaterOrEqual(t, len(r.Extra), 1, "DNS响应中额外记录数量不正确")

		// 检查额外记录中是否包含正确的IP
		for _, extra := range r.Extra {
			if a, ok := extra.(*dnspkg.A); ok {
				assert.Equal(t, serviceIP, a.A.String(), "额外记录中的IP不正确")
			}
		}
	})

	// 测试不存在的服务
	t.Run("NonExistentService", func(t *testing.T) {
		// 查询不存在的服务
		domain := "non-existent-service.default.service.local."
		r := doDNSQuery(t, domain, dnspkg.TypeA)

		// 检查响应代码 - 应该是NXDOMAIN
		assert.Equal(t, dnspkg.RcodeNameError, r.Rcode, "DNS响应错误码不正确")
		assert.Empty(t, r.Answer, "DNS响应中不应该有记录")
	})

	// 测试外部域名上游转发
	t.Run("UpstreamForwarding", func(t *testing.T) {
		// 查询外部域名
		domain := "example.com."
		r := doDNSQuery(t, domain, dnspkg.TypeA)

		// 检查响应代码
		// 注意：上游DNS可能会返回不同的响应码，取决于配置和网络状况
		// 这里我们主要检查是否得到了响应，而不是具体的响应内容
		assert.NotEqual(t, dnspkg.RcodeServerFailure, r.Rcode, "DNS上游转发失败")
	})
}
