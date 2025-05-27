package etcdclient

import (
	"context"
	"fmt"
	"time"

	"github.com/hewenyu/kong-discovery/internal/config"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

// Client 定义etcd客户端接口
type Client interface {
	// Connect 连接到etcd集群
	Connect() error

	// Close 关闭连接
	Close() error

	// Ping 检查etcd集群状态
	Ping(ctx context.Context) error

	// Get 从etcd获取指定key的值
	Get(ctx context.Context, key string) (string, error)

	// GetWithPrefix 从etcd获取指定前缀的所有key-value
	GetWithPrefix(ctx context.Context, prefix string) (map[string]string, error)
}

// EtcdClient 实现Client接口
type EtcdClient struct {
	client *clientv3.Client
	cfg    *config.Config
	logger config.Logger
}

// NewEtcdClient 创建一个新的etcd客户端
func NewEtcdClient(cfg *config.Config, logger config.Logger) Client {
	return &EtcdClient{
		cfg:    cfg,
		logger: logger,
	}
}

// Connect 连接到etcd集群
func (e *EtcdClient) Connect() error {
	var err error
	e.logger.Info("连接到etcd集群", zap.Strings("endpoints", e.cfg.Etcd.Endpoints))

	e.client, err = clientv3.New(clientv3.Config{
		Endpoints:   e.cfg.Etcd.Endpoints,
		DialTimeout: 5 * time.Second,
		Username:    e.cfg.Etcd.Username,
		Password:    e.cfg.Etcd.Password,
	})

	if err != nil {
		e.logger.Error("连接etcd失败", zap.Error(err))
		return fmt.Errorf("连接etcd失败: %w", err)
	}

	return nil
}

// Close 关闭连接
func (e *EtcdClient) Close() error {
	if e.client != nil {
		e.logger.Info("关闭etcd连接")
		return e.client.Close()
	}
	return nil
}

// Ping 检查etcd集群状态
func (e *EtcdClient) Ping(ctx context.Context) error {
	if e.client == nil {
		return fmt.Errorf("etcd客户端未连接")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := e.client.Status(ctx, e.cfg.Etcd.Endpoints[0])
	if err != nil {
		e.logger.Error("etcd健康检查失败", zap.Error(err))
		return fmt.Errorf("etcd健康检查失败: %w", err)
	}

	e.logger.Info("etcd健康检查成功")
	return nil
}

// Get 从etcd获取指定key的值
func (e *EtcdClient) Get(ctx context.Context, key string) (string, error) {
	if e.client == nil {
		return "", fmt.Errorf("etcd客户端未连接")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := e.client.Get(ctx, key)
	if err != nil {
		e.logger.Error("从etcd获取数据失败", zap.String("key", key), zap.Error(err))
		return "", fmt.Errorf("从etcd获取数据失败: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return "", fmt.Errorf("key不存在: %s", key)
	}

	return string(resp.Kvs[0].Value), nil
}

// GetWithPrefix 从etcd获取指定前缀的所有key-value
func (e *EtcdClient) GetWithPrefix(ctx context.Context, prefix string) (map[string]string, error) {
	if e.client == nil {
		return nil, fmt.Errorf("etcd客户端未连接")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := e.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		e.logger.Error("从etcd获取前缀数据失败", zap.String("prefix", prefix), zap.Error(err))
		return nil, fmt.Errorf("从etcd获取前缀数据失败: %w", err)
	}

	result := make(map[string]string)
	for _, kv := range resp.Kvs {
		result[string(kv.Key)] = string(kv.Value)
	}

	return result, nil
}
