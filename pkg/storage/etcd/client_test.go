package etcd

import (
	"testing"

	"github.com/hewenyu/kong-discovery/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestClient_GetServiceKey(t *testing.T) {
	client := &Client{
		prefix: "/kong-discovery/services/",
	}

	key := client.GetServiceKey("test-service")
	assert.Equal(t, "/kong-discovery/services/test-service", key)
}

func TestClient_GetServicesPrefix(t *testing.T) {
	client := &Client{
		prefix: "/kong-discovery/services/",
	}

	prefix := client.GetServicesPrefix()
	assert.Equal(t, "/kong-discovery/services/", prefix)
}

func TestNewClient_ConfigValidation(t *testing.T) {
	// 正确配置
	validConfig := &config.EtcdConfig{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: "5s",
	}

	// 超时格式错误配置
	invalidTimeoutConfig := &config.EtcdConfig{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: "invalid",
	}

	// 无法解析的超时应当返回错误
	_, err := NewClient(invalidTimeoutConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "解析etcd超时时间失败")

	// 这个测试只是验证代码逻辑，实际上会因为没有真实的etcd服务而失败
	// 如果在实际环境中有可用的etcd服务，这个测试会通过
	// 此处我们只是确认参数传递正确
	_, err = NewClient(validConfig)
	t.Logf("尝试连接etcd: %v", err)
}
