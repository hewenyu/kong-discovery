package memory

import (
	"context"
	"testing"
	"time"

	"github.com/hewenyu/kong-discovery/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryServiceStorage_RegisterService(t *testing.T) {
	// 创建存储实例
	s := NewServiceStorage()
	ctx := context.Background()

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

	// 验证注册是否成功
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

func TestMemoryServiceStorage_DeregisterService(t *testing.T) {
	s := NewServiceStorage()
	ctx := context.Background()

	// 注册一个服务
	service := &storage.Service{
		ID:   "test-service-2",
		Name: "test-service",
		IP:   "192.168.1.101",
		Port: 8081,
	}
	err := s.RegisterService(ctx, service)
	require.NoError(t, err)

	// 测试注销服务
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

func TestMemoryServiceStorage_GetService(t *testing.T) {
	s := NewServiceStorage()
	ctx := context.Background()

	// 注册一个服务
	service := &storage.Service{
		ID:   "test-service-3",
		Name: "test-service",
		IP:   "192.168.1.102",
		Port: 8082,
	}
	err := s.RegisterService(ctx, service)
	require.NoError(t, err)

	// 测试获取服务
	savedService, err := s.GetService(ctx, service.ID)
	require.NoError(t, err)
	assert.Equal(t, service.ID, savedService.ID)

	// 测试获取不存在的服务
	_, err = s.GetService(ctx, "non-existent-service")
	assert.Error(t, err)
}

func TestMemoryServiceStorage_ListServices(t *testing.T) {
	s := NewServiceStorage()
	ctx := context.Background()

	// 注册多个服务
	services := []*storage.Service{
		{ID: "service-1", Name: "service-a", IP: "192.168.1.1", Port: 8001},
		{ID: "service-2", Name: "service-b", IP: "192.168.1.2", Port: 8002},
		{ID: "service-3", Name: "service-a", IP: "192.168.1.3", Port: 8003},
	}

	for _, svc := range services {
		err := s.RegisterService(ctx, svc)
		require.NoError(t, err)
	}

	// 测试列出所有服务
	allServices, err := s.ListServices(ctx)
	require.NoError(t, err)
	assert.Len(t, allServices, 3)

	// 测试按名称列出服务
	serviceA, err := s.ListServicesByName(ctx, "service-a")
	require.NoError(t, err)
	assert.Len(t, serviceA, 2)

	serviceB, err := s.ListServicesByName(ctx, "service-b")
	require.NoError(t, err)
	assert.Len(t, serviceB, 1)
}

func TestMemoryServiceStorage_UpdateServiceHeartbeat(t *testing.T) {
	s := NewServiceStorage()
	ctx := context.Background()

	// 注册一个服务
	service := &storage.Service{
		ID:   "test-service-4",
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

func TestMemoryServiceStorage_CleanupStaleServices(t *testing.T) {
	s := NewServiceStorage()
	ctx := context.Background()

	// 注册服务
	services := []*storage.Service{
		{ID: "service-stale-1", Name: "service-stale", IP: "192.168.1.1", Port: 8001},
		{ID: "service-stale-2", Name: "service-stale", IP: "192.168.1.2", Port: 8002},
	}

	for _, svc := range services {
		err := s.RegisterService(ctx, svc)
		require.NoError(t, err)
	}

	// 确认服务存在
	_, err := s.GetService(ctx, "service-stale-1")
	require.NoError(t, err)

	// 直接修改内存中的对象
	s.mu.Lock()
	s.services["service-stale-1"].LastHeartbeat = time.Now().Add(-2 * time.Minute)
	s.mu.Unlock()

	// 清理过期服务（超时设置为1分钟）
	err = s.CleanupStaleServices(ctx, 1*time.Minute)
	require.NoError(t, err)

	// 验证过期服务已被清理
	_, err = s.GetService(ctx, "service-stale-1")
	assert.Error(t, err)

	// 验证未过期服务仍存在
	_, err = s.GetService(ctx, "service-stale-2")
	assert.NoError(t, err)
}
