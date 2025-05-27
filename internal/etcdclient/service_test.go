package etcdclient

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServiceToDNSRecords 测试将服务实例转换为DNS记录的功能
func TestServiceToDNSRecords(t *testing.T) {
	// 跳过集成测试，除非明确要求运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建etcd客户端并连接
	client := CreateEtcdClientForTest(t)
	defer client.Close()

	// 创建测试服务名，确保唯一性
	testServiceName := fmt.Sprintf("test-service-%d", time.Now().UnixNano())
	testDomain := testServiceName + ".default.svc.cluster.local"

	// 注册两个测试服务实例
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 第一个实例
	instance1 := &ServiceInstance{
		ServiceName: testServiceName,
		InstanceID:  "instance-1",
		IPAddress:   "192.168.1.100",
		Port:        8080,
		TTL:         60,
	}
	err := client.RegisterService(ctx, instance1)
	require.NoError(t, err, "注册第一个服务实例失败")

	// 第二个实例
	instance2 := &ServiceInstance{
		ServiceName: testServiceName,
		InstanceID:  "instance-2",
		IPAddress:   "192.168.1.101",
		Port:        8080,
		TTL:         60,
	}
	err = client.RegisterService(ctx, instance2)
	require.NoError(t, err, "注册第二个服务实例失败")

	// 确保测试结束后清理
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()

		_ = client.DeregisterService(cleanupCtx, testServiceName, "instance-1")
		_ = client.DeregisterService(cleanupCtx, testServiceName, "instance-2")
	}()

	// 测试转换服务到DNS记录
	records, err := client.ServiceToDNSRecords(ctx, testDomain)
	require.NoError(t, err)
	require.NotNil(t, records)

	// 验证A记录
	aRecord, ok := records["A"]
	require.True(t, ok, "应该存在A记录")
	assert.Equal(t, "A", aRecord.Type)
	assert.Equal(t, "192.168.1.100", aRecord.Value) // 应该使用第一个实例的IP

	// 验证SRV记录
	foundSRV := false
	for key, record := range records {
		if strings.HasPrefix(key, "SRV-") {
			foundSRV = true
			assert.Equal(t, "SRV", record.Type)
			assert.Contains(t, record.Value, "8080") // 端口号应该包含在SRV记录中
		}
	}
	assert.True(t, foundSRV, "应该存在SRV记录")
}
