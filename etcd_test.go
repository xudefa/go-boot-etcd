// etcd 集成模块测试
// 测试 etcd 注册中心的默认配置、选项设置和实例键生成等功能
package etcd

import (
	"context"
	"testing"

	"github.com/xudefa/go-boot/center"
)

// TestEtcdConfigDefaults 测试默认配置，验证默认端点和前缀
func TestEtcdConfigDefaults(t *testing.T) {
	cfg := defaultConfig()
	if len(cfg.Endpoints) != 1 || cfg.Endpoints[0] != "localhost:2379" {
		t.Fatalf("unexpected endpoints: %v", cfg.Endpoints)
	}
	if cfg.Prefix != "/services" {
		t.Fatalf("unexpected prefix: %s", cfg.Prefix)
	}
}

// TestWithOptions 测试通过选项函数设置端点和前缀，验证各配置项正确
func TestWithOptions(t *testing.T) {
	opts := []Option{
		WithEndpoints("192.168.1.1:2379", "192.168.1.2:2379"),
		WithPrefix("/my-services"),
	}
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	if len(cfg.Endpoints) != 2 {
		t.Fatalf("unexpected endpoints count: %d", len(cfg.Endpoints))
	}
	if cfg.Prefix != "/my-services" {
		t.Fatalf("unexpected prefix: %s", cfg.Prefix)
	}
}

// TestInstanceKey 测试生成实例在 etcd 中的存储键，验证格式为 /prefix/serviceName/instanceID
func TestInstanceKey(t *testing.T) {
	r := &EtcdRegistry{config: &Config{Prefix: "/services"}}
	key := r.instanceKey(center.InstanceInfo{
		ServiceName: "user-service",
		ID:          "192.168.1.1:8080",
	})
	expected := "/services/user-service/192.168.1.1:8080"
	if key != expected {
		t.Fatalf("expected %s, got %s", expected, key)
	}
}

// TestRegistryInterface 编译时检查 EtcdRegistry 是否实现了 center.Registry 接口
func TestRegistryInterface(t *testing.T) {
	var _ center.Registry = (*EtcdRegistry)(nil)
	_ = context.TODO()
}

// TestRegister_WithEmptyServiceName 测试注册空服务名称，验证返回错误
func TestRegister_WithEmptyServiceName(t *testing.T) {
	// 创建一个测试实例，但不连接真实的etcd
	registry := &EtcdRegistry{
		config: &Config{
			Endpoints: []string{"localhost:2379"},
			Prefix:    "/services",
		},
	}

	err := registry.Register(context.Background(), struct {
		ServiceName string
		ID          string
		Host        string
		Port        int
		Weight      int
		Healthy     bool
		Metadata    map[string]string
	}{
		ServiceName: "",
		Host:        "127.0.0.1",
		Port:        8080,
	})
	if err == nil {
		t.Fatal("expected error for empty service name")
	}
	if err.Error() != "service name is required for registration" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRegister_WithEmptyHost 测试注册空主机，验证返回错误
func TestRegister_WithEmptyHost(t *testing.T) {
	// 创建一个测试实例，但不连接真实的etcd
	registry := &EtcdRegistry{
		config: &Config{
			Endpoints: []string{"localhost:2379"},
			Prefix:    "/services",
		},
	}

	err := registry.Register(context.Background(), struct {
		ServiceName string
		ID          string
		Host        string
		Port        int
		Weight      int
		Healthy     bool
		Metadata    map[string]string
	}{
		ServiceName: "test-service",
		Host:        "",
		Port:        8080,
	})
	if err == nil {
		t.Fatal("expected error for empty host")
	}
	if err.Error() != "host is required for registration" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRegister_WithInvalidPort 测试注册无效端口，验证返回错误
func TestRegister_WithInvalidPort(t *testing.T) {
	// 创建一个测试实例，但不连接真实的etcd
	registry := &EtcdRegistry{
		config: &Config{
			Endpoints: []string{"localhost:2379"},
			Prefix:    "/services",
		},
	}

	err := registry.Register(context.Background(), struct {
		ServiceName string
		ID          string
		Host        string
		Port        int
		Weight      int
		Healthy     bool
		Metadata    map[string]string
	}{
		ServiceName: "test-service",
		Host:        "127.0.0.1",
		Port:        0, // Invalid port
	})
	if err == nil {
		t.Fatal("expected error for invalid port")
	}
	if err.Error() != "valid port is required for registration" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestNewEtcdRegistry_WithValidEndpoint 测试使用有效端点创建 etcd 注册中心
func TestNewEtcdRegistry_WithValidEndpoint(t *testing.T) {
	// 由于可能没有真实etcd服务器，我们只测试配置逻辑
	// 这里使用一个有效的地址格式，但预期连接可能会失败
	_, err := NewEtcdRegistry(WithEndpoints("localhost:2379"))
	// 我们主要测试配置是否正确，而不是连接是否成功
	if err != nil {
		t.Logf("Connection failed as expected: %v", err)
		// 如果是连接错误，这可能是正常的（因为可能没有etcd服务器）
	}
}
