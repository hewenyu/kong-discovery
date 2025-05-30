package dns

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/hewenyu/kong-discovery/internal/core/config"
	"github.com/hewenyu/kong-discovery/internal/core/model"
	"github.com/hewenyu/kong-discovery/internal/store/etcd"
	"github.com/hewenyu/kong-discovery/internal/store/service"
	"github.com/miekg/dns"
)

func getEtcdClient() (*etcd.Client, error) {
	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
	if etcdEndpoints == "" {
		return nil, errors.New("ETCD_ENDPOINTS 未设置")
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
	testKey := "/test/dns-test-connection"
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

func TestDNSServer(t *testing.T) {
	// 使用非标准端口以避免需要root权限
	config := DefaultConfig()
	config.DNSAddr = "127.0.0.1:15353"

	// 创建并启动DNS服务器
	server := NewServer(config)
	ctx := context.Background()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("启动DNS服务器失败: %v", err)
	}

	// 确保服务器有时间启动
	time.Sleep(500 * time.Millisecond)

	// 创建DNS客户端并测试查询
	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion("test.service.local.", dns.TypeA)
	m.RecursionDesired = true

	r, _, err := c.Exchange(m, "127.0.0.1:15353")
	if err != nil {
		t.Fatalf("DNS查询失败: %v", err)
	}

	// 检查是否收到响应
	if r == nil {
		t.Fatal("未收到DNS响应")
	}

	// 检查响应代码
	if r.Rcode != dns.RcodeSuccess {
		t.Fatalf("DNS响应错误，代码: %v", r.Rcode)
	}

	// 检查是否有回答
	if len(r.Answer) == 0 {
		t.Fatal("DNS响应中没有回答")
	}

	// 检查A记录
	aRecord, ok := r.Answer[0].(*dns.A)
	if !ok {
		t.Fatalf("响应不是A记录: %T", r.Answer[0])
	}

	// 检查IP地址是否为127.0.0.1（默认硬编码响应）
	if aRecord.A.String() != "127.0.0.1" {
		t.Fatalf("A记录IP错误，期望:127.0.0.1，实际:%s", aRecord.A.String())
	}

	// 关闭服务器
	if err := server.Stop(); err != nil {
		t.Fatalf("停止DNS服务器失败: %v", err)
	}
}

func TestDNSServerWithRealServiceStore(t *testing.T) {
	// 获取etcd客户端，如果连接失败则跳过测试
	etcdClient := skipIfNoEtcd(t)
	if etcdClient == nil {
		return
	}
	defer etcdClient.Close()

	ctx := context.Background()

	// 创建真实的服务存储
	serviceStore := service.NewEtcdServiceStore(etcdClient, "default")

	// 注册测试服务
	testService := &model.Service{
		ID:            "test-service-real-1",
		Namespace:     "default",
		Name:          "test-service-real",
		IP:            "192.168.1.100",
		Port:          8080,
		Health:        model.HealthStatusHealthy,
		RegisteredAt:  time.Now(),
		LastHeartbeat: time.Now(),
		TTL:           60 * time.Second,
	}

	// 清理可能存在的测试服务
	_ = serviceStore.Deregister(ctx, testService.ID)

	// 注册测试服务
	if err := serviceStore.Register(ctx, testService); err != nil {
		t.Fatalf("注册测试服务失败: %v", err)
	}
	// 确保测试结束后清理服务
	defer serviceStore.Deregister(ctx, testService.ID)

	// 使用非标准端口以避免需要root权限
	config := DefaultConfig()
	config.DNSAddr = "127.0.0.1:15354"
	config.ServiceStore = serviceStore

	// 创建并启动DNS服务器
	server := NewServer(config)

	if err := server.Start(ctx); err != nil {
		t.Fatalf("启动DNS服务器失败: %v", err)
	}

	// 确保服务器有时间启动
	time.Sleep(500 * time.Millisecond)

	// 测试A记录查询
	t.Run("ARecordQuery", func(t *testing.T) {
		c := new(dns.Client)
		m := new(dns.Msg)
		m.SetQuestion("test-service-real.default.service.local.", dns.TypeA)
		m.RecursionDesired = true

		r, _, err := c.Exchange(m, "127.0.0.1:15354")
		if err != nil {
			t.Fatalf("DNS查询失败: %v", err)
		}

		// 检查是否收到响应
		if r == nil {
			t.Fatal("未收到DNS响应")
		}

		// 检查响应代码
		if r.Rcode != dns.RcodeSuccess {
			t.Fatalf("DNS响应错误，代码: %v", r.Rcode)
		}

		// 检查是否有回答
		if len(r.Answer) == 0 {
			t.Fatal("DNS响应中没有回答")
		}

		// 检查A记录
		aRecord, ok := r.Answer[0].(*dns.A)
		if !ok {
			t.Fatalf("响应不是A记录: %T", r.Answer[0])
		}

		// 检查IP地址是否为测试服务的IP
		if aRecord.A.String() != testService.IP {
			t.Fatalf("A记录IP错误，期望:%s，实际:%s", testService.IP, aRecord.A.String())
		}
	})

	// 测试SRV记录查询
	t.Run("SRVRecordQuery", func(t *testing.T) {
		c := new(dns.Client)
		m := new(dns.Msg)
		m.SetQuestion("test-service-real.default.service.local.", dns.TypeSRV)
		m.RecursionDesired = true

		r, _, err := c.Exchange(m, "127.0.0.1:15354")
		if err != nil {
			t.Fatalf("DNS查询失败: %v", err)
		}

		// 检查是否收到响应
		if r == nil {
			t.Fatal("未收到DNS响应")
		}

		// 检查响应代码
		if r.Rcode != dns.RcodeSuccess {
			t.Fatalf("DNS响应错误，代码: %v", r.Rcode)
		}

		// 检查是否有回答
		if len(r.Answer) == 0 {
			t.Fatal("DNS响应中没有回答")
		}

		// 检查SRV记录
		srvRecord, ok := r.Answer[0].(*dns.SRV)
		if !ok {
			t.Fatalf("响应不是SRV记录: %T", r.Answer[0])
		}

		// 检查端口是否为测试服务的端口
		if srvRecord.Port != uint16(testService.Port) {
			t.Fatalf("SRV记录端口错误，期望:%d，实际:%d", testService.Port, srvRecord.Port)
		}

		// 检查附加区段（应包含A记录）
		if len(r.Extra) == 0 {
			t.Fatal("DNS响应中没有附加记录")
		}

		// 检查附加A记录
		aRecord, ok := r.Extra[0].(*dns.A)
		if !ok {
			t.Fatalf("附加记录不是A记录: %T", r.Extra[0])
		}

		// 检查附加A记录的IP是否为测试服务的IP
		if aRecord.A.String() != testService.IP {
			t.Fatalf("附加A记录IP错误，期望:%s，实际:%s", testService.IP, aRecord.A.String())
		}
	})

	// 测试返回所有服务实例 - 注册多个服务实例
	t.Run("AllServicesReturnedTest", func(t *testing.T) {
		// 注册第二个测试服务实例
		testService2 := &model.Service{
			ID:            "test-service-real-2",
			Namespace:     "default",
			Name:          "test-service-real",
			IP:            "192.168.1.200",
			Port:          8081,
			Health:        model.HealthStatusHealthy,
			RegisteredAt:  time.Now(),
			LastHeartbeat: time.Now(),
			TTL:           60 * time.Second,
		}

		// 清理可能存在的测试服务
		_ = serviceStore.Deregister(ctx, testService2.ID)

		// 注册第二个测试服务
		if err := serviceStore.Register(ctx, testService2); err != nil {
			t.Fatalf("注册第二个测试服务失败: %v", err)
		}
		// 确保测试结束后清理服务
		defer serviceStore.Deregister(ctx, testService2.ID)

		// 测试A记录是否返回所有服务实例
		c := new(dns.Client)
		m := new(dns.Msg)
		m.SetQuestion("test-service-real.default.service.local.", dns.TypeA)
		m.RecursionDesired = true

		r, _, err := c.Exchange(m, "127.0.0.1:15354")
		if err != nil {
			t.Fatalf("DNS查询失败: %v", err)
		}

		if r.Rcode != dns.RcodeSuccess {
			t.Fatalf("DNS响应错误，代码: %v", r.Rcode)
		}

		// 检查是否返回了两个A记录（对应两个服务实例）
		if len(r.Answer) != 2 {
			t.Fatalf("期望返回2个A记录，实际返回%d个", len(r.Answer))
		}

		// 检查返回的IP地址是否包含我们注册的两个服务实例
		ips := make(map[string]bool)
		for _, ans := range r.Answer {
			aRecord, ok := ans.(*dns.A)
			if !ok {
				t.Fatalf("响应不是A记录: %T", ans)
			}
			ips[aRecord.A.String()] = true
		}

		if !ips[testService.IP] || !ips[testService2.IP] {
			t.Fatalf("返回的A记录不包含所有服务实例的IP，找到的IP: %v", ips)
		} else {
			t.Logf("返回的A记录包含所有服务实例的IP: %v", ips)
		}

		// 测试SRV记录是否返回所有服务实例
		m = new(dns.Msg)
		m.SetQuestion("test-service-real.default.service.local.", dns.TypeSRV)
		m.RecursionDesired = true

		r, _, err = c.Exchange(m, "127.0.0.1:15354")
		if err != nil {
			t.Fatalf("DNS查询失败: %v", err)
		}

		if r.Rcode != dns.RcodeSuccess {
			t.Fatalf("DNS响应错误，代码: %v", r.Rcode)
		}

		// 检查是否返回了两个SRV记录（对应两个服务实例）
		if len(r.Answer) != 2 {
			t.Fatalf("期望返回2个SRV记录，实际返回%d个", len(r.Answer))
		}

		// 检查返回的端口是否包含我们注册的两个服务实例
		ports := make(map[uint16]bool)
		for _, ans := range r.Answer {
			srvRecord, ok := ans.(*dns.SRV)
			if !ok {
				t.Fatalf("响应不是SRV记录: %T", ans)
			}
			ports[srvRecord.Port] = true
		}

		if !ports[uint16(testService.Port)] || !ports[uint16(testService2.Port)] {
			t.Fatalf("返回的SRV记录不包含所有服务实例的端口，找到的端口: %v", ports)
		} else {
			t.Logf("返回的SRV记录包含所有服务实例的端口: %v", ports)
		}

		// 检查Extra部分是否包含所有A记录
		if len(r.Extra) != 2 {
			t.Fatalf("期望返回2个额外A记录，实际返回%d个", len(r.Extra))
		}

		// 检查Extra部分的IP地址
		extraIPs := make(map[string]bool)
		for _, extra := range r.Extra {
			aRecord, ok := extra.(*dns.A)
			if !ok {
				t.Fatalf("额外记录不是A记录: %T", extra)
			}
			extraIPs[aRecord.A.String()] = true
		}

		if !extraIPs[testService.IP] || !extraIPs[testService2.IP] {
			t.Fatalf("额外A记录不包含所有服务实例的IP，找到的IP: %v", extraIPs)
		} else {
			t.Logf("额外A记录包含所有服务实例的IP: %v", extraIPs)
		}
	})

	// 关闭服务器
	if err := server.Stop(); err != nil {
		t.Fatalf("停止DNS服务器失败: %v", err)
	}
}
