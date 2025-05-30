package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hewenyu/kong-discovery/internal/core/model"
	serviceStore "github.com/hewenyu/kong-discovery/internal/store/service"
)

// RegistrationService 提供服务注册相关的业务逻辑
type RegistrationService interface {
	// RegisterService 注册服务
	RegisterService(ctx context.Context, req *model.ServiceRegistrationRequest) (*model.ServiceRegistrationResponse, error)

	// DeregisterService 注销服务
	DeregisterService(ctx context.Context, serviceID string) error

	// UpdateHeartbeat 更新服务心跳
	UpdateHeartbeat(ctx context.Context, serviceID string) (*model.ServiceHeartbeatResponse, error)

	// CleanupStaleServices 清理过期服务
	CleanupStaleServices(ctx context.Context) (int, error)
}

// registrationService 实现 RegistrationService 接口
type registrationService struct {
	store            serviceStore.ServiceStore
	defaultTTL       time.Duration
	heartbeatTimeout time.Duration
}

// NewRegistrationService 创建一个新的服务注册服务
func NewRegistrationService(store serviceStore.ServiceStore, defaultTTL, heartbeatTimeout time.Duration) RegistrationService {
	return &registrationService{
		store:            store,
		defaultTTL:       defaultTTL,
		heartbeatTimeout: heartbeatTimeout,
	}
}

// RegisterService 注册服务
func (s *registrationService) RegisterService(ctx context.Context, req *model.ServiceRegistrationRequest) (*model.ServiceRegistrationResponse, error) {
	// 创建服务实例
	service := &model.Service{
		Name:      req.Name,
		Namespace: req.Namespace,
		IP:        req.IP,
		Port:      req.Port,
		Tags:      req.Tags,
		Metadata:  req.Metadata,
	}

	// 解析TTL
	if req.TTL != "" {
		ttl, err := time.ParseDuration(req.TTL)
		if err != nil {
			return nil, fmt.Errorf("解析TTL失败: %w", err)
		}
		service.TTL = ttl
	} else {
		service.TTL = s.defaultTTL
	}

	// 注册服务
	if err := s.store.Register(ctx, service); err != nil {
		return nil, fmt.Errorf("注册服务失败: %w", err)
	}

	// 返回注册响应
	return &model.ServiceRegistrationResponse{
		ServiceID:    service.ID,
		RegisteredAt: service.RegisteredAt,
	}, nil
}

// DeregisterService 注销服务
func (s *registrationService) DeregisterService(ctx context.Context, serviceID string) error {
	if err := s.store.Deregister(ctx, serviceID); err != nil {
		return fmt.Errorf("注销服务失败: %w", err)
	}
	return nil
}

// UpdateHeartbeat 更新服务心跳
func (s *registrationService) UpdateHeartbeat(ctx context.Context, serviceID string) (*model.ServiceHeartbeatResponse, error) {
	// 更新服务心跳
	if err := s.store.UpdateHeartbeat(ctx, serviceID); err != nil {
		return nil, fmt.Errorf("更新服务心跳失败: %w", err)
	}

	// 获取服务信息
	service, err := s.store.GetService(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("获取服务信息失败: %w", err)
	}

	// 返回心跳响应
	return &model.ServiceHeartbeatResponse{
		LastHeartbeat: service.LastHeartbeat,
	}, nil
}

// CleanupStaleServices 清理过期服务
// 注意：etcd会根据TTL自动清理过期服务，本方法主要作为备份机制，
// 防止由于某些原因（如网络故障、etcd异常等）导致的服务未能及时清理
func (s *registrationService) CleanupStaleServices(ctx context.Context) (int, error) {
	// 计算过期时间
	now := time.Now()
	before := now.Add(-s.heartbeatTimeout)

	// 添加详细日志
	log.Printf("执行过期服务清理，当前时间: %v, 心跳超时时间: %v, 心跳阈值时间: %v (心跳时间早于此视为过期)",
		now.Format("2006-01-02 15:04:05"), s.heartbeatTimeout, before.Format("2006-01-02 15:04:05"))
	log.Printf("注意：此方法仅作为备份机制，主要依靠etcd根据TTL自动清理过期服务")

	// 清理过期服务
	count, err := s.store.CleanupStaleServices(ctx, before)
	if err != nil {
		return 0, fmt.Errorf("清理过期服务失败: %w", err)
	}

	return count, nil
}
