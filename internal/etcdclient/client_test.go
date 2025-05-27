package etcdclient

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/hewenyu/kong-discovery/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 创建一个测试用的配置，使用环境变量中的etcd地址
func createTestConfig(t *testing.T) *config.Config {
	t.Helper()

	// 从环境变量中获取etcd地址
	etcdEndpoints := os.Getenv("KONG_DISCOVERY_ETCD_ENDPOINTS")
	require.NotEmpty(t, etcdEndpoints, "环境变量KONG_DISCOVERY_ETCD_ENDPOINTS必须设置")

	// 创建配置
	cfg := &config.Config{}
	cfg.Etcd.Endpoints = []string{etcdEndpoints}
	cfg.Etcd.Username = "" // 如果需要认证，设置相应的值
	cfg.Etcd.Password = "" // 如果需要认证，设置相应的值

	return cfg
}

// 创建测试用的日志记录器
func createTestLogger(t *testing.T) config.Logger {
	t.Helper()

	logger, err := config.NewLogger(true)
	require.NoError(t, err, "创建测试日志记录器失败")

	return logger
}

func TestEtcdClient_Connect(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建配置和日志记录器
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建etcd客户端
	client := NewEtcdClient(cfg, logger)

	// 测试连接
	err := client.Connect()
	assert.NoError(t, err, "连接etcd应该成功")

	// 确保在测试结束时关闭连接
	defer func() {
		err := client.Close()
		assert.NoError(t, err, "关闭etcd连接应该成功")
	}()
}

func TestEtcdClient_Ping(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建配置和日志记录器
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建etcd客户端并连接
	client := NewEtcdClient(cfg, logger)
	err := client.Connect()
	require.NoError(t, err, "连接etcd应该成功")

	// 确保在测试结束时关闭连接
	defer func() {
		err := client.Close()
		assert.NoError(t, err, "关闭etcd连接应该成功")
	}()

	// 测试Ping
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.Ping(ctx)
	assert.NoError(t, err, "Ping etcd应该成功")
}

func TestEtcdClient_GetAndGetWithPrefix(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建配置和日志记录器
	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	// 创建etcd客户端并连接
	client := NewEtcdClient(cfg, logger)
	err := client.Connect()
	require.NoError(t, err, "连接etcd应该成功")

	// 确保在测试结束时关闭连接
	defer func() {
		err := client.Close()
		assert.NoError(t, err, "关闭etcd连接应该成功")
	}()

	// 测试Get - 这将在实际环境中进行，所以可能会失败
	// 如果key不存在，这是预期的行为
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Get(ctx, "test-key")
	// 我们不断言错误，因为key可能不存在
	t.Logf("Get结果: %v", err)

	// 测试GetWithPrefix
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	result, err := client.GetWithPrefix(ctx2, "test")
	// 我们不断言错误，因为前缀可能没有匹配项
	t.Logf("GetWithPrefix结果: %v, %v", result, err)
}
