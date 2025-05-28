package etcdclient

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/hewenyu/kong-discovery/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWatcher(t *testing.T) {
	// 跳过测试，如果环境变量ETCD_TEST_ENDPOINTS未设置
	etcdEndpoints := os.Getenv("ETCD_TEST_ENDPOINTS")
	if etcdEndpoints == "" {
		t.Skip("跳过测试：未设置ETCD_TEST_ENDPOINTS环境变量")
	}

	// 创建测试配置
	cfg := &config.Config{}
	cfg.Etcd.Endpoints = []string{etcdEndpoints}
	cfg.Etcd.Username = os.Getenv("ETCD_TEST_USERNAME")
	cfg.Etcd.Password = os.Getenv("ETCD_TEST_PASSWORD")

	// 创建日志器
	logger, err := config.NewLogger(true)
	require.NoError(t, err, "创建日志器失败")

	// 创建etcd客户端
	client := NewEtcdClient(cfg, logger)
	err = client.Connect()
	require.NoError(t, err, "连接etcd失败")
	defer client.Close()

	// 创建一个测试前缀
	testPrefix := "/test/watcher/" + time.Now().Format("20060102150405")
	testKey := testPrefix + "/testkey"

	// 创建一个等待组和通道来接收事件
	var wg sync.WaitGroup
	wg.Add(3) // 等待创建、更新和删除事件

	events := make([]WatchEvent, 0, 3)
	var eventsMutex sync.Mutex

	// 启动监听
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.StartWatch(ctx, testPrefix, func(event WatchEvent) {
		eventsMutex.Lock()
		defer eventsMutex.Unlock()

		events = append(events, event)
		logger.Info("收到事件",
			zap.String("type", event.EventType),
			zap.String("key", event.Key),
			zap.String("value", event.Value))
		wg.Done()
	})
	require.NoError(t, err, "启动监听失败")

	// 等待一小段时间，确保监听已启动
	time.Sleep(1 * time.Second)

	// 创建测试键
	ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel1()
	_, err = client.(*EtcdClient).client.Put(ctx1, testKey, "value1")
	require.NoError(t, err, "创建测试键失败")

	// 更新测试键
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	_, err = client.(*EtcdClient).client.Put(ctx2, testKey, "value2")
	require.NoError(t, err, "更新测试键失败")

	// 删除测试键
	ctx3, cancel3 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel3()
	_, err = client.(*EtcdClient).client.Delete(ctx3, testKey)
	require.NoError(t, err, "删除测试键失败")

	// 等待所有事件被处理
	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()

	// 设置超时
	select {
	case <-waitCh:
		// 所有事件都已处理
	case <-time.After(10 * time.Second):
		t.Fatal("测试超时：未收到所有期望的事件")
	}

	// 验证事件
	eventsMutex.Lock()
	defer eventsMutex.Unlock()

	assert.Equal(t, 3, len(events), "应该收到3个事件")

	// 验证事件类型和值
	foundCreate := false
	foundUpdate := false
	foundDelete := false

	for _, event := range events {
		assert.Equal(t, testKey, event.Key, "事件键应该匹配")

		switch event.EventType {
		case "create":
			foundCreate = true
			assert.Equal(t, "value1", event.Value, "创建事件的值应该是value1")
		case "update":
			foundUpdate = true
			assert.Equal(t, "value2", event.Value, "更新事件的值应该是value2")
			assert.Equal(t, "value1", event.PrevValue, "更新事件的前值应该是value1")
		case "delete":
			foundDelete = true
			assert.Equal(t, "", event.Value, "删除事件的值应该是空")
			assert.Equal(t, "value2", event.PrevValue, "删除事件的前值应该是value2")
		}
	}

	assert.True(t, foundCreate, "应该收到创建事件")
	assert.True(t, foundUpdate, "应该收到更新事件")
	assert.True(t, foundDelete, "应该收到删除事件")
}

func TestServiceWatcher(t *testing.T) {
	// 跳过测试，如果环境变量ETCD_TEST_ENDPOINTS未设置
	etcdEndpoints := os.Getenv("ETCD_TEST_ENDPOINTS")
	if etcdEndpoints == "" {
		t.Skip("跳过测试：未设置ETCD_TEST_ENDPOINTS环境变量")
	}

	// 创建测试配置
	cfg := &config.Config{}
	cfg.Etcd.Endpoints = []string{etcdEndpoints}
	cfg.Etcd.Username = os.Getenv("ETCD_TEST_USERNAME")
	cfg.Etcd.Password = os.Getenv("ETCD_TEST_PASSWORD")

	// 创建日志器
	logger, err := config.NewLogger(true)
	require.NoError(t, err, "创建日志器失败")

	// 创建etcd客户端
	client := NewEtcdClient(cfg, logger)
	err = client.Connect()
	require.NoError(t, err, "连接etcd失败")
	defer client.Close()

	// 创建一个测试服务
	testService := &ServiceInstance{
		ServiceName: "test-service",
		InstanceID:  "test-instance-1",
		IPAddress:   "192.168.1.1",
		Port:        8080,
		TTL:         60,
		Metadata: map[string]string{
			"version": "1.0.0",
			"env":     "test",
		},
	}

	// 创建一个等待组和通道来接收事件
	var wg sync.WaitGroup
	wg.Add(2) // 等待注册和注销事件

	var serviceEvent *WatchEvent
	var eventMutex sync.Mutex

	// 启动监听
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.StartWatch(ctx, "/services/", func(event WatchEvent) {
		if event.ServiceObj != nil &&
			event.ServiceObj.ServiceName == testService.ServiceName &&
			event.ServiceObj.InstanceID == testService.InstanceID {

			eventMutex.Lock()
			serviceEvent = &event
			eventMutex.Unlock()

			logger.Info("收到服务事件",
				zap.String("type", event.EventType),
				zap.String("service", event.ServiceObj.ServiceName),
				zap.String("id", event.ServiceObj.InstanceID))

			wg.Done()
		}
	})
	require.NoError(t, err, "启动监听失败")

	// 等待一小段时间，确保监听已启动
	time.Sleep(1 * time.Second)

	// 注册服务
	ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel1()
	err = client.RegisterService(ctx1, testService)
	require.NoError(t, err, "注册服务失败")

	// 等待事件处理
	time.Sleep(1 * time.Second)

	// 检查事件
	eventMutex.Lock()
	assert.NotNil(t, serviceEvent, "应该收到服务注册事件")
	assert.Equal(t, "create", serviceEvent.EventType, "事件类型应该是create")
	assert.NotNil(t, serviceEvent.ServiceObj, "服务对象不应该为nil")
	assert.Equal(t, testService.ServiceName, serviceEvent.ServiceObj.ServiceName, "服务名称应该匹配")
	assert.Equal(t, testService.InstanceID, serviceEvent.ServiceObj.InstanceID, "实例ID应该匹配")
	assert.Equal(t, testService.IPAddress, serviceEvent.ServiceObj.IPAddress, "IP地址应该匹配")
	assert.Equal(t, testService.Port, serviceEvent.ServiceObj.Port, "端口应该匹配")
	eventMutex.Unlock()

	// 注销服务
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	err = client.DeregisterService(ctx2, testService.ServiceName, testService.InstanceID)
	require.NoError(t, err, "注销服务失败")

	// 等待所有事件被处理
	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()

	// 设置超时
	select {
	case <-waitCh:
		// 所有事件都已处理
	case <-time.After(10 * time.Second):
		t.Fatal("测试超时：未收到所有期望的事件")
	}
}
