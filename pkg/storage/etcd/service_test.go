package etcd

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hewenyu/kong-discovery/pkg/config"
	"github.com/hewenyu/kong-discovery/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 这些测试主要验证接口签名和基础逻辑，而不是与真实etcd的交互
// 真实的etcd交互测试需要在集成测试中进行

func TestEtcdServiceStorage_Implements_ServiceStorage(t *testing.T) {
	// 确保ServiceStorage实现了storage.ServiceStorage接口
	// 这是一个编译时检查
	var _ storage.ServiceStorage = (*ServiceStorage)(nil)
}

func TestServiceStorage_Basic(t *testing.T) {
	// 测试Client方法
	client := &Client{
		servicePrefix:   "/kong-discovery/services/",
		namespacePrefix: "/kong-discovery/namespaces/",
	}

	// 测试获取服务键
	key := client.GetServiceKey("test-service")
	assert.Equal(t, "/kong-discovery/services/test-service", key)

	// 测试获取服务前缀
	prefix := client.GetServicesPrefix()
	assert.Equal(t, "/kong-discovery/services/", prefix)

	// 测试带命名空间的服务键
	nsKey := client.GetNamespacedServiceKey("test-ns", "test-service")
	assert.Equal(t, "/kong-discovery/services/test-ns/test-service", nsKey)

	// 测试命名空间前缀
	nsPrefix := client.GetNamespaceServicesPrefix("test-ns")
	assert.Equal(t, "/kong-discovery/services/test-ns/", nsPrefix)
}

// 检查是否有可用的etcd环境
func hasEtcdEnvironment() bool {
	return os.Getenv("ETCD_ENDPOINTS") != ""
}

// 从环境变量获取etcd配置
func getEtcdConfigFromEnv() *config.EtcdConfig {
	endpoints := os.Getenv("ETCD_ENDPOINTS")
	if endpoints == "" {
		endpoints = "localhost:2379" // 默认地址
	}

	// 设置更长的超时时间
	dialTimeout := os.Getenv("ETCD_DIAL_TIMEOUT")
	if dialTimeout == "" {
		dialTimeout = "10s" // 更长的超时时间
	}

	username := os.Getenv("ETCD_USERNAME")
	password := os.Getenv("ETCD_PASSWORD")

	return &config.EtcdConfig{
		Endpoints:   strings.Split(endpoints, ","),
		DialTimeout: dialTimeout,
		Username:    username,
		Password:    password,
	}
}

// 这些测试需要一个运行中的etcd实例
// 如果没有设置ETCD_ENDPOINTS环境变量，测试将被跳过

func TestEtcdServiceStorage_IntegrationTest(t *testing.T) {
	if !hasEtcdEnvironment() {
		t.Skip("跳过etcd集成测试 - 未设置ETCD_ENDPOINTS环境变量")
	}

	// 创建etcd客户端
	cfg := getEtcdConfigFromEnv()
	client, err := NewClient(cfg)
	require.NoError(t, err, "创建etcd客户端失败")
	defer client.Close()

	// 创建服务存储
	serviceStorage := NewServiceStorage(client)

	// 创建上下文，延长超时时间到30分钟
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// 先测试连接是否正常
	_, err = client.GetClient().Status(ctx, cfg.Endpoints[0])
	if err != nil {
		t.Skipf("etcd连接测试失败，跳过测试: %v", err)
		return
	}

	// 清理可能的测试数据
	cleanupTestData(t, ctx, serviceStorage)

	// 运行测试
	t.Run("RegisterAndGetService", func(t *testing.T) {
		testRegisterAndGetService(t, ctx, serviceStorage)
	})

	t.Run("ListServices", func(t *testing.T) {
		testListServices(t, ctx, serviceStorage)
	})

	t.Run("UpdateHeartbeat", func(t *testing.T) {
		testUpdateHeartbeat(t, ctx, serviceStorage)
	})

	t.Run("DeregisterService", func(t *testing.T) {
		testDeregisterService(t, ctx, serviceStorage)
	})

	t.Run("CleanupStaleServices", func(t *testing.T) {
		testCleanupStaleServices(t, ctx, serviceStorage)
	})

	// 添加命名空间特性测试
	t.Run("NamespaceFeatures", func(t *testing.T) {
		testNamespaceFeatures(t, ctx, serviceStorage)
	})

	// 测试完成后清理数据
	cleanupTestData(t, ctx, serviceStorage)
}

