package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewLogger(t *testing.T) {
	// 测试开发环境日志初始化
	devLogger, err := NewLogger(true)
	require.NoError(t, err, "开发环境日志初始化应成功")
	require.NotNil(t, devLogger, "开发环境日志不应为nil")

	// 测试生产环境日志初始化
	prodLogger, err := NewLogger(false)
	require.NoError(t, err, "生产环境日志初始化应成功")
	require.NotNil(t, prodLogger, "生产环境日志不应为nil")

	// 测试日志接口方法
	// 这里我们只测试方法不会崩溃，无法直接验证日志内容
	testLoggerMethods(t, devLogger)
	testLoggerMethods(t, prodLogger)
}

func testLoggerMethods(t *testing.T, logger Logger) {
	t.Helper()

	// 确保所有日志方法都不会抛出异常
	assert.NotPanics(t, func() {
		logger.Debug("测试Debug日志", zap.String("key", "value"))
		logger.Info("测试Info日志", zap.String("key", "value"))
		logger.Warn("测试Warn日志", zap.String("key", "value"))
		logger.Error("测试Error日志", zap.String("key", "value"))
		// 不测试Fatal，它会调用os.Exit
	}, "日志方法不应panic")
}
