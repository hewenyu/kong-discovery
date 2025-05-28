package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// 从默认位置加载配置
	config, err := LoadConfig("")
	require.NoError(t, err, "无法加载默认配置")
	require.NotNil(t, config, "配置不应为nil")

	// 验证默认值
	assert.Equal(t, 53, config.DNS.Port, "DNS端口应为53")
	assert.Equal(t, 8080, config.API.Management.Port, "管理API端口应为8080")
	assert.Equal(t, 8081, config.API.Registration.Port, "注册API端口应为8081")
	assert.Equal(t, "both", config.DNS.Protocol, "DNS协议应为both")
	assert.Equal(t, []string{"8.8.8.8:53", "8.8.4.4:53"}, config.DNS.UpstreamDNS, "上游DNS应为默认值")
}

func TestLoadConfigFromEnvVars(t *testing.T) {
	// 设置环境变量
	os.Setenv("KONG_DISCOVERY_DNS_PORT", "5353")
	os.Setenv("KONG_DISCOVERY_MANAGEMENT_API_PORT", "9090")
	defer func() {
		os.Unsetenv("KONG_DISCOVERY_DNS_PORT")
		os.Unsetenv("KONG_DISCOVERY_MANAGEMENT_API_PORT")
	}()

	// 加载配置
	config, err := LoadConfig("")
	require.NoError(t, err, "无法加载配置")
	require.NotNil(t, config, "配置不应为nil")

	// 验证环境变量覆盖
	assert.Equal(t, 5353, config.DNS.Port, "环境变量应正确覆盖DNS端口")
	assert.Equal(t, 9090, config.API.Management.Port, "环境变量应正确覆盖管理API端口")

	// 确认其他值不受影响
	assert.Equal(t, 8081, config.API.Registration.Port, "注册API端口不应被环境变量影响")
}

func TestLoadConfigWithMissingFile(t *testing.T) {
	// 尝试从不存在的文件加载配置
	config, err := LoadConfig("non_existent_file.yaml")

	// 应该返回错误
	assert.Error(t, err, "从不存在的文件加载配置应该失败")

	// 不应该返回配置对象
	assert.Nil(t, config, "加载不存在的配置文件应该返回nil配置")
}