func cleanupTestData(t *testing.T, ctx context.Context, s *ServiceStorage) {
	// 获取所有服务
	services, err := s.ListServices(ctx)
	if err != nil {
		t.Logf("获取服务列表失败: %v", err)
		return
	}

	// 删除所有以"test-"开头的服务
	for _, service := range services {
		if len(service.ID) >= 5 && service.ID[:5] == "test-" {
			err := s.DeregisterService(ctx, service.ID)
			if err != nil {
				t.Logf("删除测试服务失败 %s: %v", service.ID, err)
			}
		}
	}
}

// 创建测试命名空间
func createTestNamespace(t *testing.T, ctx context.Context, client *Client, namespaceName string) {
	nsStorage := NewNamespaceStorage(client)
	namespace := &storage.Namespace{
		Name:        namespaceName,
		Description: "测试命名空间",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := nsStorage.CreateNamespace(ctx, namespace)
	// 忽略已存在的命名空间错误
	if err != nil {
		if se, ok := err.(*storage.StorageError); !ok || se.Code != storage.ErrAlreadyExists {
			require.NoError(t, err, "创建测试命名空间失败: %s", namespaceName)
		}
	}
}

func testRegisterAndGetService(t *testing.T, ctx context.Context, s *ServiceStorage) {
	// 创建测试命名空间
	createTestNamespace(t, ctx, s.client, "test-ns")

	// 创建测试服务
	service := &storage.Service{
		ID:        "test-service-1",
		Name:      "test-service",
		Namespace: "test-ns",
		IP:        "192.168.1.100",
		Port:      8080,
		Tags:      []string{"test", "api"},
		Metadata:  map[string]string{"version": "1.0"},
		TTL:       30,
	}

	// 注册服务
	err := s.RegisterService(ctx, service)
	require.NoError(t, err)

	// 获取服务
	savedService, err := s.GetService(ctx, service.ID)
	require.NoError(t, err)
	assert.Equal(t, service.ID, savedService.ID)
	assert.Equal(t, service.Name, savedService.Name)
	assert.Equal(t, service.Namespace, savedService.Namespace)
	assert.Equal(t, service.IP, savedService.IP)
	assert.Equal(t, service.Port, savedService.Port)
	assert.Equal(t, "healthy", savedService.Health)
	assert.False(t, savedService.RegisteredAt.IsZero())
	assert.False(t, savedService.LastHeartbeat.IsZero())

	// 测试无效参数
	invalidService := &storage.Service{}
	err = s.RegisterService(ctx, invalidService)
	assert.Error(t, err)
}

func testListServices(t *testing.T, ctx context.Context, s *ServiceStorage) {
	// 创建测试命名空间
	createTestNamespace(t, ctx, s.client, "ns1")
	createTestNamespace(t, ctx, s.client, "ns2")

	// 注册多个服务，不同命名空间
	services := []*storage.Service{
		{ID: "test-service-a1", Name: "test-service-a", Namespace: "ns1", IP: "192.168.1.1", Port: 8001},
		{ID: "test-service-b1", Name: "test-service-b", Namespace: "ns1", IP: "192.168.1.2", Port: 8002},
		{ID: "test-service-a2", Name: "test-service-a", Namespace: "ns2", IP: "192.168.1.3", Port: 8003},
		{ID: "test-service-c1", Name: "test-service-c", Namespace: "ns2", IP: "192.168.1.4", Port: 8004},
	}

	for _, svc := range services {
		err := s.RegisterService(ctx, svc)
		require.NoError(t, err)
	}

	// 测试列出所有服务
	allServices, err := s.ListServices(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(allServices), 4) // 可能还有其他测试添加的服务

	// 测试按名称列出服务
	serviceA, err := s.ListServicesByName(ctx, "test-service-a")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(serviceA), 2)

	serviceB, err := s.ListServicesByName(ctx, "test-service-b")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(serviceB), 1)

	// 测试按命名空间列出服务
	ns1Services, err := s.ListServicesByNamespace(ctx, "ns1")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(ns1Services), 2)

	ns2Services, err := s.ListServicesByNamespace(ctx, "ns2")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(ns2Services), 2)

	// 测试按名称和命名空间列出服务
	serviceA_ns1, err := s.ListServicesByNameAndNamespace(ctx, "ns1", "test-service-a")
	require.NoError(t, err)
	assert.Equal(t, 1, len(serviceA_ns1))
	assert.Equal(t, "test-service-a1", serviceA_ns1[0].ID)

	serviceA_ns2, err := s.ListServicesByNameAndNamespace(ctx, "ns2", "test-service-a")
	require.NoError(t, err)
	assert.Equal(t, 1, len(serviceA_ns2))
	assert.Equal(t, "test-service-a2", serviceA_ns2[0].ID)
}

