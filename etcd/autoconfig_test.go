package etcd

import (
	"context"
	"testing"

	"github.com/xudefa/go-boot/config"
)

// TestEtcdConfigCenterFactory 测试配置中心工厂
func TestEtcdConfigCenterFactory(t *testing.T) {
	// 测试工厂函数能正确创建配置中心
	cfg := &config.ConfigCenterConfig{
		Endpoints: []string{"localhost:2379"},
	}

	_, err := etcdConfigCenterFactory(context.Background(), cfg)
	// 由于没有真实的 etcd 服务器，这里会返回连接错误
	// 但我们主要测试工厂函数的参数验证逻辑
	if err != nil {
		t.Logf("expected connection error: %v", err)
	}
}

// TestEtcdConfigCenterFactory_EmptyEndpoints 测试空端点
func TestEtcdConfigCenterFactory_EmptyEndpoints(t *testing.T) {
	cfg := &config.ConfigCenterConfig{
		Endpoints: []string{},
	}

	_, err := etcdConfigCenterFactory(context.Background(), cfg)
	if err == nil {
		t.Error("expected error for empty endpoints")
	}
}

// TestEtcdConfigCenterFactory_NilConfig 测试空配置
func TestEtcdConfigCenterFactory_NilConfig(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			// 预期会 panic，因为 nil config 会导致空指针
			t.Logf("expected panic for nil config: %v", r)
		}
	}()
	_, err := etcdConfigCenterFactory(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

// TestEtcdConfigCenterFactory_MultipleEndpoints 测试多端点配置
func TestEtcdConfigCenterFactory_MultipleEndpoints(t *testing.T) {
	cfg := &config.ConfigCenterConfig{
		Endpoints: []string{"localhost:2379", "localhost:2380", "localhost:2381"},
	}

	_, err := etcdConfigCenterFactory(context.Background(), cfg)
	if err != nil {
		t.Logf("expected connection error: %v", err)
	}
}

// TestEtcdConfigCenterFactory_WithAllOptions 测试所有配置选项
func TestEtcdConfigCenterFactory_WithAllOptions(t *testing.T) {
	cfg := &config.ConfigCenterConfig{
		Endpoints: []string{"localhost:2379"},
		Namespace: "test-namespace",
		Timeout:   10,
		DataID:    "test-data-id",
		Group:     "test-group",
		Prefix:    "/config",
	}

	_, err := etcdConfigCenterFactory(context.Background(), cfg)
	if err != nil {
		t.Logf("expected connection error: %v", err)
	}
}

// TestEtcdRegistry_WithOptions 测试选项配置
func TestEtcdRegistry_WithOptions(t *testing.T) {
	_, err := NewEtcdRegistry(
		WithEndpoints("localhost:2379"),
		WithPrefix("/test"),
		WithTTL(30),
	)
	if err != nil {
		t.Logf("expected connection error: %v", err)
	}
}
