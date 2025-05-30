package etcd

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/hewenyu/kong-discovery/internal/core/config"
)

// Client 封装了etcd客户端
type Client struct {
	client *clientv3.Client
	cfg    *config.EtcdConfig
}

// NewClient 创建一个新的etcd客户端
func NewClient(cfg *config.EtcdConfig) (*Client, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.DialTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("创建etcd客户端失败: %w", err)
	}

	return &Client{
		client: client,
		cfg:    cfg,
	}, nil
}

// Close 关闭etcd客户端连接
func (c *Client) Close() error {
	return c.client.Close()
}

// Get 获取键值
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
	defer cancel()

	resp, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("etcd获取键值失败 [%s]: %w", key, err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil // 键不存在
	}

	return resp.Kvs[0].Value, nil
}

// GetWithPrefix 获取指定前缀的所有键值
func (c *Client) GetWithPrefix(ctx context.Context, prefix string) (map[string][]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
	defer cancel()

	resp, err := c.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("etcd获取前缀键值失败 [%s]: %w", prefix, err)
	}

	result := make(map[string][]byte, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		result[string(kv.Key)] = kv.Value
	}

	return result, nil
}

// Put 设置键值
func (c *Client) Put(ctx context.Context, key string, value []byte) error {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
	defer cancel()

	_, err := c.client.Put(ctx, key, string(value))
	if err != nil {
		return fmt.Errorf("etcd设置键值失败 [%s]: %w", key, err)
	}

	return nil
}

// PutWithLease 设置带租约的键值
func (c *Client) PutWithLease(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
	defer cancel()

	// 创建租约
	lease, err := c.client.Grant(ctx, int64(ttl.Seconds()))
	if err != nil {
		return fmt.Errorf("etcd创建租约失败: %w", err)
	}

	// 设置带租约的键值
	_, err = c.client.Put(ctx, key, string(value), clientv3.WithLease(lease.ID))
	if err != nil {
		return fmt.Errorf("etcd设置带租约的键值失败 [%s]: %w", key, err)
	}

	return nil
}

// Delete 删除键值
func (c *Client) Delete(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
	defer cancel()

	_, err := c.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("etcd删除键值失败 [%s]: %w", key, err)
	}

	return nil
}

// DeleteWithPrefix 删除指定前缀的所有键值
func (c *Client) DeleteWithPrefix(ctx context.Context, prefix string) error {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
	defer cancel()

	_, err := c.client.Delete(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("etcd删除前缀键值失败 [%s]: %w", prefix, err)
	}

	return nil
}

// Watch 监听键的变化
func (c *Client) Watch(ctx context.Context, key string) clientv3.WatchChan {
	return c.client.Watch(ctx, key)
}

// WatchWithPrefix 监听指定前缀的键的变化
func (c *Client) WatchWithPrefix(ctx context.Context, prefix string) clientv3.WatchChan {
	return c.client.Watch(ctx, prefix, clientv3.WithPrefix())
}

// KeepAlive 保持租约活跃
func (c *Client) KeepAlive(ctx context.Context, leaseID clientv3.LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	return c.client.KeepAlive(ctx, leaseID)
}

// Grant 创建租约
func (c *Client) Grant(ctx context.Context, ttl time.Duration) (*clientv3.LeaseGrantResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
	defer cancel()

	return c.client.Grant(ctx, int64(ttl.Seconds()))
}

// RevokeLease 撤销租约
func (c *Client) RevokeLease(ctx context.Context, leaseID clientv3.LeaseID) error {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
	defer cancel()

	_, err := c.client.Revoke(ctx, leaseID)
	if err != nil {
		return fmt.Errorf("etcd撤销租约失败 [%d]: %w", leaseID, err)
	}

	return nil
}

// GetLeaseTimeToLive 获取租约的剩余时间
func (c *Client) GetLeaseTimeToLive(ctx context.Context, leaseID clientv3.LeaseID) (*clientv3.LeaseTimeToLiveResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
	defer cancel()

	return c.client.TimeToLive(ctx, leaseID)
}
