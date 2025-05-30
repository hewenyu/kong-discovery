package registration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/hewenyu/kong-discovery/internal/core/config"
	"github.com/hewenyu/kong-discovery/internal/core/model"
	"github.com/hewenyu/kong-discovery/internal/registration"
	"github.com/hewenyu/kong-discovery/internal/store/etcd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试基础URL
const baseURL = "http://localhost:18080/api/v1"

// 获取etcd客户端
func getEtcdClient() (*etcd.Client, error) {
	etcdEndpoints := os.Getenv("ETCD_ENDPOINTS")
	if etcdEndpoints == "" {
		// 默认使用本地etcd
		etcdEndpoints = "localhost:2379"
	}

	return etcd.NewClient(&config.EtcdConfig{
		Endpoints:      []string{etcdEndpoints},
		DialTimeout:    5 * time.Second,
		RequestTimeout: 5 * time.Second,
	})
}

// 跳过测试如果无法连接etcd
func skipIfNoEtcd(t *testing.T) *etcd.Client {
	etcdClient, err := getEtcdClient()
	if err != nil {
		t.Skip("跳过测试：无法连接etcd")
		return nil
	}

	// 测试etcd连接
	ctx := context.Background()
	testKey := "/test/api-test-connection"
	testValue := []byte("test-connection")
	err = etcdClient.Put(ctx, testKey, testValue)
	if err != nil {
		etcdClient.Close()
		t.Skip("跳过测试：etcd连接测试失败")
		return nil
	}

	value, err := etcdClient.Get(ctx, testKey)
	if err != nil || string(value) != string(testValue) {
		etcdClient.Close()
		t.Skip("跳过测试：etcd读取测试失败")
		return nil
	}

	err = etcdClient.Delete(ctx, testKey)
	if err != nil {
		etcdClient.Close()
		t.Skip("跳过测试：etcd删除测试失败")
		return nil
	}

	return etcdClient
}

// 启动服务注册API服务器
func startRegistrationServer(t *testing.T, etcdClient *etcd.Client) *registration.Server {
	// 创建配置
	cfg := &config.Config{
		Server: config.ServerConfig{
			Registration: config.RegistrationConfig{
				Host: "localhost",
				Port: 18080, // 使用非标准端口避免冲突
			},
		},
		Namespace: config.NamespaceConfig{
			Default: "default",
		},
		Service: config.ServiceConfig{
			Heartbeat: config.HeartbeatConfig{
				Interval: 30 * time.Second,
				Timeout:  90 * time.Second,
			},
		},
	}

	// 创建服务器
	server := registration.NewServer(etcdClient, cfg)
	require.NotNil(t, server, "创建服务注册服务器失败")

	// 启动服务器
	err := server.Start()
	require.NoError(t, err, "启动服务注册服务器失败")

	// 等待服务器启动
	time.Sleep(1 * time.Second)

	return server
}

