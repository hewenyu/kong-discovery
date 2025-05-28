package etcdclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

// WatchEvent 定义监听事件类型
type WatchEvent struct {
	EventType  string           // 事件类型: "create", "update", "delete"
	Key        string           // 发生变化的key
	Value      string           // 变化后的值 (对于delete事件，此字段为空)
	PrevValue  string           // 变化前的值 (对于create事件，此字段为空)
	ServiceObj *ServiceInstance // 解析后的服务对象 (如果是服务相关的事件)
}

// WatchCallback 定义监听回调函数类型
type WatchCallback func(event WatchEvent)

// StartWatch 开始监听指定前缀的key变化
func (e *EtcdClient) StartWatch(ctx context.Context, prefix string, callback WatchCallback) error {
	if e.client == nil {
		return fmt.Errorf("etcd客户端未连接")
	}

	e.logger.Info("开始监听etcd变化", zap.String("prefix", prefix))

	// 获取当前所有键值，用于初始化
	getResp, err := e.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		e.logger.Error("获取初始键值失败", zap.String("prefix", prefix), zap.Error(err))
		return fmt.Errorf("获取初始键值失败: %w", err)
	}

	// 存储初始键值对，用于比较
	initialKVs := make(map[string]string)
	for _, kv := range getResp.Kvs {
		initialKVs[string(kv.Key)] = string(kv.Value)
	}

	// 从最新的revision开始监听
	watchStartRevision := getResp.Header.Revision + 1

	// 启动监听
	watchChan := e.client.Watch(ctx, prefix, clientv3.WithPrefix(), clientv3.WithRev(watchStartRevision), clientv3.WithPrevKV())

	// 在后台协程中处理监听事件
	go func() {
		for watchResp := range watchChan {
			if watchResp.Canceled {
				e.logger.Warn("etcd监听被取消", zap.String("prefix", prefix), zap.Error(watchResp.Err()))
				// 尝试重新连接
				time.Sleep(1 * time.Second)
				newWatchChan := e.client.Watch(ctx, prefix, clientv3.WithPrefix(), clientv3.WithPrevKV())
				watchChan = newWatchChan
				continue
			}

			for _, event := range watchResp.Events {
				key := string(event.Kv.Key)
				value := string(event.Kv.Value)
				var prevValue string
				var eventType string

				switch event.Type {
				case clientv3.EventTypePut:
					// 判断是create还是update
					if event.IsCreate() {
						eventType = "create"
					} else {
						eventType = "update"
						if event.PrevKv != nil {
							prevValue = string(event.PrevKv.Value)
						}
					}
				case clientv3.EventTypeDelete:
					eventType = "delete"
					if event.PrevKv != nil {
						prevValue = string(event.PrevKv.Value)
					}
				}

				// 构造事件对象
				watchEvent := WatchEvent{
					EventType: eventType,
					Key:       key,
					Value:     value,
					PrevValue: prevValue,
				}

				// 如果是服务相关的key，尝试解析服务对象
				if isServiceKey(key) && eventType != "delete" && value != "" {
					serviceObj, err := parseServiceFromJSON(value)
					if err == nil {
						watchEvent.ServiceObj = serviceObj
					} else {
						e.logger.Warn("解析服务对象失败",
							zap.String("key", key),
							zap.Error(err))
					}
				} else if isServiceKey(key) && eventType == "delete" && prevValue != "" {
					// 对于删除事件，尝试从prevValue解析服务对象，这样可以在回调中获取更多信息
					serviceObj, err := parseServiceFromJSON(prevValue)
					if err == nil {
						watchEvent.ServiceObj = serviceObj
					}
				}

				// 调用回调函数
				callback(watchEvent)

				e.logger.Info("检测到etcd变化",
					zap.String("type", eventType),
					zap.String("key", key))
			}
		}
	}()

	return nil
}

// isServiceKey 判断一个key是否是服务相关的key
func isServiceKey(key string) bool {
	// 判断key是否以服务前缀开头
	return len(key) > 10 && key[:10] == "/services/"
}

// parseServiceFromJSON 从JSON字符串解析服务实例
func parseServiceFromJSON(jsonStr string) (*ServiceInstance, error) {
	if jsonStr == "" {
		return nil, fmt.Errorf("空的JSON字符串")
	}

	var service ServiceInstance
	err := json.Unmarshal([]byte(jsonStr), &service)
	if err != nil {
		return nil, err
	}

	return &service, nil
}
