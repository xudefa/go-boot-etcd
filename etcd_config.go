package etcd

import (
	"crypto/tls"
	"time"

	"github.com/xudefa/go-boot/core"
)

// Config etcd 注册中心配置。
type Config struct {
	Endpoints   []string
	Prefix      string
	TTL         time.Duration
	DialTimeout time.Duration
	TLS         *tls.Config
	Container   core.Container
}

// Option etcd 配置的函数式选项。
type Option func(*Config)

// WithEndpoints 设置 etcd 集群地址列表。
func WithEndpoints(endpoints ...string) Option {
	return func(c *Config) { c.Endpoints = endpoints }
}

// WithPrefix 设置服务实例在 etcd 中的存储前缀。
func WithPrefix(prefix string) Option {
	return func(c *Config) { c.Prefix = prefix }
}

// WithTTL 设置租约 TTL，即实例的心跳间隔。
func WithTTL(ttl time.Duration) Option {
	return func(c *Config) { c.TTL = ttl }
}

// WithDialTimeout 设置 etcd 客户端拨号超时时间。
func WithDialTimeout(timeout time.Duration) Option {
	return func(c *Config) { c.DialTimeout = timeout }
}

// WithTLS 设置 etcd 客户端 TLS 配置。
func WithTLS(tlsConfig *tls.Config) Option {
	return func(c *Config) { c.TLS = tlsConfig }
}

// WithContainer 设置 IoC 容器实例。
func WithContainer(ctn core.Container) Option {
	return func(c *Config) { c.Container = ctn }
}

func defaultConfig() *Config {
	return &Config{
		Endpoints:   []string{"localhost:2379"},
		Prefix:      "/services",
		TTL:         10 * time.Second,
		DialTimeout: 5 * time.Second,
	}
}
