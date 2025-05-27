package config

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewLogger(t *testing.T) {
	// 测试开发环境日志初始化
	devLogger, err := NewLogger(true)
	if err != nil {
		t.Fatalf("开发环境日志初始化失败: %v", err)
	}
	if devLogger == nil {
		t.Fatal("开发环境日志初始化返回nil")
	}

	// 测试生产环境日志初始化
	prodLogger, err := NewLogger(false)
	if err != nil {
		t.Fatalf("生产环境日志初始化失败: %v", err)
	}
	if prodLogger == nil {
		t.Fatal("生产环境日志初始化返回nil")
	}

	// 测试日志输出 (这里只能验证不会崩溃，无法验证实际输出内容)
	devLogger.Info("测试开发环境日志", zap.String("test", "value"))
	prodLogger.Info("测试生产环境日志", zap.String("test", "value"))
}
