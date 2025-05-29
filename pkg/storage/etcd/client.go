package etcd

import (
	"context"
	"fmt"
	"time"

	"github.com/hewenyu/kong-discovery/pkg/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Client 封装etcd客户端
type Client struct {
	client *clientv3.Client
	prefix string
}

// NewClient 创建新的etcd客户端
func NewClient(cfg *config.EtcdConfig) (*Client, error) {
	// 解析超时时间
	dialTimeout, err := time.ParseDuration(cfg.DialTimeout)
	if err != nil {
		return nil, fmt.Errorf("解析etcd超时时间失败: %w", err)
	}

	// 创建etcd客户端
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: dialTimeout,
		Username:    cfg.Username,
		Password:    cfg.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("连接etcd失败: %w", err)
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	_, err = client.Status(ctx, cfg.Endpoints[0])
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("etcd连接测试失败: %w", err)
	}

	return &Client{
		client: client,
		prefix: "/kong-discovery/services/",
	}, nil
}

// Close 关闭etcd客户端连接
func (c *Client) Close() error {
	return c.client.Close()
}

// GetClient 获取原始etcd客户端
func (c *Client) GetClient() *clientv3.Client {
	return c.client
}

// GetServiceKey 获取服务的完整存储键值
func (c *Client) GetServiceKey(serviceID string) string {
	return c.prefix + serviceID
}

// GetServicesPrefix 获取服务列表的前缀
func (c *Client) GetServicesPrefix() string {
	return c.prefix
}
