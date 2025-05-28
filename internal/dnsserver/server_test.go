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
)

// 创建一个测试用的配置，使用环境变量中的etcd地址
func createTestConfig(t *testing.T) *config.Config {
	t.Helper()

	// 从环境变量中获取etcd地址
	etcdEndpoints := os.Getenv("KONG_DISCOVERY_ETCD_ENDPOINTS")
	if etcdEndpoints == "" {
		t.Skip("跳过测试：未设置KONG_DISCOVERY_ETCD_ENDPOINTS环境变量")
	}

	// 创建配置
	cfg := &config.Config{}
	cfg.Etcd.Endpoints = []string{etcdEndpoints}
	cfg.Etcd.Username = os.Getenv("KONG_DISCOVERY_ETCD_USERNAME") // 如果需要认证，获取用户名
	cfg.Etcd.Password = os.Getenv("KONG_DISCOVERY_ETCD_PASSWORD") // 如果需要认证，获取密码
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

func TestDNSServerDynamicServiceDiscovery(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建一个测试配置
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建真实的etcd客户端
	client := etcdclient.CreateEtcdClientForTest(t)
	defer client.Close()

	// 创建DNS服务器
	server := NewDNSServer(cfg, logger)

	// 设置etcd客户端
	server.SetEtcdClient(client)

	// 测试缓存更新
	// 1. 创建测试服务实例
	testService := &etcdclient.ServiceInstance{
		ServiceName: "test-service",
		InstanceID:  "test-instance-1",
		IPAddress:   "192.168.1.1",
		Port:        8080,
		TTL:         60,
	}

	// 2. 注册测试服务实例
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := client.RegisterService(ctx, testService)
	cancel()
	require.NoError(t, err, "注册测试服务实例失败")

	// 等待服务注册事件被处理
	time.Sleep(100 * time.Millisecond)

	// 3. 更新缓存
	server.UpdateServiceCache(testService)

	// 4. 验证缓存是否被正确更新
	dnsServer := server.(*DNSServer)

	dnsServer.cacheMutex.RLock()
	instances, ok := dnsServer.serviceCache["test-service"]
	dnsServer.cacheMutex.RUnlock()

	assert.True(t, ok, "服务缓存应该包含test-service")
	assert.Equal(t, 1, len(instances), "test-service应该有1个实例")
	assert.Equal(t, testService, instances["test-instance-1"], "缓存的服务实例应该与原始实例相同")

	// 5. 测试DNS查询
	// 创建一个DNS查询消息
	request := new(dns.Msg)
	request.SetQuestion("test-service.default.svc.cluster.local.", dns.TypeA)

	// 创建一个响应消息
	response := new(dns.Msg)
	response.SetReply(request)

	// 处理查询
	found := dnsServer.handleQuery(request.Question[0], response)

	// 验证查询结果
	assert.True(t, found, "应该找到DNS记录")
	assert.Equal(t, 1, len(response.Answer), "应该有1个答案")

	// 验证A记录
	aRecord := response.Answer[0]
	assert.Equal(t, "test-service.default.svc.cluster.local.", aRecord.Header().Name, "域名应该匹配")
	assert.Equal(t, dns.TypeA, aRecord.Header().Rrtype, "记录类型应该是A")

	// 提取IP地址
	aRecordA, ok := aRecord.(*dns.A)
	assert.True(t, ok, "记录应该是A类型")
	assert.Equal(t, "192.168.1.1", aRecordA.A.String(), "IP地址应该匹配")

	// 6. 测试服务删除
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	err = client.DeregisterService(ctx2, "test-service", "test-instance-1")
	cancel2()
	require.NoError(t, err, "注销测试服务实例失败")

	// 等待服务注销事件被处理
	time.Sleep(100 * time.Millisecond)

	// 手动从缓存中删除服务
	server.RemoveServiceFromCache("test-service", "test-instance-1")

	dnsServer.cacheMutex.RLock()
	_, ok = dnsServer.serviceCache["test-service"]
	dnsServer.cacheMutex.RUnlock()

	assert.False(t, ok, "服务缓存应该不再包含test-service")

	// 7. 测试DNS记录缓存
	testDNSRecord := &etcdclient.DNSRecord{
		Type:  "A",
		Value: "192.168.1.100",
		TTL:   300,
	}

	// 将DNS记录保存到etcd
	ctx3, cancel3 := context.WithTimeout(context.Background(), 5*time.Second)
	err = client.PutDNSRecord(ctx3, "example.com", testDNSRecord)
	cancel3()
	require.NoError(t, err, "保存DNS记录失败")

	// 等待DNS记录事件被处理
	time.Sleep(100 * time.Millisecond)

	// 更新缓存
	server.UpdateCache("example.com", "A", testDNSRecord)

	// 验证缓存
	dnsServer.cacheMutex.RLock()
	records, ok := dnsServer.dnsCache["example.com"]
	dnsServer.cacheMutex.RUnlock()

	assert.True(t, ok, "DNS缓存应该包含example.com")
	assert.Equal(t, 1, len(records), "example.com应该有1个记录")
	assert.Equal(t, testDNSRecord, records["A"], "缓存的DNS记录应该与原始记录相同")

	// 创建一个DNS查询消息
	request = new(dns.Msg)
	request.SetQuestion("example.com.", dns.TypeA)

	// 创建一个响应消息
	response = new(dns.Msg)
	response.SetReply(request)

	// 处理查询
	found = dnsServer.handleQuery(request.Question[0], response)

	// 验证查询结果
	assert.True(t, found, "应该找到DNS记录")
	assert.Equal(t, 1, len(response.Answer), "应该有1个答案")

	// 验证A记录
	aRecord = response.Answer[0]
	assert.Equal(t, "example.com.", aRecord.Header().Name, "域名应该匹配")
	assert.Equal(t, dns.TypeA, aRecord.Header().Rrtype, "记录类型应该是A")

	// 8. 测试DNS记录删除
	// 直接从etcd中删除记录
	ctx4, cancel4 := context.WithTimeout(context.Background(), 5*time.Second)
	etcdClientImpl := client.(*etcdclient.EtcdClient)
	_, err = etcdClientImpl.Client().Delete(ctx4, "/dns/records/example.com/A")
	cancel4()
	require.NoError(t, err, "删除DNS记录失败")

	// 等待DNS记录删除事件被处理
	time.Sleep(100 * time.Millisecond)

	// 手动从缓存中删除记录
	server.RemoveFromCache("example.com", "A")

	dnsServer.cacheMutex.RLock()
	_, ok = dnsServer.dnsCache["example.com"]
	dnsServer.cacheMutex.RUnlock()

	assert.False(t, ok, "DNS缓存应该不再包含example.com")
}

func TestDNSServerWatchEvents(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建一个测试配置
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建真实的etcd客户端
	client := etcdclient.CreateEtcdClientForTest(t)
	defer client.Close()

	// 创建并启动DNS服务器
	server := NewDNSServer(cfg, logger)
	dnsServer := server.(*DNSServer)

	// 设置etcd客户端
	server.SetEtcdClient(client)

	// 启动etcd监听
	dnsServer.startEtcdWatcher()

	// 确保watcher已启动
	require.True(t, dnsServer.watcherStarted, "watcher应该已启动")

	// 1. 测试DNS记录创建事件
	testDNSRecord := &etcdclient.DNSRecord{
		Type:  "A",
		Value: "192.168.1.100",
		TTL:   300,
	}

	ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	err := client.PutDNSRecord(ctx1, "example.com", testDNSRecord)
	cancel1()
	require.NoError(t, err, "保存DNS记录失败")

	// 等待DNS记录事件被处理
	time.Sleep(500 * time.Millisecond)

	// 验证缓存是否更新
	dnsServer.cacheMutex.RLock()
	records, ok := dnsServer.dnsCache["example.com"]
	dnsServer.cacheMutex.RUnlock()

	assert.True(t, ok, "DNS缓存应该包含example.com")
	assert.Equal(t, 1, len(records), "example.com应该有1个记录")
	assert.Equal(t, "A", records["A"].Type, "记录类型应该是A")
	assert.Equal(t, "192.168.1.100", records["A"].Value, "记录值应该是192.168.1.100")

	// 2. 测试服务创建事件
	serviceInstance := &etcdclient.ServiceInstance{
		ServiceName: "test-service",
		InstanceID:  "test-instance-1",
		IPAddress:   "192.168.1.1",
		Port:        8080,
		TTL:         60,
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	err = client.RegisterService(ctx2, serviceInstance)
	cancel2()
	require.NoError(t, err, "注册服务失败")

	// 等待服务事件被处理
	time.Sleep(500 * time.Millisecond)

	// 验证缓存是否更新
	dnsServer.cacheMutex.RLock()
	instances, ok := dnsServer.serviceCache["test-service"]
	dnsServer.cacheMutex.RUnlock()

	assert.True(t, ok, "服务缓存应该包含test-service")
	assert.Equal(t, 1, len(instances), "test-service应该有1个实例")
	assert.Equal(t, "test-instance-1", instances["test-instance-1"].InstanceID, "实例ID应该匹配")
	assert.Equal(t, "192.168.1.1", instances["test-instance-1"].IPAddress, "IP地址应该匹配")

	// 3. 测试删除事件
	ctx3, cancel3 := context.WithTimeout(context.Background(), 5*time.Second)
	err = client.DeregisterService(ctx3, "test-service", "test-instance-1")
	cancel3()
	require.NoError(t, err, "注销服务失败")

	// 等待删除事件被处理
	time.Sleep(500 * time.Millisecond)

	// 验证缓存是否更新
	dnsServer.cacheMutex.RLock()
	_, ok = dnsServer.serviceCache["test-service"]
	dnsServer.cacheMutex.RUnlock()

	assert.False(t, ok, "服务缓存应该不再包含test-service")

	// 4. 测试DNS记录删除
	ctx4, cancel4 := context.WithTimeout(context.Background(), 5*time.Second)
	etcdClientImpl := client.(*etcdclient.EtcdClient)
	_, err = etcdClientImpl.Client().Delete(ctx4, "/dns/records/example.com/A")
	cancel4()
	require.NoError(t, err, "删除DNS记录失败")

	// 等待DNS记录删除事件被处理
	time.Sleep(500 * time.Millisecond)

	// 验证缓存是否更新
	dnsServer.cacheMutex.RLock()
	_, ok = dnsServer.dnsCache["example.com"]
	dnsServer.cacheMutex.RUnlock()

	assert.False(t, ok, "DNS缓存应该不再包含example.com")
}

func TestDNSServer_DynamicServiceDiscovery_Integration(t *testing.T) {
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

	// 创建并启动DNS服务器
	server := NewDNSServer(cfg, logger)
	server.SetEtcdClient(client)
	err := server.Start()
	require.NoError(t, err, "启动DNS服务器失败")
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	// 等待服务器启动和Watcher初始化
	time.Sleep(500 * time.Millisecond)

	// 创建一个唯一的测试服务名，避免测试冲突
	testServiceName := fmt.Sprintf("test-service-%d", time.Now().UnixNano())
	testInstanceID := "instance-001"

	// 注册测试服务实例
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testService := &etcdclient.ServiceInstance{
		ServiceName: testServiceName,
		InstanceID:  testInstanceID,
		IPAddress:   "10.0.0.1",
		Port:        8080,
		TTL:         60,
		Metadata: map[string]string{
			"version": "1.0.0",
			"env":     "test",
		},
	}

	err = client.RegisterService(ctx, testService)
	require.NoError(t, err, "注册测试服务实例失败")

	// 等待Watcher处理事件
	time.Sleep(1 * time.Second)

	// 验证DNS解析 (A记录)
	dnsClient := new(dns.Client)
	queryMsg := new(dns.Msg)
	queryMsg.SetQuestion(fmt.Sprintf("%s.default.svc.cluster.local.", testServiceName), dns.TypeA)
	queryMsg.RecursionDesired = true

	r, _, err := dnsClient.Exchange(queryMsg, fmt.Sprintf("%s:%d", cfg.DNS.ListenAddress, cfg.DNS.Port))
	require.NoError(t, err, "DNS查询失败")
	require.NotNil(t, r, "DNS响应不应为nil")

	// 验证DNS响应
	assert.Equal(t, dns.RcodeSuccess, r.Rcode, "DNS响应代码应为成功")
	assert.GreaterOrEqual(t, len(r.Answer), 1, "应至少有一个答案")

	// 验证A记录
	found := false
	for _, ans := range r.Answer {
		if a, ok := ans.(*dns.A); ok {
			assert.Equal(t, "10.0.0.1", a.A.String(), "IP地址应匹配")
			found = true
			break
		}
	}
	assert.True(t, found, "应找到A记录")

	// 验证SRV记录
	srvQueryMsg := new(dns.Msg)
	srvQueryMsg.SetQuestion(fmt.Sprintf("%s.default.svc.cluster.local.", testServiceName), dns.TypeSRV)
	srvQueryMsg.RecursionDesired = true

	srvR, _, err := dnsClient.Exchange(srvQueryMsg, fmt.Sprintf("%s:%d", cfg.DNS.ListenAddress, cfg.DNS.Port))
	require.NoError(t, err, "SRV查询失败")
	require.NotNil(t, srvR, "SRV响应不应为nil")

	// 验证SRV响应
	assert.Equal(t, dns.RcodeSuccess, srvR.Rcode, "SRV响应代码应为成功")
	assert.GreaterOrEqual(t, len(srvR.Answer), 1, "应至少有一个SRV答案")

	// 验证SRV记录
	found = false
	for _, ans := range srvR.Answer {
		if srv, ok := ans.(*dns.SRV); ok {
			assert.Equal(t, 8080, int(srv.Port), "端口应匹配")
			found = true
			break
		}
	}
	assert.True(t, found, "应找到SRV记录")

	// 测试修改服务
	testService.IPAddress = "10.0.0.2"
	testService.Port = 8081

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	err = client.RegisterService(ctx2, testService)
	require.NoError(t, err, "更新测试服务实例失败")

	// 等待Watcher处理事件
	time.Sleep(1 * time.Second)

	// 再次查询A记录，验证IP已更新
	r2, _, err := dnsClient.Exchange(queryMsg, fmt.Sprintf("%s:%d", cfg.DNS.ListenAddress, cfg.DNS.Port))
	require.NoError(t, err, "更新后的DNS查询失败")
	require.NotNil(t, r2, "更新后的DNS响应不应为nil")

	// 验证更新后的DNS响应
	found = false
	for _, ans := range r2.Answer {
		if a, ok := ans.(*dns.A); ok {
			assert.Equal(t, "10.0.0.2", a.A.String(), "更新后的IP地址应匹配")
			found = true
			break
		}
	}
	assert.True(t, found, "应找到更新后的A记录")

	// 测试删除服务
	ctx3, cancel3 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel3()

	err = client.DeregisterService(ctx3, testServiceName, testInstanceID)
	require.NoError(t, err, "注销测试服务实例失败")

	// 等待Watcher处理事件
	time.Sleep(1 * time.Second)

	// 再次查询，验证服务已删除
	r3, _, err := dnsClient.Exchange(queryMsg, fmt.Sprintf("%s:%d", cfg.DNS.ListenAddress, cfg.DNS.Port))
	require.NoError(t, err, "删除后的DNS查询失败")
	require.NotNil(t, r3, "删除后的DNS响应不应为nil")

	// 因为已经删除服务，应该没有找到记录
	assert.Equal(t, dns.RcodeNameError, r3.Rcode, "删除后的DNS响应代码应为NXDOMAIN")
	assert.Equal(t, 0, len(r3.Answer), "删除后不应该有答案")
}

func TestDNSServer_DynamicDNSRecords_Integration(t *testing.T) {
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

	// 创建并启动DNS服务器
	server := NewDNSServer(cfg, logger)
	server.SetEtcdClient(client)
	err := server.Start()
	require.NoError(t, err, "启动DNS服务器失败")
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	// 等待服务器启动和Watcher初始化
	time.Sleep(500 * time.Millisecond)

	// 创建一个唯一的测试域名，避免测试冲突
	testDomain := fmt.Sprintf("test-domain-%d.com", time.Now().UnixNano())

	// 创建测试DNS记录
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testRecord := &etcdclient.DNSRecord{
		Type:  "A",
		Value: "192.168.1.1",
		TTL:   300,
	}

	err = client.PutDNSRecord(ctx, testDomain, testRecord)
	require.NoError(t, err, "创建测试DNS记录失败")

	// 等待Watcher处理事件
	time.Sleep(1 * time.Second)

	// 验证DNS解析
	dnsClient := new(dns.Client)
	queryMsg := new(dns.Msg)
	queryMsg.SetQuestion(fmt.Sprintf("%s.", testDomain), dns.TypeA)
	queryMsg.RecursionDesired = true

	r, _, err := dnsClient.Exchange(queryMsg, fmt.Sprintf("%s:%d", cfg.DNS.ListenAddress, cfg.DNS.Port))
	require.NoError(t, err, "DNS查询失败")
	require.NotNil(t, r, "DNS响应不应为nil")

	// 验证DNS响应
	assert.Equal(t, dns.RcodeSuccess, r.Rcode, "DNS响应代码应为成功")
	assert.GreaterOrEqual(t, len(r.Answer), 1, "应至少有一个答案")

	// 验证A记录
	found := false
	for _, ans := range r.Answer {
		if a, ok := ans.(*dns.A); ok {
			assert.Equal(t, "192.168.1.1", a.A.String(), "IP地址应匹配")
			found = true
			break
		}
	}
	assert.True(t, found, "应找到A记录")

	// 更新测试DNS记录
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	testRecord.Value = "192.168.1.2"
	err = client.PutDNSRecord(ctx2, testDomain, testRecord)
	require.NoError(t, err, "更新测试DNS记录失败")

	// 等待Watcher处理事件
	time.Sleep(1 * time.Second)

	// 再次查询A记录，验证IP已更新
	r2, _, err := dnsClient.Exchange(queryMsg, fmt.Sprintf("%s:%d", cfg.DNS.ListenAddress, cfg.DNS.Port))
	require.NoError(t, err, "更新后的DNS查询失败")
	require.NotNil(t, r2, "更新后的DNS响应不应为nil")

	// 验证更新后的DNS响应
	found = false
	for _, ans := range r2.Answer {
		if a, ok := ans.(*dns.A); ok {
			assert.Equal(t, "192.168.1.2", a.A.String(), "更新后的IP地址应匹配")
			found = true
			break
		}
	}
	assert.True(t, found, "应找到更新后的A记录")

	// 测试删除DNS记录 (模拟删除操作，因为etcdclient.Client接口没有提供删除DNS记录的方法)
	// 使用etcd客户端直接删除键
	ctx3, cancel3 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel3()

	// 获取原始etcdClient
	etcdClientImpl, ok := client.(*etcdclient.EtcdClient)
	require.True(t, ok, "client应为EtcdClient类型")

	// 直接删除DNS记录键
	key := fmt.Sprintf("/dns/records/%s/A", testDomain)
	_, err = etcdClientImpl.Client().Delete(ctx3, key)
	require.NoError(t, err, "删除DNS记录失败")

	// 等待Watcher处理事件
	time.Sleep(1 * time.Second)

	// 再次查询，验证记录已删除
	r3, _, err := dnsClient.Exchange(queryMsg, fmt.Sprintf("%s:%d", cfg.DNS.ListenAddress, cfg.DNS.Port))
	require.NoError(t, err, "删除后的DNS查询失败")
	require.NotNil(t, r3, "删除后的DNS响应不应为nil")

	// 因为已经删除记录，应该没有找到记录
	assert.Equal(t, dns.RcodeNameError, r3.Rcode, "删除后的DNS响应代码应为NXDOMAIN")
	assert.Equal(t, 0, len(r3.Answer), "删除后不应该有答案")
}
