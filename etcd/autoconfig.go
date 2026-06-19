// Package etcd 提供 etcd 注册中心的自动配置。
//
// 当 etcd.enabled=true 时自动启用，从 Environment 中读取 etcd.endpoints、etcd.prefix 等配置项，
// 创建并注册 EtcdRegistry Bean 到 IoC 容器中（Bean ID: etcdRegistry），实现 center.Registry 接口。
package etcd

import (
	"context"
	"fmt"
	"strings"
	"time"

	etcdcore "github.com/xudefa/go-boot-etcd"

	"github.com/xudefa/go-boot/boot"
	"github.com/xudefa/go-boot/condition"
	"github.com/xudefa/go-boot/config"
	"github.com/xudefa/go-boot/constants"
	"github.com/xudefa/go-boot/core"
)

// init 注册 etcd 自动配置和配置中心工厂
func init() {
	boot.RegisterAutoConfig(&EtcdAutoConfiguration{},
		condition.OnProperty("etcd.enabled", "true"),
	)

	boot.RegisterConfigCenterFactory("etcd", etcdConfigCenterFactory)
}

// etcdConfigCenterFactory Etcd 配置中心工厂函数
func etcdConfigCenterFactory(ctx context.Context, cfg *config.ConfigCenterConfig) (config.ConfigCenter, error) {
	return etcdcore.NewEtcdConfigCenter(cfg)
}

// EtcdAutoConfiguration etcd 注册中心的自动配置。
// 从环境变量读取配置并创建 EtcdRegistry，注册到 IoC 容器中。
// 启用条件：etcd.enabled=true
type EtcdAutoConfiguration struct{}

// Configure 执行自动配置逻辑。
// 从 Environment 中读取 etcd.endpoints、etcd.prefix 等配置项。
func (e *EtcdAutoConfiguration) Configure(ctx boot.ApplicationContext) error {
	env := ctx.Environment()

	// 注册配置中心（如果启用）
	if env.GetBool("etcd.config-center.enabled", false) {
		cfg := &config.ConfigCenterConfig{
			Endpoints: strings.Split(env.GetString("etcd.endpoints", "localhost:2379"), ","),
			Timeout:   5 * time.Second,
			DataID:    env.GetString("etcd.config-center.data-id", "app-config"),
			Group:     env.GetString("etcd.config-center.group", "DEFAULT_GROUP"),
			Prefix:    env.GetString("etcd.config-center.prefix", "/config"),
		}
		center, err := etcdcore.NewEtcdConfigCenter(cfg)
		if err != nil {
			return fmt.Errorf("create etcd config center failed: %w", err)
		}
		if err := ctx.Register(constants.ConfigCenterBeanID, core.Bean(center), core.Singleton()); err != nil {
			return err
		}
	}

	endpointsStr := env.GetString("etcd.endpoints", "localhost:2379")
	endpoints := strings.Split(endpointsStr, ",")

	reg, err := etcdcore.NewEtcdRegistry(
		etcdcore.WithEndpoints(endpoints...),
		etcdcore.WithPrefix(env.GetString("etcd.prefix", "/services")),
		etcdcore.WithContainer(ctx.Container()),
	)
	if err != nil {
		return err
	}

	if err := ctx.Register("etcdRegistry",
		core.Bean(reg),
		core.Singleton(),
	); err != nil {
		return err
	}

	return nil
}
