package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
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

	// 直接创建两个过期服务和两个新鲜服务
	freshTime := time.Now()
	staleTime := freshTime.Add(-5 * time.Minute) // 5分钟前，模拟过期服务

	// 修复：服务ID在JSON数据中必须与键名中的ID一致
	freshID1 := uuid.New().String()
	freshID2 := uuid.New().String()
	freshServices := []struct {
		id        string
		name      string
		namespace string
		data      []byte
	}{
		{
			id:        freshID1,
			name:      "fresh-service-1",
			namespace: "test",
			data:      []byte(`{"id":"` + freshID1 + `","name":"fresh-service-1","namespace":"test","ip":"192.168.1.1","port":8081,"health":"healthy","registered_at":"` + freshTime.Format(time.RFC3339Nano) + `","last_heartbeat":"` + freshTime.Format(time.RFC3339Nano) + `"}`),
		},
		{
			id:        freshID2,
			name:      "fresh-service-2",
			namespace: "test",
			data:      []byte(`{"id":"` + freshID2 + `","name":"fresh-service-2","namespace":"test","ip":"192.168.1.2","port":8082,"health":"healthy","registered_at":"` + freshTime.Format(time.RFC3339Nano) + `","last_heartbeat":"` + freshTime.Format(time.RFC3339Nano) + `"}`),
		},
	}

	// 手动构造过期服务
	staleID1 := uuid.New().String()
	staleID2 := uuid.New().String()
	staleServices := []struct {
		id        string
		name      string
		namespace string
		data      []byte
	}{
		{
			id:        staleID1,
			name:      "stale-service-1",
			namespace: "test",
			data:      []byte(`{"id":"` + staleID1 + `","name":"stale-service-1","namespace":"test","ip":"192.168.1.3","port":8083,"health":"healthy","registered_at":"` + staleTime.Format(time.RFC3339Nano) + `","last_heartbeat":"` + staleTime.Format(time.RFC3339Nano) + `"}`),
		},
		{
			id:        staleID2,
			name:      "stale-service-2",
			namespace: "test",
			data:      []byte(`{"id":"` + staleID2 + `","name":"stale-service-2","namespace":"test","ip":"192.168.1.4","port":8084,"health":"healthy","registered_at":"` + staleTime.Format(time.RFC3339Nano) + `","last_heartbeat":"` + staleTime.Format(time.RFC3339Nano) + `"}`),
		},
	}

	// 手动写入服务数据到etcd
	for _, service := range freshServices {
		// 写入服务数据
		err := client.Put(ctx, getServiceKey(service.id), service.data)
		assert.NoError(t, err)

		// 写入服务名称索引
		nameIndexKey := getServiceNameIndexKey(service.name, service.namespace)
		serviceIDs := []string{service.id}
		serviceIDsData, err := json.Marshal(serviceIDs)
		assert.NoError(t, err)
		err = client.Put(ctx, nameIndexKey, serviceIDsData)
		assert.NoError(t, err)
	}

	for _, service := range staleServices {
		// 写入服务数据
		err := client.Put(ctx, getServiceKey(service.id), service.data)
		assert.NoError(t, err)

		// 写入服务名称索引
		nameIndexKey := getServiceNameIndexKey(service.name, service.namespace)
		serviceIDs := []string{service.id}
		serviceIDsData, err := json.Marshal(serviceIDs)
		assert.NoError(t, err)
		err = client.Put(ctx, nameIndexKey, serviceIDsData)
		assert.NoError(t, err)
	}

	// 获取所有服务
	allServices, err := store.ListAllServices(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(freshServices)+len(staleServices), len(allServices))

	// 打印所有服务的心跳时间，用于调试
	for _, service := range allServices {
		t.Logf("服务 %s (ID: %s) 的最后心跳时间: %v", service.Name, service.ID, service.LastHeartbeat)
	}

	// 清理过期服务 (超过3分钟未心跳)
	before := time.Now().Add(-3 * time.Minute)
	t.Logf("清理 %v 之前的服务", before)
	count, err := store.CleanupStaleServices(ctx, before)
	assert.NoError(t, err)
	assert.Equal(t, len(staleServices), count)

	// 获取所有服务，应该只剩下新鲜的服务
	remainingServices, err := store.ListAllServices(ctx)
	assert.NoError(t, err)
	assert.Equal(t, len(freshServices), len(remainingServices))

	// 检查剩余的服务是否都是新鲜的服务
	for _, service := range remainingServices {
		assert.True(t, strings.HasPrefix(service.Name, "fresh-"), "剩余的服务应该都是新鲜的服务")
	}

	// 清理测试数据
	cleanupTestData(client, servicePrefix)
	cleanupTestData(client, serviceNameIndexPrefix)
}
