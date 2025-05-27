package etcdclient

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/hewenyu/kong-discovery/internal/config"
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

// CreateEtcdClientForTest 创建并连接真实的etcd客户端，供测试使用
// 这是一个导出函数，可以被其他包使用
func CreateEtcdClientForTest(t *testing.T) Client {
	t.Helper()

	cfg := createTestConfig(t)
	logger := createTestLogger(t)

	client := NewEtcdClient(cfg, logger)
	err := client.Connect()
	require.NoError(t, err, "连接etcd失败")

	// 确保连接正常
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = client.Ping(ctx)
	require.NoError(t, err, "Ping etcd失败")

	return client
}
