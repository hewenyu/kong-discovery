package config

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 定义日志接口
type Logger interface {
	Debug(msg string, fields ...zapcore.Field)
	Info(msg string, fields ...zapcore.Field)
	Warn(msg string, fields ...zapcore.Field)
	Error(msg string, fields ...zapcore.Field)
	Fatal(msg string, fields ...zapcore.Field)
}

// ZapLogger 实现Logger接口
type ZapLogger struct {
	logger *zap.Logger
}

// NewLogger 创建并返回一个新的Logger实例
func NewLogger(isDevelopment bool) (Logger, error) {
	var config zap.Config
	if isDevelopment {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	zapLogger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &ZapLogger{
		logger: zapLogger,
	}, nil
}

// Debug 记录Debug级别日志
func (l *ZapLogger) Debug(msg string, fields ...zapcore.Field) {
	l.logger.Debug(msg, fields...)
}

// Info 记录Info级别日志
func (l *ZapLogger) Info(msg string, fields ...zapcore.Field) {
	l.logger.Info(msg, fields...)
}

// Warn 记录Warn级别日志
func (l *ZapLogger) Warn(msg string, fields ...zapcore.Field) {
	l.logger.Warn(msg, fields...)
}

// Error 记录Error级别日志
func (l *ZapLogger) Error(msg string, fields ...zapcore.Field) {
	l.logger.Error(msg, fields...)
}

// Fatal 记录Fatal级别日志
func (l *ZapLogger) Fatal(msg string, fields ...zapcore.Field) {
	l.logger.Fatal(msg, fields...)
}
