package etcd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetServiceKey(t *testing.T) {
	if !hasEtcdEnvironment() {
		t.Skip("没有可用的etcd环境，跳过测试")
	}

	cfg := getEtcdConfigFromEnv()
	client, err := NewClient(cfg)
	require.NoError(t, err, "创建etcd客户端失败")
	defer client.Close()

	key := client.GetServiceKey("test-service")
	assert.Equal(t, "/kong-discovery/services/test-service", key)
}

func TestClient_GetServicesPrefix(t *testing.T) {
	if !hasEtcdEnvironment() {
		t.Skip("没有可用的etcd环境，跳过测试")
	}

	cfg := getEtcdConfigFromEnv()
	client, err := NewClient(cfg)
	require.NoError(t, err, "创建etcd客户端失败")
	defer client.Close()

	prefix := client.GetServicesPrefix()
	assert.Equal(t, "/kong-discovery/services/", prefix)
}

func TestNewClient_ConfigValidation(t *testing.T) {
	// 正确配置
	validConfig := getEtcdConfigFromEnv()

	// 超时格式错误配置
	invalidTimeoutConfig := getEtcdConfigFromEnv()
	invalidTimeoutConfig.DialTimeout = "invalid"

	// 无法解析的超时应当返回错误
	_, err := NewClient(invalidTimeoutConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "解析etcd超时时间失败")

	_, err = NewClient(validConfig)
	t.Logf("尝试连接etcd: %v", err)
}
