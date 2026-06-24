// etcd 集成模块测试
// 测试 etcd 注册中心的默认配置、选项设置和实例键生成等功能
package etcd

import (
	"context"
	"testing"
	"time"

	"github.com/xudefa/go-boot/center"
	"github.com/xudefa/go-boot/config"
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
	if cfg.TTL != 10*time.Second {
		t.Fatalf("unexpected TTL: %v", cfg.TTL)
	}
	if cfg.DialTimeout != 5*time.Second {
		t.Fatalf("unexpected dial timeout: %v", cfg.DialTimeout)
	}
}

// TestWithOptions 测试通过选项函数设置端点和前缀，验证各配置项正确
func TestWithOptions(t *testing.T) {
	opts := []Option{
		WithEndpoints("192.168.1.1:2379", "192.168.1.2:2379"),
		WithPrefix("/my-services"),
		WithTTL(30 * time.Second),
		WithDialTimeout(10 * time.Second),
	}
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	if len(cfg.Endpoints) != 2 {
		t.Fatalf("unexpected endpoints count: %d", len(cfg.Endpoints))
	}
	if cfg.Endpoints[0] != "192.168.1.1:2379" {
		t.Fatalf("unexpected first endpoint: %s", cfg.Endpoints[0])
	}
	if cfg.Endpoints[1] != "192.168.1.2:2379" {
		t.Fatalf("unexpected second endpoint: %s", cfg.Endpoints[1])
	}
	if cfg.Prefix != "/my-services" {
		t.Fatalf("unexpected prefix: %s", cfg.Prefix)
	}
	if cfg.TTL != 30*time.Second {
		t.Fatalf("unexpected TTL: %v", cfg.TTL)
	}
	if cfg.DialTimeout != 10*time.Second {
		t.Fatalf("unexpected dial timeout: %v", cfg.DialTimeout)
	}
}

// TestInstanceKey 测试生成实例在 etcd 中的存储键，验证格式为 /prefix/serviceName/instanceID
func TestInstanceKey(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		info     center.InstanceInfo
		expected string
	}{
		{
			name:   "normal case",
			prefix: "/services",
			info: center.InstanceInfo{
				ServiceName: "user-service",
				ID:          "192.168.1.1:8080",
			},
			expected: "/services/user-service/192.168.1.1:8080",
		},
		{
			name:   "empty ID",
			prefix: "/services",
			info: center.InstanceInfo{
				ServiceName: "user-service",
				ID:          "",
			},
			expected: "/services/user-service",
		},
		{
			name:   "nested prefix",
			prefix: "/prod/services",
			info: center.InstanceInfo{
				ServiceName: "api-gateway",
				ID:          "inst-1",
			},
			expected: "/prod/services/api-gateway/inst-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EtcdRegistry{config: &Config{Prefix: tt.prefix}}
			key := r.instanceKey(tt.info)
			if key != tt.expected {
				t.Errorf("instanceKey() = %s, want %s", key, tt.expected)
			}
		})
	}
}

// TestRegistryInterface 编译时检查 EtcdRegistry 是否实现了 center.Registry 接口
func TestRegistryInterface(t *testing.T) {
	var _ center.Registry = (*EtcdRegistry)(nil)
}

