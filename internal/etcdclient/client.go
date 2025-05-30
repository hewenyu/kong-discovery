package etcdclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hewenyu/kong-discovery/internal/config"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

// etcd操作的超时时间
const etcdTimeout = 5 * time.Second

// DNSRecord 表示存储在etcd中的DNS记录
type DNSRecord struct {
	Type  string   `json:"type"`           // 记录类型 (A, AAAA, SRV, CNAME等)
	Value string   `json:"value"`          // 记录值 (对于A记录是IP地址，CNAME是目标域名等)
	TTL   int      `json:"ttl"`            // 记录的TTL (秒)
	Tags  []string `json:"tags,omitempty"` // 可选标签，用于记录分组或筛选
}

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

	// GetDNSRecord 从etcd获取DNS记录
	GetDNSRecord(ctx context.Context, domain string, recordType string) (*DNSRecord, error)

	// PutDNSRecord 将DNS记录存储到etcd
	PutDNSRecord(ctx context.Context, domain string, record *DNSRecord) error

	// GetDNSRecordsForDomain 获取域名的所有DNS记录
	GetDNSRecordsForDomain(ctx context.Context, domain string) (map[string]*DNSRecord, error)

	// RegisterService 将服务实例注册到etcd
	RegisterService(ctx context.Context, instance *ServiceInstance) error

	// DeregisterService 从etcd注销服务实例
	DeregisterService(ctx context.Context, serviceName, instanceID string) error

	// GetServiceInstances 获取指定服务的所有实例
	GetServiceInstances(ctx context.Context, serviceName string) ([]*ServiceInstance, error)

	// ServiceToDNSRecords 将服务实例转换为DNS记录
	ServiceToDNSRecords(ctx context.Context, domain string) (map[string]*DNSRecord, error)

	// RefreshServiceLease 刷新服务实例的租约
	RefreshServiceLease(ctx context.Context, serviceName, instanceID string, ttl int) error
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

// getDNSRecordKey 生成DNS记录的etcd键
func getDNSRecordKey(domain, recordType string) string {
	return fmt.Sprintf("/dns/records/%s/%s", domain, recordType)
}

// GetDNSRecord 从etcd获取DNS记录
func (e *EtcdClient) GetDNSRecord(ctx context.Context, domain string, recordType string) (*DNSRecord, error) {
	if e.client == nil {
		return nil, fmt.Errorf("etcd客户端未连接")
	}

	key := getDNSRecordKey(domain, recordType)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := e.client.Get(ctx, key)
	if err != nil {
		e.logger.Error("从etcd获取DNS记录失败", zap.String("key", key), zap.Error(err))
		return nil, fmt.Errorf("从etcd获取DNS记录失败: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("DNS记录不存在: %s", key)
	}

	var record DNSRecord
	if err := json.Unmarshal(resp.Kvs[0].Value, &record); err != nil {
		e.logger.Error("解析DNS记录失败", zap.String("key", key), zap.Error(err))
		return nil, fmt.Errorf("解析DNS记录失败: %w", err)
	}

	return &record, nil
}

// PutDNSRecord 将DNS记录存储到etcd
func (e *EtcdClient) PutDNSRecord(ctx context.Context, domain string, record *DNSRecord) error {
	if e.client == nil {
		return fmt.Errorf("etcd客户端未连接")
	}

	key := getDNSRecordKey(domain, record.Type)

	recordJSON, err := json.Marshal(record)
	if err != nil {
		e.logger.Error("序列化DNS记录失败", zap.String("domain", domain), zap.Error(err))
		return fmt.Errorf("序列化DNS记录失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = e.client.Put(ctx, key, string(recordJSON))
	if err != nil {
		e.logger.Error("保存DNS记录到etcd失败", zap.String("key", key), zap.Error(err))
		return fmt.Errorf("保存DNS记录到etcd失败: %w", err)
	}

	e.logger.Info("DNS记录保存成功",
		zap.String("domain", domain),
		zap.String("type", record.Type),
		zap.String("value", record.Value))
	return nil
}

// GetDNSRecordsForDomain 获取域名的所有DNS记录
func (e *EtcdClient) GetDNSRecordsForDomain(ctx context.Context, domain string) (map[string]*DNSRecord, error) {
	if e.client == nil {
		return nil, fmt.Errorf("etcd客户端未连接")
	}

	prefix := fmt.Sprintf("/dns/records/%s/", domain)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := e.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		e.logger.Error("从etcd获取DNS记录失败", zap.String("prefix", prefix), zap.Error(err))
		return nil, fmt.Errorf("从etcd获取DNS记录失败: %w", err)
	}

	records := make(map[string]*DNSRecord)
	for _, kv := range resp.Kvs {
		var record DNSRecord
		if err := json.Unmarshal(kv.Value, &record); err != nil {
			e.logger.Error("解析DNS记录失败", zap.String("key", string(kv.Key)), zap.Error(err))
			continue
		}

		// 从key中提取记录类型
		recordType := record.Type
		records[recordType] = &record
	}

	return records, nil
}
