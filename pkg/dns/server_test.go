package dns

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hewenyu/kong-discovery/pkg/config"
	"github.com/hewenyu/kong-discovery/pkg/storage/etcd"
)

func TestDNSServer(t *testing.T) {
	// 跳过CI环境测试
	if os.Getenv("CI") == "true" {
		t.Skip("在CI环境中跳过etcd集成测试")
	}

	// 创建测试配置
	conf := &config.Config{
		Server: config.ServerConfig{
			RegisterPort: 8080,
			AdminPort:    9090,
			DNSPort:      15353, // 使用非标准端口避免冲突
		},
		DNS: config.DNSConfig{
			Domain:   "service.test",
			Upstream: []string{"8.8.8.8:53", "114.114.114.114:53"},
			CacheTTL: 60,
		},
		Heartbeat: config.HeartbeatConfig{
			Interval: 30,
			Timeout:  90,
		},
	}

	// 获取etcd配置
	etcdConfig := getEtcdConfig()
	etcdConfig.DialTimeout = "10s" // 增加超时时间
	client, err := etcd.NewClient(etcdConfig)
	require.NoError(t, err, "连接etcd失败")
	defer client.Close()

	// 测试etcd连接是否正常
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.GetClient().Status(ctx, etcdConfig.Endpoints[0])
	if err != nil {
		t.Skipf("etcd连接测试失败，跳过测试: %v", err)
		return
	}

	// 创建服务存储
	serviceStorage := etcd.NewServiceStorage(client)
	// 创建命名空间存储
	namespaceStorage := etcd.NewNamespaceStorage(client)

	// 创建服务注册数据
	service := prepareTestService(t, serviceStorage)
	defer cleanupTestService(t, serviceStorage, service.ID)

	// 创建DNS服务器
	server, err := NewServer(conf, serviceStorage, namespaceStorage)
	require.NoError(t, err, "创建DNS服务器失败")

	// 启动服务器
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	err = server.Start(ctx)
	require.NoError(t, err, "启动DNS服务器失败")

	// 让服务器运行一段时间
	time.Sleep(1 * time.Second)

	// 停止服务器
	err = server.Stop()
	require.NoError(t, err, "停止DNS服务器失败")
}