// TestRegistrationAPI 测试服务注册API的功能
func TestRegistrationAPI(t *testing.T) {
	// 获取etcd客户端，如果连接失败则跳过测试
	etcdClient := skipIfNoEtcd(t)
	if etcdClient == nil {
		return
	}
	defer etcdClient.Close()

	// 启动服务注册API服务器
	server := startRegistrationServer(t, etcdClient)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	// 测试服务注册
	t.Run("RegisterService", func(t *testing.T) {
		// 创建服务注册请求
		serviceReq := model.ServiceRegistrationRequest{
			Name:      "test-service-api",
			Namespace: "test-namespace",
			IP:        "192.168.1.100",
			Port:      8080,
			Tags:      []string{"test", "api"},
			Metadata: map[string]string{
				"version":     "1.0.0",
				"environment": "test",
			},
			TTL: "30s",
		}

		// 转换为JSON
		jsonData, err := json.Marshal(serviceReq)
		require.NoError(t, err, "序列化服务注册请求失败")

		// 发送POST请求
		resp, err := http.Post(
			fmt.Sprintf("%s/services", baseURL),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		require.NoError(t, err, "发送服务注册请求失败")
		defer resp.Body.Close()

		// 检查响应状态码
		assert.Equal(t, http.StatusOK, resp.StatusCode, "服务注册响应状态码错误")

		// 解析响应
		var apiResp model.ApiResponse
		err = json.NewDecoder(resp.Body).Decode(&apiResp)
		require.NoError(t, err, "解析服务注册响应失败")

		// 检查响应内容
		assert.Equal(t, http.StatusOK, apiResp.Code, "服务注册响应代码错误")
		assert.Equal(t, "服务注册成功", apiResp.Message, "服务注册响应消息错误")
		assert.NotNil(t, apiResp.Data, "服务注册响应数据为空")

		// 提取服务ID
		regResp, ok := apiResp.Data.(map[string]interface{})
		require.True(t, ok, "响应数据格式错误")

		serviceID, ok := regResp["service_id"].(string)
		require.True(t, ok, "无法获取服务ID")
		assert.NotEmpty(t, serviceID, "服务ID为空")

		// 用于后续测试
		t.Logf("注册的服务ID: %s", serviceID)

		// 测试心跳更新
		t.Run("UpdateHeartbeat", func(t *testing.T) {
			// 发送PUT请求更新心跳
			req, err := http.NewRequest(
				http.MethodPut,
				fmt.Sprintf("%s/services/%s/heartbeat", baseURL, serviceID),
				nil,
			)
			require.NoError(t, err, "创建心跳请求失败")

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err, "发送心跳请求失败")
			defer resp.Body.Close()

			// 检查响应状态码
			assert.Equal(t, http.StatusOK, resp.StatusCode, "心跳响应状态码错误")

			// 解析响应
			var apiResp model.ApiResponse
			err = json.NewDecoder(resp.Body).Decode(&apiResp)
			require.NoError(t, err, "解析心跳响应失败")

			// 检查响应内容
			assert.Equal(t, http.StatusOK, apiResp.Code, "心跳响应代码错误")
			assert.Equal(t, "心跳更新成功", apiResp.Message, "心跳响应消息错误")
			assert.NotNil(t, apiResp.Data, "心跳响应数据为空")

			// 提取最后心跳时间
			heartbeatResp, ok := apiResp.Data.(map[string]interface{})
			require.True(t, ok, "心跳响应数据格式错误")

			lastHeartbeat, ok := heartbeatResp["last_heartbeat"].(string)
			require.True(t, ok, "无法获取最后心跳时间")
			assert.NotEmpty(t, lastHeartbeat, "最后心跳时间为空")
			t.Logf("最后心跳时间: %s", lastHeartbeat)
		})

		// 测试错误心跳
		t.Run("InvalidHeartbeat", func(t *testing.T) {
			// 发送PUT请求更新不存在的服务的心跳
			invalidID := "non-existent-service-id"
			req, err := http.NewRequest(
				http.MethodPut,
				fmt.Sprintf("%s/services/%s/heartbeat", baseURL, invalidID),
				nil,
			)
			require.NoError(t, err, "创建心跳请求失败")

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err, "发送心跳请求失败")
			defer resp.Body.Close()

			// 检查响应状态码 - 应该是错误
			assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, "非法心跳响应状态码错误")
		})

		// 测试服务注销
		t.Run("DeregisterService", func(t *testing.T) {
			// 发送DELETE请求注销服务
			req, err := http.NewRequest(
				http.MethodDelete,
				fmt.Sprintf("%s/services/%s", baseURL, serviceID),
				nil,
			)
			require.NoError(t, err, "创建注销请求失败")

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err, "发送注销请求失败")
			defer resp.Body.Close()

			// 检查响应状态码
			assert.Equal(t, http.StatusOK, resp.StatusCode, "注销响应状态码错误")

			// 解析响应
			var apiResp model.ApiResponse
			err = json.NewDecoder(resp.Body).Decode(&apiResp)
			require.NoError(t, err, "解析注销响应失败")

			// 检查响应内容
			assert.Equal(t, http.StatusOK, apiResp.Code, "注销响应代码错误")
			assert.Equal(t, "服务注销成功", apiResp.Message, "注销响应消息错误")
		})

		// 测试注销不存在的服务
		t.Run("DeregisterNonExistentService", func(t *testing.T) {
			// 发送DELETE请求注销不存在的服务
			invalidID := "non-existent-service-id"
			req, err := http.NewRequest(
				http.MethodDelete,
				fmt.Sprintf("%s/services/%s", baseURL, invalidID),
				nil,
			)
			require.NoError(t, err, "创建注销请求失败")

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err, "发送注销请求失败")
			defer resp.Body.Close()

			// 检查响应状态码 - 应该是错误
			assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, "非法注销响应状态码错误")
		})
	})

	// 测试无效的服务注册请求
	t.Run("InvalidRegistrationRequest", func(t *testing.T) {
		// 1. 缺少服务名称
		t.Run("MissingName", func(t *testing.T) {
			invalidReq := model.ServiceRegistrationRequest{
				IP:   "192.168.1.100",
				Port: 8080,
			}

			jsonData, _ := json.Marshal(invalidReq)
			resp, err := http.Post(
				fmt.Sprintf("%s/services", baseURL),
				"application/json",
				bytes.NewBuffer(jsonData),
			)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		// 2. 缺少IP地址
		t.Run("MissingIP", func(t *testing.T) {
			invalidReq := model.ServiceRegistrationRequest{
				Name: "test-service",
				Port: 8080,
			}

			jsonData, _ := json.Marshal(invalidReq)
			resp, err := http.Post(
				fmt.Sprintf("%s/services", baseURL),
				"application/json",
				bytes.NewBuffer(jsonData),
			)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		// 3. 无效的端口号
		t.Run("InvalidPort", func(t *testing.T) {
			invalidReq := model.ServiceRegistrationRequest{
				Name: "test-service",
				IP:   "192.168.1.100",
				Port: 0, // 无效端口
			}

			jsonData, _ := json.Marshal(invalidReq)
			resp, err := http.Post(
				fmt.Sprintf("%s/services", baseURL),
				"application/json",
				bytes.NewBuffer(jsonData),
			)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		// 4. 无效的TTL格式
		t.Run("InvalidTTL", func(t *testing.T) {
			invalidReq := model.ServiceRegistrationRequest{
				Name: "test-service",
				IP:   "192.168.1.100",
				Port: 8080,
				TTL:  "invalid-ttl", // 无效TTL
			}

			jsonData, _ := json.Marshal(invalidReq)
			resp, err := http.Post(
				fmt.Sprintf("%s/services", baseURL),
				"application/json",
				bytes.NewBuffer(jsonData),
			)
			require.NoError(t, err)
			defer resp.Body.Close()

			// 这可能会导致内部服务器错误，因为TTL解析失败
			assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		})
	})

	// 测试并发注册
	t.Run("ConcurrentRegistration", func(t *testing.T) {
		// 并发数量
		concurrency := 5
		done := make(chan bool, concurrency)
		success := make(chan string, concurrency)
		failed := make(chan error, concurrency)

		// 启动并发请求
		for i := 0; i < concurrency; i++ {
			go func(index int) {
				serviceReq := model.ServiceRegistrationRequest{
					Name:      fmt.Sprintf("concurrent-service-%d", index),
					Namespace: "test-namespace",
					IP:        fmt.Sprintf("192.168.1.%d", 100+index),
					Port:      8080 + index,
					Tags:      []string{"concurrent", "test"},
					TTL:       "30s",
				}

				jsonData, err := json.Marshal(serviceReq)
				if err != nil {
					failed <- fmt.Errorf("序列化请求失败: %v", err)
					done <- true
					return
				}

				resp, err := http.Post(
					fmt.Sprintf("%s/services", baseURL),
					"application/json",
					bytes.NewBuffer(jsonData),
				)
				if err != nil {
					failed <- fmt.Errorf("发送请求失败: %v", err)
					done <- true
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					failed <- fmt.Errorf("响应状态码错误: %d", resp.StatusCode)
					done <- true
					return
				}

				var apiResp model.ApiResponse
				if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
					failed <- fmt.Errorf("解析响应失败: %v", err)
					done <- true
					return
				}

				regResp, ok := apiResp.Data.(map[string]interface{})
				if !ok {
					failed <- fmt.Errorf("响应数据格式错误")
					done <- true
					return
				}

				serviceID, ok := regResp["service_id"].(string)
				if !ok || serviceID == "" {
					failed <- fmt.Errorf("无法获取服务ID")
					done <- true
					return
				}

				success <- serviceID
				done <- true
			}(i)
		}

		// 等待所有请求完成
		registeredIDs := []string{}
		for i := 0; i < concurrency; i++ {
			<-done
		}

		// 收集成功注册的服务ID
		close(success)
		for id := range success {
			registeredIDs = append(registeredIDs, id)

			// 确保测试结束后清理注册的服务
			defer func(serviceID string) {
				req, _ := http.NewRequest(
					http.MethodDelete,
					fmt.Sprintf("%s/services/%s", baseURL, serviceID),
					nil,
				)
				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					log.Printf("清理服务失败[%s]: %v", serviceID, err)
					return
				}
				resp.Body.Close()
			}(id)
		}

		// 收集失败的错误
		close(failed)
		var errors []error
		for err := range failed {
			errors = append(errors, err)
		}

		// 检查结果
		t.Logf("成功注册: %d, 失败: %d", len(registeredIDs), len(errors))
		for _, err := range errors {
			t.Logf("并发注册错误: %v", err)
		}

		assert.Equal(t, concurrency, len(registeredIDs), "并发注册成功数量与预期不符")
		assert.Empty(t, errors, "并发注册出现错误")
	})
}
