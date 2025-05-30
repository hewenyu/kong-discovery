package service

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/hewenyu/kong-discovery/internal/core/config"
	"github.com/hewenyu/kong-discovery/internal/core/model"
	"github.com/hewenyu/kong-discovery/internal/store/etcd"
)

// 这些测试需要一个正在运行的etcd实例
// 如果测试环境中有ETCD_ENDPOINTS环境变量，会使用该地址，否则使用localhost:2379

func getEtcdClient() (*etcd.Client, error) {

	if os.Getenv("ETCD_ENDPOINTS") != "" {
		cfg := &config.EtcdConfig{
			Endpoints:      []string{os.Getenv("ETCD_ENDPOINTS")},
			DialTimeout:    5 * time.Second,
			RequestTimeout: 10 * time.Second,
		}
		return etcd.NewClient(cfg)
	}
	return nil, errors.New("ETCD_ENDPOINTS 未设置")
}

func setupEtcdClient(t *testing.T) *etcd.Client {
	client, err := getEtcdClient()
	if err != nil {
		t.Skip("跳过测试，无法连接到etcd: ", err)
		return nil
	}

	return client
}

func cleanupTestData(client *etcd.Client, prefix string) {
	ctx := context.Background()
	_ = client.DeleteWithPrefix(ctx, prefix)
}

func TestEtcdServiceStore_Register(t *testing.T) {
	client := setupEtcdClient(t)
	if client == nil {
		return
	}
	defer client.Close()

	// 清理测试数据
	cleanupTestData(client, servicePrefix)
	cleanupTestData(client, serviceNameIndexPrefix)

	// 创建服务存储
	store := NewEtcdServiceStore(client, "test")

	// 创建测试上下文
	ctx := context.Background()

	// 创建测试服务
	service := &model.Service{
		Name:      "test-service",
		Namespace: "test",
		IP:        "192.168.1.100",
		Port:      8080,
		Tags:      []string{"test", "integration"},
		Metadata: map[string]string{
			"version": "1.0.0",
		},
		TTL: 30 * time.Second,
	}

	// 注册服务
	err := store.Register(ctx, service)
	assert.NoError(t, err)
	assert.NotEmpty(t, service.ID)
	assert.Equal(t, model.HealthStatusHealthy, service.Health)
	assert.NotEmpty(t, service.RegisteredAt)
	assert.NotEmpty(t, service.LastHeartbeat)

	// 获取服务
	retrievedService, err := store.GetService(ctx, service.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedService)
	assert.Equal(t, service.ID, retrievedService.ID)
	assert.Equal(t, service.Name, retrievedService.Name)
	assert.Equal(t, service.Namespace, retrievedService.Namespace)
	assert.Equal(t, service.IP, retrievedService.IP)
	assert.Equal(t, service.Port, retrievedService.Port)
	assert.Equal(t, service.Health, retrievedService.Health)

	// 查询服务名
	services, err := store.GetServiceByName(ctx, service.Name, service.Namespace)
	assert.NoError(t, err)
	assert.Len(t, services, 1)
	assert.Equal(t, service.ID, services[0].ID)

	// 清理测试数据
	cleanupTestData(client, servicePrefix)
	cleanupTestData(client, serviceNameIndexPrefix)
}

func TestEtcdServiceStore_RegisterMultiple(t *testing.T) {
	client := setupEtcdClient(t)
	if client == nil {
		return
	}
	defer client.Close()

	// 清理测试数据
	cleanupTestData(client, servicePrefix)
	cleanupTestData(client, serviceNameIndexPrefix)

	// 创建服务存储
	store := NewEtcdServiceStore(client, "test")

	// 创建测试上下文
	ctx := context.Background()

	// 创建多个相同名称的服务
	serviceName := "test-service-multi"
	namespace := "test"
	count := 3

	var serviceIDs []string
	for i := 0; i < count; i++ {
		service := &model.Service{
			Name:      serviceName,
			Namespace: namespace,
			IP:        "192.168.1.1",
			Port:      8080 + i,
			TTL:       30 * time.Second,
		}

		// 注册服务
		err := store.Register(ctx, service)
		assert.NoError(t, err)
		assert.NotEmpty(t, service.ID)
		serviceIDs = append(serviceIDs, service.ID)
	}

	// 查询服务名
	services, err := store.GetServiceByName(ctx, serviceName, namespace)
	assert.NoError(t, err)
	assert.Len(t, services, count)

	// 验证服务列表
	for _, service := range services {
		assert.Contains(t, serviceIDs, service.ID)
		assert.Equal(t, serviceName, service.Name)
		assert.Equal(t, namespace, service.Namespace)
	}

	// 查询命名空间下所有服务
	services, err = store.ListServices(ctx, namespace)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(services), count)

	// 清理测试数据
	cleanupTestData(client, servicePrefix)
	cleanupTestData(client, serviceNameIndexPrefix)
}