func testUpdateHeartbeat(t *testing.T, ctx context.Context, s *ServiceStorage) {
	// 创建测试命名空间
	createTestNamespace(t, ctx, s.client, "test-ns-heartbeat")

	// 注册一个服务
	service := &storage.Service{
		ID:        "test-service-heartbeat",
		Name:      "test-service",
		Namespace: "test-ns-heartbeat", // 添加命名空间
		IP:        "192.168.1.104",
		Port:      8084,
	}
	err := s.RegisterService(ctx, service)
	require.NoError(t, err)

	// 记录初始心跳时间
	initialService, err := s.GetService(ctx, service.ID)
	require.NoError(t, err)
	initialHeartbeat := initialService.LastHeartbeat

	// 等待一小段时间
	time.Sleep(10 * time.Millisecond)

	// 更新心跳
	err = s.UpdateServiceHeartbeat(ctx, service.ID)
	require.NoError(t, err)

	// 验证心跳已更新
	updatedService, err := s.GetService(ctx, service.ID)
	require.NoError(t, err)
	assert.True(t, updatedService.LastHeartbeat.After(initialHeartbeat))
	assert.Equal(t, service.Namespace, updatedService.Namespace) // 验证命名空间

	// 测试更新不存在的服务心跳
	err = s.UpdateServiceHeartbeat(ctx, "non-existent-service")
	assert.Error(t, err)
}

func testDeregisterService(t *testing.T, ctx context.Context, s *ServiceStorage) {
	// 创建测试命名空间
	createTestNamespace(t, ctx, s.client, "test-ns-deregister")

	// 注册一个服务
	service := &storage.Service{
		ID:        "test-service-deregister",
		Name:      "test-service",
		Namespace: "test-ns-deregister", // 添加命名空间
		IP:        "192.168.1.105",
		Port:      8085,
	}
	err := s.RegisterService(ctx, service)
	require.NoError(t, err)

	// 注销服务
	err = s.DeregisterService(ctx, service.ID)
	require.NoError(t, err)

	// 验证服务已被注销
	_, err = s.GetService(ctx, service.ID)
	assert.Error(t, err)
	storageErr, ok := err.(*storage.StorageError)
	require.True(t, ok)
	assert.Equal(t, storage.ErrNotFound, storageErr.Code)

	// 测试注销不存在的服务
	err = s.DeregisterService(ctx, "non-existent-service")
	assert.Error(t, err)
}

