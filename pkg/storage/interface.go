package storage

import (
	"context"
	"time"
)

// Service 表示一个服务实例
type Service struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	IP            string            `json:"ip"`
	Port          int               `json:"port"`
	Tags          []string          `json:"tags"`
	Metadata      map[string]string `json:"metadata"`
	Health        string            `json:"health"`
	RegisteredAt  time.Time         `json:"registered_at"`
	LastHeartbeat time.Time         `json:"last_heartbeat"`
	TTL           int               `json:"ttl"`
}

// ServiceStorage 定义服务存储接口
type ServiceStorage interface {
	// RegisterService 注册服务实例
	RegisterService(ctx context.Context, service *Service) error

	// DeregisterService 注销服务实例
	DeregisterService(ctx context.Context, serviceID string) error

	// GetService 获取服务实例详情
	GetService(ctx context.Context, serviceID string) (*Service, error)

	// ListServices 获取所有服务实例列表
	ListServices(ctx context.Context) ([]*Service, error)

	// ListServicesByName 获取指定名称的服务实例列表
	ListServicesByName(ctx context.Context, serviceName string) ([]*Service, error)

	// UpdateServiceHeartbeat 更新服务心跳时间
	UpdateServiceHeartbeat(ctx context.Context, serviceID string) error

	// CleanupStaleServices 清理过期的服务实例
	CleanupStaleServices(ctx context.Context, timeout time.Duration) error
}

// StorageError 定义存储操作可能返回的错误类型
type StorageError struct {
	Code    int
	Message string
}

// Error 实现error接口
func (e *StorageError) Error() string {
	return e.Message
}

// 定义错误代码
const (
	// ErrNotFound 资源不存在
	ErrNotFound = iota + 1
	// ErrAlreadyExists 资源已存在
	ErrAlreadyExists
	// ErrInvalidArgument 参数无效
	ErrInvalidArgument
	// ErrInternal 内部错误
	ErrInternal
)

// NewNotFoundError 创建资源不存在错误
func NewNotFoundError(message string) *StorageError {
	return &StorageError{
		Code:    ErrNotFound,
		Message: message,
	}
}

// NewAlreadyExistsError 创建资源已存在错误
func NewAlreadyExistsError(message string) *StorageError {
	return &StorageError{
		Code:    ErrAlreadyExists,
		Message: message,
	}
}

// NewInvalidArgumentError 创建参数无效错误
func NewInvalidArgumentError(message string) *StorageError {
	return &StorageError{
		Code:    ErrInvalidArgument,
		Message: message,
	}
}

// NewInternalError 创建内部错误
func NewInternalError(message string) *StorageError {
	return &StorageError{
		Code:    ErrInternal,
		Message: message,
	}
}