func TestEtcdServiceStore_UpdateHeartbeat(t *testing.T) {
	client := setupEtcdClient(t)
	if client == nil {
		return
	}
	defer client.Close()

	// 清理测试数据
	cleanupTestData(client, servicePrefix)
	cleanupTestData(client, serviceNameIndexPrefix)

	// 创建服务存储
	store := NewEtcdServiceStore(client, "test")

	// 创建测试上下文
	ctx := context.Background()

	// 创建测试服务
	service := &model.Service{
		Name:      "test-service-heartbeat",
		Namespace: "test",
		IP:        "192.168.1.100",
		Port:      8080,
		TTL:       30 * time.Second,
	}

	// 注册服务
	err := store.Register(ctx, service)
	assert.NoError(t, err)

	// 记录初始心跳时间
	initialHeartbeat := service.LastHeartbeat

	// 等待1秒
	time.Sleep(1 * time.Second)

	// 更新心跳
	err = store.UpdateHeartbeat(ctx, service.ID)
	assert.NoError(t, err)

	// 获取服务
	updatedService, err := store.GetService(ctx, service.ID)
	assert.NoError(t, err)
	assert.NotNil(t, updatedService)
	assert.True(t, updatedService.LastHeartbeat.After(initialHeartbeat))

	// 清理测试数据
	cleanupTestData(client, servicePrefix)
	cleanupTestData(client, serviceNameIndexPrefix)
}

func TestEtcdServiceStore_Deregister(t *testing.T) {
	client := setupEtcdClient(t)
	if client == nil {
		return
	}
	defer client.Close()

	// 清理测试数据
	cleanupTestData(client, servicePrefix)
	cleanupTestData(client, serviceNameIndexPrefix)

	// 创建服务存储
	store := NewEtcdServiceStore(client, "test")

	// 创建测试上下文
	ctx := context.Background()

	// 创建测试服务
	service := &model.Service{
		Name:      "test-service-deregister",
		Namespace: "test",
		IP:        "192.168.1.100",
		Port:      8080,
		TTL:       30 * time.Second,
	}

	// 注册服务
	err := store.Register(ctx, service)
	assert.NoError(t, err)

	// 获取服务
	retrievedService, err := store.GetService(ctx, service.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedService)

	// 注销服务
	err = store.Deregister(ctx, service.ID)
	assert.NoError(t, err)

	// 获取服务，应该不存在
	retrievedService, err = store.GetService(ctx, service.ID)
	assert.NoError(t, err)
	assert.Nil(t, retrievedService)

	// 尝试注销不存在的服务
	nonexistentID := uuid.New().String()
	err = store.Deregister(ctx, nonexistentID)
	assert.Error(t, err)

	// 清理测试数据
	cleanupTestData(client, servicePrefix)
	cleanupTestData(client, serviceNameIndexPrefix)
}

func TestEtcdServiceStore_CleanupStaleServices(t *testing.T) {
	client := setupEtcdClient(t)
	if client == nil {
		return
	}
	defer client.Close()

	// 清理测试数据
	cleanupTestData(client, servicePrefix)
	cleanupTestData(client, serviceNameIndexPrefix)

	// 创建服务存储
	store := NewEtcdServiceStore(client, "test")

	// 创建测试上下文
	ctx := context.Background()

	// 创建多个服务，包括一些过期的服务
	services := []*model.Service{
		{
			Name:          "fresh-service-1",
			Namespace:     "test",
			IP:            "192.168.1.1",
			Port:          8081,
			LastHeartbeat: time.Now(),
		},
		{
			Name:          "fresh-service-2",
			Namespace:     "test",
			IP:            "192.168.1.2",
			Port:          8082,
			LastHeartbeat: time.Now(),
		},
		{
			Name:          "stale-service-1",
			Namespace:     "test",
			IP:            "192.168.1.3",
			Port:          8083,
			LastHeartbeat: time.Now().Add(-5 * time.Minute),
		},
		{
			Name:          "stale-service-2",
			Namespace:     "test",
			IP:            "192.168.1.4",
			Port:          8084,
			LastHeartbeat: time.Now().Add(-10 * time.Minute),
		},
	}

	// 注册所有服务
	for _, service := range services {
		err := store.Register(ctx, service)
		assert.NoError(t, err)
	}

	// 获取所有服务
	allServices, err := store.ListAllServices(ctx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(allServices), len(services))

	// 清理过期服务 (超过3分钟未心跳)
	before := time.Now().Add(-3 * time.Minute)
	count, err := store.CleanupStaleServices(ctx, before)
	assert.NoError(t, err)
	assert.Equal(t, 2, count)

	// 获取所有服务，应该只剩下新鲜的服务
	remainingServices, err := store.ListAllServices(ctx)
	assert.NoError(t, err)
	assert.Len(t, remainingServices, 2)

	// 清理测试数据
	cleanupTestData(client, servicePrefix)
	cleanupTestData(client, serviceNameIndexPrefix)
}
