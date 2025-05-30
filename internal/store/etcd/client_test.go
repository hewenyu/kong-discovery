package etcd

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/hewenyu/kong-discovery/internal/core/config"
)

// 这些测试需要一个正在运行的etcd实例
// 可以通过docker运行: docker run -d --name etcd-test -p 2379:2379 bitnami/etcd:3.5 --allow-none-authentication

func getEtcdClient() (*Client, error) {

	if os.Getenv("ETCD_ENDPOINTS") != "" {
		cfg := &config.EtcdConfig{
			Endpoints:      []string{os.Getenv("ETCD_ENDPOINTS")},
			DialTimeout:    5 * time.Second,
			RequestTimeout: 10 * time.Second,
		}
		return NewClient(cfg)
	}
	return nil, errors.New("ETCD_ENDPOINTS 未设置")
}

func TestEtcdClient(t *testing.T) {
	// 如果没有etcd实例运行，跳过测试
	client, err := getEtcdClient()
	if err != nil {
		t.Skip("跳过测试，无法连接到etcd: ", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// 测试基本的Put和Get操作
	testKey := "/test/key1"
	testValue := []byte("test-value-1")

	// 删除可能存在的测试键
	_ = client.Delete(ctx, testKey)

	// 测试Put
	err = client.Put(ctx, testKey, testValue)
	if err != nil {
		t.Fatalf("Put失败: %v", err)
	}

	// 测试Get
	value, err := client.Get(ctx, testKey)
	if err != nil {
		t.Fatalf("Get失败: %v", err)
	}
	if string(value) != string(testValue) {
		t.Fatalf("Get返回值不一致，期望 %s，实际 %s", testValue, value)
	}

	// 测试Delete
	err = client.Delete(ctx, testKey)
	if err != nil {
		t.Fatalf("Delete失败: %v", err)
	}

	// 确认键已被删除
	value, err = client.Get(ctx, testKey)
	if err != nil {
		t.Fatalf("Get失败: %v", err)
	}
	if value != nil {
		t.Fatalf("键应该已被删除，但仍然存在值: %s", value)
	}

	// 测试PutWithLease
	testLeaseKey := "/test/lease-key"
	testLeaseValue := []byte("test-lease-value")

	// 设置短租约（1秒）以便测试过期
	err = client.PutWithLease(ctx, testLeaseKey, testLeaseValue, 1*time.Second)
	if err != nil {
		t.Fatalf("PutWithLease失败: %v", err)
	}

	// 验证键值已设置
	value, err = client.Get(ctx, testLeaseKey)
	if err != nil {
		t.Fatalf("Get失败: %v", err)
	}
	if string(value) != string(testLeaseValue) {
		t.Fatalf("Get返回值不一致，期望 %s，实际 %s", testLeaseValue, value)
	}

	// 等待租约过期
	time.Sleep(2 * time.Second)

	// 确认键已过期
	value, err = client.Get(ctx, testLeaseKey)
	if err != nil {
		t.Fatalf("Get失败: %v", err)
	}
	if value != nil {
		t.Fatalf("租约应该已过期，但键仍然存在值: %s", value)
	}

	t.Log("etcd客户端基本功能测试通过")
}

func TestEtcdClientPrefix(t *testing.T) {
	// 如果没有etcd实例运行，跳过测试
	client, err := getEtcdClient()
	if err != nil {
		t.Skip("跳过测试，无法连接到etcd: ", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// 创建测试前缀
	testPrefix := "/test/prefix/"
	testKeys := []string{
		testPrefix + "key1",
		testPrefix + "key2",
		testPrefix + "key3",
	}
	testValues := [][]byte{
		[]byte("value1"),
		[]byte("value2"),
		[]byte("value3"),
	}

	// 清理测试前缀
	_ = client.DeleteWithPrefix(ctx, testPrefix)

	// 设置测试键值
	for i, key := range testKeys {
		err := client.Put(ctx, key, testValues[i])
		if err != nil {
			t.Fatalf("设置测试键值失败 [%s]: %v", key, err)
		}
	}

	// 测试GetWithPrefix
	values, err := client.GetWithPrefix(ctx, testPrefix)
	if err != nil {
		t.Fatalf("GetWithPrefix失败: %v", err)
	}

	// 验证结果
	if len(values) != len(testKeys) {
		t.Fatalf("GetWithPrefix返回键数量不一致，期望 %d，实际 %d", len(testKeys), len(values))
	}

	for i, key := range testKeys {
		value, exists := values[key]
		if !exists {
			t.Fatalf("键 %s 在结果中不存在", key)
		}
		if string(value) != string(testValues[i]) {
			t.Fatalf("键 %s 的值不一致，期望 %s，实际 %s", key, testValues[i], value)
		}
	}

	// 测试DeleteWithPrefix
	err = client.DeleteWithPrefix(ctx, testPrefix)
	if err != nil {
		t.Fatalf("DeleteWithPrefix失败: %v", err)
	}

	// 验证所有键都已被删除
	values, err = client.GetWithPrefix(ctx, testPrefix)
	if err != nil {
		t.Fatalf("GetWithPrefix失败: %v", err)
	}
	if len(values) != 0 {
		t.Fatalf("DeleteWithPrefix后仍有键存在: %v", values)
	}

	t.Log("etcd客户端前缀操作测试通过")
}
