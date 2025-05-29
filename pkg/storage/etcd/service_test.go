package etcd

import (
	"context"
	"os"
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
		prefix: "/kong-discovery/services/",
	}

	// 测试获取服务键
	key := client.GetServiceKey("test-service")
	assert.Equal(t, "/kong-discovery/services/test-service", key)

	// 测试获取服务前缀
	prefix := client.GetServicesPrefix()
	assert.Equal(t, "/kong-discovery/services/", prefix)
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

	username := os.Getenv("ETCD_USERNAME")
	password := os.Getenv("ETCD_PASSWORD")

	return &config.EtcdConfig{
		Endpoints:   []string{endpoints},
		DialTimeout: "5s",
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

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

func testRegisterAndGetService(t *testing.T, ctx context.Context, s *ServiceStorage) {
	// 创建测试服务
	service := &storage.Service{
		ID:       "test-service-1",
		Name:     "test-service",
		IP:       "192.168.1.100",
		Port:     8080,
		Tags:     []string{"test", "api"},
		Metadata: map[string]string{"version": "1.0"},
		TTL:      30,
	}

	// 注册服务
	err := s.RegisterService(ctx, service)
	require.NoError(t, err)

	// 获取服务
	savedService, err := s.GetService(ctx, service.ID)
	require.NoError(t, err)
	assert.Equal(t, service.ID, savedService.ID)
	assert.Equal(t, service.Name, savedService.Name)
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
	// 注册多个服务
	services := []*storage.Service{
		{ID: "test-service-a1", Name: "test-service-a", IP: "192.168.1.1", Port: 8001},
		{ID: "test-service-b1", Name: "test-service-b", IP: "192.168.1.2", Port: 8002},
		{ID: "test-service-a2", Name: "test-service-a", IP: "192.168.1.3", Port: 8003},
	}

	for _, svc := range services {
		err := s.RegisterService(ctx, svc)
		require.NoError(t, err)
	}

	// 测试列出所有服务
	allServices, err := s.ListServices(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(allServices), 3) // 可能还有其他测试添加的服务

	// 测试按名称列出服务
	serviceA, err := s.ListServicesByName(ctx, "test-service-a")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(serviceA), 2)

	serviceB, err := s.ListServicesByName(ctx, "test-service-b")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(serviceB), 1)
}

func testUpdateHeartbeat(t *testing.T, ctx context.Context, s *ServiceStorage) {
	// 注册一个服务
	service := &storage.Service{
		ID:   "test-service-heartbeat",
		Name: "test-service",
		IP:   "192.168.1.104",
		Port: 8084,
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

	// 测试更新不存在的服务心跳
	err = s.UpdateServiceHeartbeat(ctx, "non-existent-service")
	assert.Error(t, err)
}

func testDeregisterService(t *testing.T, ctx context.Context, s *ServiceStorage) {
	// 注册一个服务
	service := &storage.Service{
		ID:   "test-service-deregister",
		Name: "test-service",
		IP:   "192.168.1.105",
		Port: 8085,
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
	// 先检查并删除可能存在的测试服务，避免干扰
	_ = s.DeregisterService(ctx, "test-service-stale")
	_ = s.DeregisterService(ctx, "test-service-active")

	time.Sleep(100 * time.Millisecond) // 确保删除操作完成

	// 注册服务
	staleService := &storage.Service{
		ID:   "test-service-stale",
		Name: "test-service-stale",
		IP:   "192.168.1.10",
		Port: 8010,
		// 直接设置为过期时间
		LastHeartbeat: time.Now().Add(-2 * time.Minute),
	}

	activeService := &storage.Service{
		ID:   "test-service-active",
		Name: "test-service-stale",
		IP:   "192.168.1.11",
		Port: 8011,
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
	_, err = s.GetService(ctx, "test-service-active")
	assert.NoError(t, err, "活跃服务应该仍然存在")
}
