package dns

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
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

	// 测试同一IP不同端口的情况
	t.Run("SameIPDifferentPortsTest", func(t *testing.T) {
		// 清理之前的测试数据
		_ = serviceStore.Deregister(ctx, testService.ID)

		// 注册多个使用相同IP但不同端口的服务实例
		sameIP := "192.168.1.50"
		instances := []struct {
			id   string
			port int
		}{
			{"same-ip-service-1", 8001},
			{"same-ip-service-2", 8002},
			{"same-ip-service-3", 8003},
		}

		// 注册服务实例
		for _, instance := range instances {
			svcInstance := &model.Service{
				ID:            instance.id,
				Namespace:     "default",
				Name:          "same-ip-service",
				IP:            sameIP,
				Port:          instance.port,
				Health:        model.HealthStatusHealthy,
				RegisteredAt:  time.Now(),
				LastHeartbeat: time.Now(),
				TTL:           60 * time.Second,
			}

			// 清理可能存在的实例
			_ = serviceStore.Deregister(ctx, svcInstance.ID)

			// 注册服务实例
			if err := serviceStore.Register(ctx, svcInstance); err != nil {
				t.Fatalf("注册服务实例失败[%s]: %v", instance.id, err)
			}

			// 确保测试结束后清理服务
			defer serviceStore.Deregister(ctx, svcInstance.ID)
		}

		// 测试A记录查询
		c := new(dns.Client)
		m := new(dns.Msg)
		m.SetQuestion("same-ip-service.default.service.local.", dns.TypeA)
		m.RecursionDesired = true

		r, _, err := c.Exchange(m, "127.0.0.1:15354")
		if err != nil {
			t.Fatalf("DNS查询失败: %v", err)
		}

		if r.Rcode != dns.RcodeSuccess {
			t.Fatalf("DNS响应错误，代码: %v", r.Rcode)
		}

		// 对于A记录，应该只返回一个IP（因为IP相同）
		// 但在实际应用中DNS服务返回所有实例的IP，即使重复
		if len(r.Answer) != len(instances) {
			t.Fatalf("期望返回%d个A记录，实际返回%d个", len(instances), len(r.Answer))
		}

		// 检查返回的IP是否为预期值
		for _, ans := range r.Answer {
			aRecord, ok := ans.(*dns.A)
			if !ok {
				t.Fatalf("响应不是A记录: %T", ans)
			}
			if aRecord.A.String() != sameIP {
				t.Fatalf("A记录IP错误，期望:%s，实际:%s", sameIP, aRecord.A.String())
			}
		}

		t.Logf("A记录查询正确返回了相同IP: %s", sameIP)

		// 测试SRV记录查询
		m = new(dns.Msg)
		m.SetQuestion("same-ip-service.default.service.local.", dns.TypeSRV)
		m.RecursionDesired = true

		// 添加EDNS0选项，设置更大的缓冲区大小
		opt := new(dns.OPT)
		opt.Hdr.Name = "."
		opt.Hdr.Rrtype = dns.TypeOPT
		opt.SetUDPSize(4096) // 设置更大的UDP缓冲区大小
		m.Extra = append(m.Extra, opt)

		r, _, err = c.Exchange(m, "127.0.0.1:15354")
		if err != nil {
			t.Fatalf("DNS查询失败: %v", err)
		}

		if r.Rcode != dns.RcodeSuccess {
			t.Fatalf("DNS响应错误，代码: %v", r.Rcode)
		}

		// 检查是否返回了所有服务实例的SRV记录
		if len(r.Answer) != len(instances) {
			t.Fatalf("期望返回%d个SRV记录，实际返回%d个", len(instances), len(r.Answer))
		}

		// 检查返回的端口是否包含所有实例的端口
		ports := make(map[uint16]bool)
		for _, ans := range r.Answer {
			srvRecord, ok := ans.(*dns.SRV)
			if !ok {
				t.Fatalf("响应不是SRV记录: %T", ans)
			}
			ports[srvRecord.Port] = true
		}

		// 验证所有期望的端口都存在
		for _, instance := range instances {
			if !ports[uint16(instance.port)] {
				t.Fatalf("返回的SRV记录缺少端口: %d", instance.port)
			}
		}

		t.Logf("SRV记录查询正确返回了所有端口: %v", ports)

		// 检查Extra部分是否包含所有A记录
		if len(r.Extra) != len(instances) {
			t.Fatalf("期望返回%d个额外A记录，实际返回%d个", len(instances), len(r.Extra))
		}

		// 检查所有Extra部分的IP地址是否相同
		for _, extra := range r.Extra {
			aRecord, ok := extra.(*dns.A)
			if !ok {
				t.Fatalf("额外记录不是A记录: %T", extra)
			}
			if aRecord.A.String() != sameIP {
				t.Fatalf("额外A记录IP错误，期望:%s，实际:%s", sameIP, aRecord.A.String())
			}
		}

		t.Logf("额外A记录正确返回了相同IP: %s", sameIP)
	})

	// 测试混合场景：同IP不同端口、不同IP相同端口、不同IP不同端口
	t.Run("MixedServicesTest", func(t *testing.T) {
		// 清理之前的测试数据，确保环境干净
		_ = serviceStore.Deregister(ctx, "same-ip-service-1")
		_ = serviceStore.Deregister(ctx, "same-ip-service-2")
		_ = serviceStore.Deregister(ctx, "same-ip-service-3")

		// 定义测试服务实例
		instances := []struct {
			id   string
			name string
			ip   string
			port int
		}{
			// 场景1: 相同IP(10.0.0.1)，不同端口(7001,7002)
			{"mixed-service-1", "mixed-service", "10.0.0.1", 7001},
			{"mixed-service-2", "mixed-service", "10.0.0.1", 7002},

			// 场景2: 不同IP(10.0.0.2)，相同端口(7001)
			{"mixed-service-3", "mixed-service", "10.0.0.2", 7001},

			// 场景3: 不同IP(10.0.0.3)，不同端口(7003)
			{"mixed-service-4", "mixed-service", "10.0.0.3", 7003},
		}

		// 注册所有服务实例
		for _, instance := range instances {
			svcInstance := &model.Service{
				ID:            instance.id,
				Namespace:     "default",
				Name:          instance.name,
				IP:            instance.ip,
				Port:          instance.port,
				Health:        model.HealthStatusHealthy,
				RegisteredAt:  time.Now(),
				LastHeartbeat: time.Now(),
				TTL:           60 * time.Second,
			}

			// 清理可能存在的实例
			_ = serviceStore.Deregister(ctx, svcInstance.ID)

			// 注册服务实例
			if err := serviceStore.Register(ctx, svcInstance); err != nil {
				t.Fatalf("注册服务实例失败[%s]: %v", instance.id, err)
			}

			// 确保测试结束后清理服务
			defer serviceStore.Deregister(ctx, svcInstance.ID)
		}

		// 测试A记录查询
		c := new(dns.Client)
		m := new(dns.Msg)
		m.SetQuestion("mixed-service.default.service.local.", dns.TypeA)
		m.RecursionDesired = true

		r, _, err := c.Exchange(m, "127.0.0.1:15354")
		if err != nil {
			t.Fatalf("DNS查询失败: %v", err)
		}

		if r.Rcode != dns.RcodeSuccess {
			t.Fatalf("DNS响应错误，代码: %v", r.Rcode)
		}

		// 检查是否返回了所有不同的IP
		if len(r.Answer) != len(instances) {
			t.Fatalf("期望返回%d个A记录，实际返回%d个", len(instances), len(r.Answer))
		}

		// 创建一个map来跟踪返回的IP地址
		ips := make(map[string]bool)
		for _, ans := range r.Answer {
			aRecord, ok := ans.(*dns.A)
			if !ok {
				t.Fatalf("响应不是A记录: %T", ans)
			}
			ips[aRecord.A.String()] = true
		}

		// 检查是否包含所有预期的IP地址
		expectedIPs := map[string]bool{
			"10.0.0.1": true,
			"10.0.0.2": true,
			"10.0.0.3": true,
		}

		for ip := range expectedIPs {
			if !ips[ip] {
				t.Fatalf("缺少预期的IP地址: %s", ip)
			}
		}

		t.Logf("A记录查询成功返回了所有不同的IP地址: %v", ips)

		// 测试SRV记录查询 - 需要增加EDNS0支持以处理较大的响应
		m = new(dns.Msg)
		m.SetQuestion("mixed-service.default.service.local.", dns.TypeSRV)
		m.RecursionDesired = true

		// 添加EDNS0选项，设置更大的缓冲区大小
		opt := new(dns.OPT)
		opt.Hdr.Name = "."
		opt.Hdr.Rrtype = dns.TypeOPT
		opt.SetUDPSize(4096) // 设置更大的UDP缓冲区大小
		m.Extra = append(m.Extra, opt)

		r, _, err = c.Exchange(m, "127.0.0.1:15354")
		if err != nil {
			t.Fatalf("DNS查询失败: %v", err)
		}

		if r.Rcode != dns.RcodeSuccess {
			t.Fatalf("DNS响应错误，代码: %v", r.Rcode)
		}

		// 检查是否返回了所有服务实例的SRV记录
		if len(r.Answer) != len(instances) {
			t.Fatalf("期望返回%d个SRV记录，实际返回%d个", len(instances), len(r.Answer))
		}

		// 创建map来跟踪返回的IP:端口组合
		ipPorts := make(map[string]bool)
		for _, ans := range r.Answer {
			srvRecord, ok := ans.(*dns.SRV)
			if !ok {
				t.Fatalf("响应不是SRV记录: %T", ans)
			}

			// 通过SRV记录的Target找到对应的A记录
			targetIP := ""
			for _, extra := range r.Extra {
				if aRecord, ok := extra.(*dns.A); ok {
					if aRecord.Hdr.Name == srvRecord.Target {
						targetIP = aRecord.A.String()
						break
					}
				}
			}

			if targetIP == "" {
				t.Fatalf("SRV记录缺少对应的A记录: %v", srvRecord)
			}

			ipPort := fmt.Sprintf("%s:%d", targetIP, srvRecord.Port)
			ipPorts[ipPort] = true
		}

		// 检查是否包含所有预期的IP:端口组合
		for _, instance := range instances {
			ipPort := fmt.Sprintf("%s:%d", instance.ip, instance.port)
			if !ipPorts[ipPort] {
				t.Fatalf("缺少预期的IP:端口组合: %s", ipPort)
			}
		}

		t.Logf("SRV记录查询成功返回了所有IP:端口组合: %v", ipPorts)

		// 检查相同IP、不同端口的情况
		sameIPPorts := 0
		for ipPort := range ipPorts {
			if strings.HasPrefix(ipPort, "10.0.0.1:") {
				sameIPPorts++
			}
		}

		if sameIPPorts != 2 {
			t.Fatalf("相同IP(10.0.0.1)下应有2个不同端口，实际有%d个", sameIPPorts)
		}

		// 检查不同IP、相同端口的情况
		port7001Count := 0
		for ipPort := range ipPorts {
			if strings.HasSuffix(ipPort, ":7001") {
				port7001Count++
			}
		}

		if port7001Count != 2 {
			t.Fatalf("相同端口(7001)下应有2个不同IP，实际有%d个", port7001Count)
		}

		t.Logf("成功验证了同IP不同端口、不同IP相同端口和不同IP不同端口的所有场景")
	})

	// 关闭服务器
	if err := server.Stop(); err != nil {
		t.Fatalf("停止DNS服务器失败: %v", err)
	}
}
