// Package etcd 基于 etcd 提供注册中心实现。
//
// 该包将 etcd 与 go-boot 服务发现和注册中心接口集成，
// 使用 etcd Lease 机制实现健康检测，支持服务注册、发现和 Watch。
//
// 定义：
//
//   - EtcdRegistry: 注册中心实现了 center.Registry 接口
//   - Config: etcd 配置
//   - Option: 配置选项函数
//
// 快速开始:
//
//	// 创建 etcd 注册中心
//	registry, err := etcd.NewEtcdRegistry(
//	    etcd.WithEndpoints("127.0.0.1:2379"),
//	)
//
//	// 注册服务
//	registry.Register(ctx, center.InstanceInfo{
//	    ServiceName: "my-service",
//	    Host:        "127.0.0.1",
//	    Port:        8080,
//	})
package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/xudefa/go-boot/center"
)

// EtcdRegistry 基于 etcd 实现的注册中心。
// 服务实例信息存储为 JSON 格式的 value，key 为 {prefix}/{serviceName}/{instanceID}。
// 使用 etcd Lease 机制实现健康检测：注册时创建租约并通过 KeepAlive 续约。
type EtcdRegistry struct {
	client *clientv3.Client
	config *Config
}

// NewEtcdRegistry 创建 etcd 注册中心实例。
// 支持通过 Option 函数式选项配置 endpoints、前缀、TTL 等参数。
func NewEtcdRegistry(opts ...Option) (*EtcdRegistry, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.DialTimeout,
		TLS:         cfg.TLS,
	})
	if err != nil {
		return nil, fmt.Errorf("etcd: create client failed: %w", err)
	}

	return &EtcdRegistry{client: cli, config: cfg}, nil
}

// instanceKey 拼接实例在 etcd 中的完整存储路径。
// 格式：{prefix}/{serviceName}/{instanceID}
func (r *EtcdRegistry) instanceKey(info center.InstanceInfo) string {
	return path.Join(r.config.Prefix, info.ServiceName, info.ID)
}

// Register 向 etcd 注册一个服务实例。
// 步骤：
//  1. 将实例信息序列化为 JSON
//  2. 创建租约（TTL 由配置决定）
//  3. 将实例数据写入 etcd，绑定租约
//  4. 启动后台 KeepAlive 保持租约有效
//
// KeepAlive 使用的 context 从调用方 context 分离，不受其取消/超时影响，
// 调用方应通过 Deregister 来注销实例。
func (r *EtcdRegistry) Register(ctx context.Context, info center.InstanceInfo) error {
	if info.ServiceName == "" {
		return fmt.Errorf("service name is required for registration")
	}
	if info.Host == "" {
		return fmt.Errorf("host is required for registration")
	}
	if info.Port <= 0 {
		return fmt.Errorf("valid port is required for registration")
	}

	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("etcd: marshal instance failed: %w", err)
	}

	lease, err := r.client.Grant(ctx, int64(r.config.TTL.Seconds()))
	if err != nil {
		return fmt.Errorf("etcd: grant lease failed: %w", err)
	}

	_, err = r.client.Put(ctx, r.instanceKey(info), string(data),
		clientv3.WithLease(lease.ID),
	)
	if err != nil {
		return fmt.Errorf("etcd: put instance failed: %w", err)
	}

	// 使用 WithoutCancel 分离 KeepAlive 的 context，
	// 避免调用方的 context 超时或取消导致租约停止续约。
	keepAliveCtx := context.WithoutCancel(ctx)
	ch, err := r.client.KeepAlive(keepAliveCtx, lease.ID)
	if err != nil {
		return fmt.Errorf("etcd: keepalive failed: %w", err)
	}

	// 启动后台协程处理租约续期，并监控错误
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// 处理可能的panic
				_ = r
			}
		}()

		for {
			select {
			case _, ok := <-ch:
				if !ok {
					// 通道关闭，停止续期
					return
				}
				// 继续接收心跳
			case <-keepAliveCtx.Done():
				// 上下文被取消，停止续期
				return
			}
		}
	}()

	return nil
}

// Deregister 从 etcd 中删除指定服务实例。
// 实例被删除后，租约将不再被续约，等待 TTL 过期后自动清理。
func (r *EtcdRegistry) Deregister(ctx context.Context, info center.InstanceInfo) error {
	_, err := r.client.Delete(ctx, r.instanceKey(info))
	return err
}

// Discover 发现指定服务的所有在线实例。
// 通过前缀查询 {prefix}/{serviceName} 下的所有 key，反序列化得到实例列表。
func (r *EtcdRegistry) Discover(ctx context.Context, serviceName string) ([]center.InstanceInfo, error) {
	key := path.Join(r.config.Prefix, serviceName)
	resp, err := r.client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("etcd: discover failed: %w", err)
	}

	instances := make([]center.InstanceInfo, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var info center.InstanceInfo
		if err := json.Unmarshal(kv.Value, &info); err != nil {
			continue
		}
		instances = append(instances, info)
	}
	return instances, nil
}

// Watch 监听指定服务的实例变化。
// 底层使用 etcd Watch 监听 {prefix}/{serviceName} 前缀下的变化。
// 每次变化时调用 Discover 获取最新实例列表并发送到返回的 channel。
func (r *EtcdRegistry) Watch(ctx context.Context, serviceName string) (<-chan []center.InstanceInfo, error) {
	key := path.Join(r.config.Prefix, serviceName)
	ch := make(chan []center.InstanceInfo, 16)

	wch := r.client.Watch(ctx, key, clientv3.WithPrefix())
	go func() {
		defer close(ch)
		for range wch {
			instances, err := r.Discover(ctx, serviceName)
			if err != nil {
				continue
			}
			select {
			case ch <- instances:
			default:
			}
		}
	}()
	return ch, nil
}