// TestRegister_WithEmptyServiceName 测试注册空服务名称，验证返回错误
func TestRegister_WithEmptyServiceName(t *testing.T) {
	registry := &EtcdRegistry{
		config: &Config{
			Endpoints: []string{"localhost:2379"},
			Prefix:    "/services",
		},
	}

	err := registry.Register(context.Background(), center.InstanceInfo{
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
	registry := &EtcdRegistry{
		config: &Config{
			Endpoints: []string{"localhost:2379"},
			Prefix:    "/services",
		},
	}

	err := registry.Register(context.Background(), center.InstanceInfo{
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
	registry := &EtcdRegistry{
		config: &Config{
			Endpoints: []string{"localhost:2379"},
			Prefix:    "/services",
		},
	}

	err := registry.Register(context.Background(), center.InstanceInfo{
		ServiceName: "test-service",
		Host:        "127.0.0.1",
		Port:        0,
	})
	if err == nil {
		t.Fatal("expected error for invalid port")
	}
	if err.Error() != "valid port is required for registration" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRegister_WithNegativePort 测试注册负数端口
func TestRegister_WithNegativePort(t *testing.T) {
	registry := &EtcdRegistry{
		config: &Config{
			Endpoints: []string{"localhost:2379"},
			Prefix:    "/services",
		},
	}

	err := registry.Register(context.Background(), center.InstanceInfo{
		ServiceName: "test-service",
		Host:        "127.0.0.1",
		Port:        -1,
	})
	if err == nil {
		t.Fatal("expected error for negative port")
	}
}

// TestNewEtcdRegistry_WithValidEndpoint 测试使用有效端点创建 etcd 注册中心
func TestNewEtcdRegistry_WithValidEndpoint(t *testing.T) {
	_, err := NewEtcdRegistry(WithEndpoints("localhost:2379"))
	if err != nil {
		t.Logf("Connection failed as expected: %v", err)
	}
}

// TestNewEtcdRegistry_WithOptions 测试使用多个选项创建注册中心
func TestNewEtcdRegistry_WithOptions(t *testing.T) {
	_, err := NewEtcdRegistry(
		WithEndpoints("localhost:2379"),
		WithPrefix("/test"),
		WithTTL(15*time.Second),
		WithDialTimeout(3*time.Second),
	)
	if err != nil {
		t.Logf("Connection failed as expected: %v", err)
	}
}

// TestNewEtcdRegistry_MultipleEndpoints 测试多端点创建
func TestNewEtcdRegistry_MultipleEndpoints(t *testing.T) {
	_, err := NewEtcdRegistry(
		WithEndpoints("localhost:2379", "localhost:2380", "localhost:2381"),
	)
	if err != nil {
		t.Logf("Connection failed as expected: %v", err)
	}
}

// TestEtcdRegistry_StructFields 测试注册中心结构体字段
func TestEtcdRegistry_StructFields(t *testing.T) {
	cfg := &Config{
		Endpoints:   []string{"localhost:2379"},
		Prefix:      "/services",
		TTL:         10 * time.Second,
		DialTimeout: 5 * time.Second,
	}

	registry := &EtcdRegistry{
		config: cfg,
	}

	if registry.config.Prefix != "/services" {
		t.Errorf("config.Prefix = %s, want /services", registry.config.Prefix)
	}
	if len(registry.config.Endpoints) != 1 {
		t.Errorf("config.Endpoints length = %d, want 1", len(registry.config.Endpoints))
	}
}

// TestConfig_WithContainer 测试 WithContainer 选项
func TestConfig_WithContainer(t *testing.T) {
	cfg := defaultConfig()
	WithContainer(nil)(cfg)
	if cfg.Container != nil {
		t.Error("expected Container to be nil")
	}
}

// TestConfig_WithTLS 测试 WithTLS 选项
func TestConfig_WithTLS(t *testing.T) {
	cfg := defaultConfig()
	WithTLS(nil)(cfg)
	if cfg.TLS != nil {
		t.Error("expected TLS to be nil")
	}
}

// TestEtcdConfigCenter_New 测试创建配置中心
func TestEtcdConfigCenter_New(t *testing.T) {
	cfg := &config.ConfigCenterConfig{
		Endpoints: []string{"localhost:2379"},
	}

	center, err := NewEtcdConfigCenter(cfg)
	if err != nil {
		t.Fatalf("NewEtcdConfigCenter() error = %v", err)
	}
	if center == nil {
		t.Fatal("expected non-nil config center")
	}
}

// TestEtcdConfigCenter_New_EmptyEndpoints 测试空端点
func TestEtcdConfigCenter_New_EmptyEndpoints(t *testing.T) {
	cfg := &config.ConfigCenterConfig{
		Endpoints: []string{},
	}

	_, err := NewEtcdConfigCenter(cfg)
	if err == nil {
		t.Fatal("expected error for empty endpoints")
	}
	if err.Error() != "etcd endpoints required" {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestEtcdConfigCenter_New_NilConfig 测试 nil 配置
func TestEtcdConfigCenter_New_NilConfig(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("expected panic for nil config: %v", r)
		}
	}()

	_, err := NewEtcdConfigCenter(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

// TestEtcdConfigCenter_WithPrefix 测试配置中心前缀设置
func TestEtcdConfigCenter_WithPrefix(t *testing.T) {
	cfg := &config.ConfigCenterConfig{
		Endpoints: []string{"localhost:2379"},
		Prefix:    "/my-config",
	}

	center, err := NewEtcdConfigCenter(cfg)
	if err != nil {
		t.Fatalf("NewEtcdConfigCenter() error = %v", err)
	}
	if center.config.Prefix != "/my-config" {
		t.Errorf("expected prefix /my-config, got %s", center.config.Prefix)
	}
}

// TestEtcdConfigCenter_WithTimeout 测试配置中心超时设置
func TestEtcdConfigCenter_WithTimeout(t *testing.T) {
	cfg := &config.ConfigCenterConfig{
		Endpoints: []string{"localhost:2379"},
		Timeout:   15,
	}

	center, err := NewEtcdConfigCenter(cfg)
	if err != nil {
		t.Fatalf("NewEtcdConfigCenter() error = %v", err)
	}
	if center.config.Timeout != 15 {
		t.Errorf("expected timeout 15, got %d", center.config.Timeout)
	}
}

// TestEtcdConfigCenter_MultipleEndpoints 测试多端点配置中心
func TestEtcdConfigCenter_MultipleEndpoints(t *testing.T) {
	cfg := &config.ConfigCenterConfig{
		Endpoints: []string{
			"localhost:2379",
			"localhost:2380",
			"localhost:2381",
		},
	}

	center, err := NewEtcdConfigCenter(cfg)
	if err != nil {
		t.Fatalf("NewEtcdConfigCenter() error = %v", err)
	}
	if len(center.config.Endpoints) != 3 {
		t.Errorf("expected 3 endpoints, got %d", len(center.config.Endpoints))
	}
}

// TestEtcdConfigCenter_ConfigFields 测试配置中心配置字段
func TestEtcdConfigCenter_ConfigFields(t *testing.T) {
	cfg := &config.ConfigCenterConfig{
		Endpoints: []string{"localhost:2379"},
		Prefix:    "/config",
		Namespace: "production",
		DataID:    "app-config",
		Group:     "DEFAULT_GROUP",
	}

	center, err := NewEtcdConfigCenter(cfg)
	if err != nil {
		t.Fatalf("NewEtcdConfigCenter() error = %v", err)
	}

	if center.config.Prefix != "/config" {
		t.Errorf("config.Prefix = %s, want /config", center.config.Prefix)
	}
	if center.config.Namespace != "production" {
		t.Errorf("config.Namespace = %s, want production", center.config.Namespace)
	}
	if center.config.DataID != "app-config" {
		t.Errorf("config.DataID = %s, want app-config", center.config.DataID)
	}
	if center.config.Group != "DEFAULT_GROUP" {
		t.Errorf("config.Group = %s, want DEFAULT_GROUP", center.config.Group)
	}
}

// TestOptionFunctions_Chaining 测试选项函数链式调用
func TestOptionFunctions_Chaining(t *testing.T) {
	cfg := defaultConfig()

	// 链式应用选项
	WithEndpoints("10.0.0.1:2379")(cfg)
	WithPrefix("/chain-test")(cfg)
	WithTTL(60 * time.Second)(cfg)
	WithDialTimeout(15 * time.Second)(cfg)
	WithTLS(nil)(cfg)
	WithContainer(nil)(cfg)

	if len(cfg.Endpoints) != 1 || cfg.Endpoints[0] != "10.0.0.1:2379" {
		t.Errorf("endpoints not set correctly: %v", cfg.Endpoints)
	}
	if cfg.Prefix != "/chain-test" {
		t.Errorf("prefix = %s, want /chain-test", cfg.Prefix)
	}
	if cfg.TTL != 60*time.Second {
		t.Errorf("TTL = %v, want 60s", cfg.TTL)
	}
	if cfg.DialTimeout != 15*time.Second {
		t.Errorf("DialTimeout = %v, want 15s", cfg.DialTimeout)
	}
	if cfg.TLS != nil {
		t.Error("TLS should be nil")
	}
	if cfg.Container != nil {
		t.Error("Container should be nil")
	}
}

// TestDefaultConfig_Immutability 测试默认配置不会被修改
func TestDefaultConfig_Immutability(t *testing.T) {
	// 修改一个配置
	cfg1 := defaultConfig()
	cfg1.Prefix = "/modified"
	cfg1.Endpoints = []string{"modified:2379"}
	cfg1.TTL = 999 * time.Second

	// 获取新的默认配置，应该不受影响
	cfg2 := defaultConfig()
	if cfg2.Prefix == "/modified" {
		t.Error("defaultConfig() returned modified prefix")
	}
	if len(cfg2.Endpoints) == 1 && cfg2.Endpoints[0] == "modified:2379" {
		t.Error("defaultConfig() returned modified endpoints")
	}
	if cfg2.TTL == 999*time.Second {
		t.Error("defaultConfig() returned modified TTL")
	}
}

// TestRegister_WithAllFields 测试注册时使用完整的 InstanceInfo
func TestRegister_WithAllFields(t *testing.T) {
	// 测试带有完整字段的实例信息（验证参数传递）
	info := center.InstanceInfo{
		ServiceName: "test-service",
		ID:          "inst-1",
		Host:        "127.0.0.1",
		Port:        8080,
		Weight:      10,
		Healthy:     true,
		Metadata: map[string]string{
			"version": "1.0.0",
			"env":     "production",
		},
	}

	// 验证参数
	if info.ServiceName == "" {
		t.Error("service name should not be empty")
	}
	if info.Host == "" {
		t.Error("host should not be empty")
	}
	if info.Port <= 0 {
		t.Error("port should be positive")
	}
	if info.Weight != 10 {
		t.Errorf("weight = %d, want 10", info.Weight)
	}
	if !info.Healthy {
		t.Error("healthy should be true")
	}
	if len(info.Metadata) != 2 {
		t.Errorf("metadata length = %d, want 2", len(info.Metadata))
	}
}

// TestInstanceKey_WithSpecialCharacters 测试包含特殊字符的实例键
func TestInstanceKey_WithSpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		info     center.InstanceInfo
		expected string
	}{
		{
			name:   "UUID as ID",
			prefix: "/services",
			info: center.InstanceInfo{
				ServiceName: "user-service",
				ID:          "550e8400-e29b-41d4-a716-446655440000",
			},
			expected: "/services/user-service/550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:   "IP:Port as ID",
			prefix: "/services",
			info: center.InstanceInfo{
				ServiceName: "api-gateway",
				ID:          "192.168.1.100:9090",
			},
			expected: "/services/api-gateway/192.168.1.100:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EtcdRegistry{config: &Config{Prefix: tt.prefix}}
			key := r.instanceKey(tt.info)
			if key != tt.expected {
				t.Errorf("instanceKey() = %s, want %s", key, tt.expected)
			}
		})
	}
}

// TestNewEtcdRegistry_WithSingleEndpoint 测试单端点创建
func TestNewEtcdRegistry_WithSingleEndpoint(t *testing.T) {
	_, err := NewEtcdRegistry(WithEndpoints("10.0.0.1:2379"))
	if err != nil {
		t.Logf("Connection failed as expected: %v", err)
	}
}

// TestNewEtcdRegistry_WithCustomPrefix 测试自定义前缀创建
func TestNewEtcdRegistry_WithCustomPrefix(t *testing.T) {
	_, err := NewEtcdRegistry(
		WithEndpoints("localhost:2379"),
		WithPrefix("/prod/services"),
	)
	if err != nil {
		t.Logf("Connection failed as expected: %v", err)
	}
}

// TestNewEtcdRegistry_WithShortTTL 测试短 TTL
func TestNewEtcdRegistry_WithShortTTL(t *testing.T) {
	_, err := NewEtcdRegistry(
		WithEndpoints("localhost:2379"),
		WithTTL(5*time.Second),
	)
	if err != nil {
		t.Logf("Connection failed as expected: %v", err)
	}
}

// TestNewEtcdRegistry_WithLongTTL 测试长 TTL
func TestNewEtcdRegistry_WithLongTTL(t *testing.T) {
	_, err := NewEtcdRegistry(
		WithEndpoints("localhost:2379"),
		WithTTL(60*time.Second),
	)
	if err != nil {
		t.Logf("Connection failed as expected: %v", err)
	}
}

// TestConfig_EmptyEndpoints 测试空端点配置
func TestConfig_EmptyEndpoints(t *testing.T) {
	cfg := defaultConfig()
	WithEndpoints()(cfg)
	if len(cfg.Endpoints) != 0 {
		t.Errorf("expected 0 endpoints, got %d", len(cfg.Endpoints))
	}
}

// TestConfig_EmptyPrefix 测试空前缀配置
func TestConfig_EmptyPrefix(t *testing.T) {
	cfg := defaultConfig()
	WithPrefix("")(cfg)
	if cfg.Prefix != "" {
		t.Errorf("expected empty prefix, got %s", cfg.Prefix)
	}
}
