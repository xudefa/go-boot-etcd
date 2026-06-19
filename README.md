# go-boot-etcd

[![Go Version](https://img.shields.io/github/go-mod/go-version/xudefa/go-boot-etcd)](https://go.dev/) [![License](https://img.shields.io/github/license/xudefa/go-boot-etcd)](./LICENSE) [![Build Status](https://img.shields.io/github/actions/workflow/status/xudefa/go-boot-etcd/test.yml?branch=master)](https://github.com/xudefa/go-boot-etcd/actions) [![Go Reference](https://pkg.go.dev/badge/github.com/xudefa/go-boot-etcd.svg)](https://pkg.go.dev/github.com/xudefa/go-boot-etcd) [![Go Report Card](https://goreportcard.com/badge/github.com/xudefa/go-boot-etcd)](https://goreportcard.com/report/github.com/xudefa/go-boot-etcd)

基于 [go-boot](https://github.com/xudefa/go-boot) 的 etcd 注册中心与配置中心集成模块。将 etcd 无缝集成到 go-boot 的 IoC 容器和自动配置体系中，提供服务注册、服务发现、配置加载和配置监听能力。

> 设计理念：遵循 go-boot 的开发规范，将 etcd 作为 `center.Registry` 和 `config.ConfigCenter` 接口的实现，通过自动配置实现零代码启动服务注册与配置管理。

## 整体架构

```
┌───────────────────────────────────────────────────────────────────────┐
│                    go-boot ApplicationContext                         │
│  ┌───────────┐ ┌──────────────┐ ┌───────────┐ ┌───────────┐           │
│  │ Container │ │  Environment │ │ Lifecycle │ │ EventBus  │           │
│  └───────────┘ └──────────────┘ └───────────┘ └───────────┘           │
│                       ┌─────────────────────┐                         │
│                       │ AutoConfig Registry │                         │
│                       └─────────────────────┘                         │
└───────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
                    ┌───────────────────────────────┐
                    │    go-boot-etcd Starter       │
                    │  ┌─────────────────────────┐  │
                    │  │ EtcdRegistry Bean       │  │
                    │  │ (center.Registry)       │  │
                    │  │ EtcdConfigCenter Bean   │  │
                    │  │ (config.ConfigCenter)   │  │
                    │  │ Lease & KeepAlive       │  │
                    │  └─────────────────────────┘  │
                    └───────────────────────────────┘
```

## 目录

- [快速开始](#快速开始)
- [功能特性](#功能特性)
- [服务注册与发现](#服务注册与发现)
- [配置中心](#配置中心)
- [配置选项](#配置选项)
- [项目结构](#项目结构)
- [开发指南](#开发指南)
- [贡献](#贡献)
- [许可证](#许可证)

## 快速开始

### 安装

```bash
# 安装核心框架
go get github.com/xudefa/go-boot

# 安装 etcd 集成模块
go get github.com/xudefa/go-boot-etcd
```

### 最小示例

```go
package main

import (
    "github.com/xudefa/go-boot/boot"
    "github.com/xudefa/go-boot/center"
)

func main() {
    app, err := boot.NewApplication(
        boot.WithAppName("my-service"),
        boot.WithVersion("1.0.0"),
        boot.WithProperty("etcd.enabled", "true"),
        boot.WithProperty("etcd.endpoints", "127.0.0.1:2379"),
    )
    if err != nil {
        panic(err)
    }
    defer app.Stop()

    // 获取注册中心（自动注入）
    registry := app.Container().Get("etcdRegistry").(center.Registry)

    // 注册服务
    registry.Register(context.Background(), center.InstanceInfo{
        ServiceName: "my-service",
        Host:        "127.0.0.1",
        Port:        8080,
    })

    // 发现服务
    instances, _ := registry.Discover(context.Background(), "my-service")
    for _, inst := range instances {
        fmt.Printf("发现实例: %s:%d\n", inst.Host, inst.Port)
    }

    app.Start()
    app.WaitForSignal()
}
```

## 功能特性

| 特性 | 说明 |
|------|------|
| 注册中心 | 实现 go-boot `center.Registry` 接口 |
| 配置中心 | 实现 go-boot `config.ConfigCenter` 接口 |
| 自动配置 | 通过 `etcd.enabled=true` 自动启用 |
| 租约机制 | 使用 etcd Lease + KeepAlive 实现健康检测 |
| 服务发现 | 支持前缀查询和 Watch 监听实例变化 |
| 配置监听 | 支持配置变更的实时监听 |
| 函数式选项 | 灵活的连接配置（Endpoints、Prefix、TTL 等） |

## 服务注册与发现

### 创建注册中心

```go
registry, err := etcd.NewEtcdRegistry(
    etcd.WithEndpoints("127.0.0.1:2379"),
    etcd.WithPrefix("/services"),
    etcd.WithTTL(30*time.Second),
)
```

### 注册服务

```go
registry.Register(ctx, center.InstanceInfo{
    ServiceName: "user-service",
    ID:          "instance-001",
    Host:        "127.0.0.1",
    Port:        8080,
    Weight:      10,
    Healthy:     true,
    Metadata:    map[string]string{"version": "1.0.0"},
})
```

### 发现服务

```go
instances, err := registry.Discover(ctx, "user-service")
```

### 监听服务变化

```go
ch, err := registry.Watch(ctx, "user-service")
for instances := range ch {
    fmt.Printf("当前在线实例: %d\n", len(instances))
}
```

## 配置中心

### 启用配置中心

```yaml
# application.yml
etcd:
  enabled: true
  endpoints: "127.0.0.1:2379"
  config-center:
    enabled: true
    data-id: "app-config"
    group: "DEFAULT_GROUP"
    prefix: "/config"
```

### 加载配置

```go
configCenter := app.Container().Get("configCenter").(config.ConfigCenter)
data, err := configCenter.Load()
```

### 监听配置变更

```go
configCenter.Watch("app-config", func(data config.ConfigData) {
    fmt.Printf("配置已更新: %v\n", data)
})
```

## 配置选项

通过 `boot.WithProperty()` 或配置文件设置：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `etcd.enabled` | `false` | 是否启用 etcd 注册中心 |
| `etcd.endpoints` | `localhost:2379` | etcd 服务端点（逗号分隔） |
| `etcd.prefix` | `/services` | 服务注册前缀 |
| `etcd.ttl` | `30s` | 租约 TTL |
| `etcd.dial-timeout` | `5s` | 连接超时 |
| `etcd.config-center.enabled` | `false` | 是否启用配置中心 |
| `etcd.config-center.data-id` | `app-config` | 配置 DataID |
| `etcd.config-center.group` | `DEFAULT_GROUP` | 配置分组 |
| `etcd.config-center.prefix` | `/config` | 配置前缀 |

### 示例配置

```yaml
# application.yml
etcd:
  enabled: true
  endpoints: "127.0.0.1:2379,127.0.0.1:2380"
  prefix: "/services"
  ttl: "30s"
  dial-timeout: "5s"
  config-center:
    enabled: true
    data-id: "my-app"
    prefix: "/config/my-app"
```

## 项目结构

```
go-boot-etcd/
├── etcd.go              # EtcdRegistry 实现 center.Registry
├── etcd_config.go       # 配置选项（Config, Option）
├── config_center.go     # EtcdConfigCenter 实现 config.ConfigCenter
├── autoconfig.go        # 自动配置注册
├── etcd_test.go         # 单元测试
├── README.md
├── LICENSE
└── go.mod
```

## 开发指南

### 构建

```bash
go build ./...
```

### 测试

```bash
go test ./...
go test -cover ./...       # 带覆盖率
go test -race ./...        # 数据竞争检测
```

### 代码规范

```bash
go fmt ./...
golangci-lint run
```

## 贡献

欢迎提交 Issue 和 Pull Request！详细贡献指南请参阅 [CONTRIBUTING.md](./CONTRIBUTING.md)。

## 许可证

本项目采用 MIT 许可证 — 详情请参阅 [LICENSE](./LICENSE) 文件。