func testCleanupStaleServices(t *testing.T, ctx context.Context, s *ServiceStorage) {
	// 创建测试命名空间
	createTestNamespace(t, ctx, s.client, "test-ns-cleanup1")
	createTestNamespace(t, ctx, s.client, "test-ns-cleanup2")

	// 先检查并删除可能存在的测试服务，避免干扰
	_ = s.DeregisterService(ctx, "test-service-stale")
	_ = s.DeregisterService(ctx, "test-service-active")

	time.Sleep(100 * time.Millisecond) // 确保删除操作完成

	// 注册服务，测试不同命名空间的过期服务清理
	staleService := &storage.Service{
		ID:        "test-service-stale",
		Name:      "test-service-stale",
		Namespace: "test-ns-cleanup1", // 添加命名空间
		IP:        "192.168.1.10",
		Port:      8010,
		// 直接设置为过期时间
		LastHeartbeat: time.Now().Add(-2 * time.Minute),
	}

	activeService := &storage.Service{
		ID:        "test-service-active",
		Name:      "test-service-stale",
		Namespace: "test-ns-cleanup2", // 不同命名空间
		IP:        "192.168.1.11",
		Port:      8011,
	}

	// 注册服务
	err := s.RegisterService(ctx, staleService)
	require.NoError(t, err)

	err = s.RegisterService(ctx, activeService)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond) // 确保注册操作完成

	// 验证服务已注册且时间正确
	retrievedStale, err := s.GetService(ctx, "test-service-stale")
	require.NoError(t, err, "获取过期服务失败")

	t.Logf("过期服务心跳时间: %v, 当前时间: %v, 差值: %v",
		retrievedStale.LastHeartbeat, time.Now(),
		time.Now().Sub(retrievedStale.LastHeartbeat))

	// 验证心跳时间是否正确设置为过期
	assert.True(t, retrievedStale.LastHeartbeat.Before(time.Now().Add(-1*time.Minute)),
		"过期服务心跳时间应该在1分钟前")
	assert.Equal(t, "test-ns-cleanup1", retrievedStale.Namespace, "命名空间不匹配")

	// 清理过期服务
	t.Log("开始清理过期服务")
	err = s.CleanupStaleServices(ctx, 1*time.Minute)
	require.NoError(t, err)

	// 等待确保etcd操作完成
	time.Sleep(500 * time.Millisecond)

	// 验证过期服务已被清理
	t.Log("验证过期服务是否已被清理")
	_, err = s.GetService(ctx, "test-service-stale")
	if err == nil {
		t.Error("过期服务应该已被清理，但仍能获取到")
	} else {
		t.Logf("获取过期服务返回错误: %v", err)
		storageErr, ok := err.(*storage.StorageError)
		if !ok {
			t.Errorf("错误类型不正确: %T", err)
		} else {
			assert.Equal(t, storage.ErrNotFound, storageErr.Code)
		}
	}

	// 验证活跃服务仍存在
	activeRetrieved, err := s.GetService(ctx, "test-service-active")
	assert.NoError(t, err, "活跃服务应该仍然存在")
	assert.Equal(t, "test-ns-cleanup2", activeRetrieved.Namespace, "活跃服务命名空间不匹配")
}

// 添加对命名空间功能的特定测试
func testNamespaceFeatures(t *testing.T, ctx context.Context, s *ServiceStorage) {
	// 清理可能存在的测试数据
	cleanupTestData(t, ctx, s)

	// 创建测试命名空间
	createTestNamespace(t, ctx, s.client, "namespace-1")
	createTestNamespace(t, ctx, s.client, "namespace-2")

	// 在不同命名空间中注册同名服务
	service1 := &storage.Service{
		ID:        "test-ns-feature-1",
		Name:      "same-service-name",
		Namespace: "namespace-1",
		IP:        "192.168.5.1",
		Port:      9001,
	}

	service2 := &storage.Service{
		ID:        "test-ns-feature-2",
		Name:      "same-service-name",
		Namespace: "namespace-2",
		IP:        "192.168.5.2",
		Port:      9002,
	}

	// 注册服务
	err := s.RegisterService(ctx, service1)
	require.NoError(t, err, "注册第一个服务失败")

	err = s.RegisterService(ctx, service2)
	require.NoError(t, err, "注册第二个服务失败")

	// 验证可以分别获取两个服务
	svc1, err := s.GetService(ctx, service1.ID)
	require.NoError(t, err, "获取第一个服务失败")
	assert.Equal(t, service1.Namespace, svc1.Namespace)
	assert.Equal(t, service1.IP, svc1.IP)

	svc2, err := s.GetService(ctx, service2.ID)
	require.NoError(t, err, "获取第二个服务失败")
	assert.Equal(t, service2.Namespace, svc2.Namespace)
	assert.Equal(t, service2.IP, svc2.IP)

	// 通过命名空间和名称查询服务
	services1, err := s.ListServicesByNameAndNamespace(ctx, "namespace-1", "same-service-name")
	require.NoError(t, err, "查询namespace-1下的服务失败")
	assert.Equal(t, 1, len(services1), "应该只有一个服务")
	assert.Equal(t, "192.168.5.1", services1[0].IP)

	services2, err := s.ListServicesByNameAndNamespace(ctx, "namespace-2", "same-service-name")
	require.NoError(t, err, "查询namespace-2下的服务失败")
	assert.Equal(t, 1, len(services2), "应该只有一个服务")
	assert.Equal(t, "192.168.5.2", services2[0].IP)

	// 清理测试数据
	err = s.DeregisterService(ctx, service1.ID)
	require.NoError(t, err, "注销第一个服务失败")

	err = s.DeregisterService(ctx, service2.ID)
	require.NoError(t, err, "注销第二个服务失败")
}
