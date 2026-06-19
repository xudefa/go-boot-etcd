// Package etcd 提供 etcd 配置中心实现
package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xudefa/go-boot/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdConfigCenter struct {
	client *clientv3.Client
	config config.ConfigCenterConfig
}

func NewEtcdConfigCenter(cfg *config.ConfigCenterConfig) (*EtcdConfigCenter, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("etcd endpoints required")
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("create etcd client failed: %w", err)
	}

	return &EtcdConfigCenter{
		client: client,
		config: *cfg,
	}, nil
}

// Load 加载所有配置数据
func (e *EtcdConfigCenter) Load() (config.ConfigData, error) {
	ctx := context.Background()
	prefix := e.config.Prefix + "/"

	resp, err := e.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	result := make(config.ConfigData)
	for _, kv := range resp.Kvs {
		relativeKey := strings.TrimPrefix(string(kv.Key), prefix)
		if relativeKey != "" {
			var value any
			if err := json.Unmarshal(kv.Value, &value); err == nil {
				result[relativeKey] = value
			} else {
				result[relativeKey] = string(kv.Value)
			}
		}
	}

	return result, nil
}

// Watch 监控配置变更
func (e *EtcdConfigCenter) Watch(key string, callback func(config.ConfigData)) error {
	ctx := context.Background()
	fullKey := e.config.Prefix + "/" + key

	watchChan := e.client.Watch(ctx, fullKey)

	go func() {
		for watchResp := range watchChan {
			for _, event := range watchResp.Events {
				if event.Type == clientv3.EventTypePut {
					var data config.ConfigData
					if err := json.Unmarshal(event.Kv.Value, &data); err == nil {
						callback(data)
					}
				}
			}
		}
	}()

	return nil
}

// Close 关闭客户端
func (e *EtcdConfigCenter) Close() error {
	return e.client.Close()
}
