# RabbitMQ 客户端模块

基于 [amqp091-go](https://github.com/rabbitmq/amqp091-go) 的 RabbitMQ 客户端配置模块，集成到 mcube IoC 容器中，支持连接自动重连、Publisher/Consumer 自动恢复、OpenTelemetry 链路追踪，以及静态凭证和 Vault KV 静态凭证两种凭证加载模式。

## 目录

- [开发环境搭建](#开发环境搭建)
- [快速开始](#快速开始)
- [配置方式](#配置方式)
  - [环境变量](#环境变量)
  - [TOML 文件](#toml-文件)
- [凭证模式](#凭证模式)
  - [静态凭证（static）](#静态凭证static)
  - [Vault KV 静态凭证（vault-secret）](#vault-kv-静态凭证vault-secret)
- [消息模式](#消息模式)
  - [队列模式（Direct）](#队列模式direct)
  - [发布订阅模式（Fanout）](#发布订阅模式fanout)
  - [主题模式（Topic）](#主题模式topic)
- [链路追踪](#链路追踪)
- [完整配置参数](#完整配置参数)

---

## 开发环境搭建

```sh
docker run -d \
  --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  -e RABBITMQ_DEFAULT_USER=guest \
  -e RABBITMQ_DEFAULT_PASS=guest \
  rabbitmq:3-management
```

通过 http://localhost:15672 访问管理界面（默认账号 `guest` / `guest`）。

---

## 快速开始

**1. 导入包（触发自动注册）**

```go
import (
    "github.com/infraboard/mcube/v2/ioc"
    _ "github.com/infraboard/mcube/v2/ioc/config/rabbitmq"
)
```

**2. 初始化**

```go
func main() {
    // 从配置文件或环境变量加载
    ioc.DevelopmentSetupWithPath("etc/application.toml")

    // 创建 Publisher
    pub, err := rabbitmq.NewPublisher()
    if err != nil {
        panic(err)
    }
    defer pub.Close()

    // 创建 Consumer
    sub, err := rabbitmq.NewConsumer()
    if err != nil {
        panic(err)
    }
    defer sub.Close()
}
```

**3. 最简 TOML 配置（`etc/application.toml`）**

```toml
[rabbitmq]
  url = "amqp://guest:guest@localhost:5672/"
```

---

## 配置方式

### 环境变量

所有配置项均可通过 `RABBITMQ_` 前缀的环境变量设置。

```bash
export RABBITMQ_URL=amqp://guest:guest@localhost:5672/
export RABBITMQ_TIMEOUT=5s
export RABBITMQ_HEART_BEAT=10s
export RABBITMQ_RECONNECT_INTERVAL=10s
export RABBITMQ_MAX_RECONNECT_ATTEMPTS=0   # 0 表示无限重连
export RABBITMQ_TRACE=true

# 凭证模式相关
export RABBITMQ_CREDENTIAL_MODE=static     # static | vault-secret
export RABBITMQ_VAULT_PATH=myapp/rabbitmq
export RABBITMQ_VAULT_URL_FIELD=url
```

### TOML 文件

```toml
[rabbitmq]
  url                    = "amqp://guest:guest@localhost:5672/"
  timeout                = "5s"
  heart_beat             = "10s"
  reconect_interval      = "10s"      # 重连间隔
  max_reconnect_attempts = 0          # 最大重连次数，0 表示无限重连
  trace                  = true       # 启用 OpenTelemetry 链路追踪

  # 凭证模式（可选，默认 static）
  credential_mode = "static"
  vault_path      = ""
  vault_url_field = "url"
```

---

## 凭证模式

通过 `credential_mode` 字段（或 `RABBITMQ_CREDENTIAL_MODE` 环境变量）控制连接凭证的来源。

### 静态凭证（static）

**默认模式**。直接从配置文件或环境变量中读取 `url`，URL 中包含用户名和密码。

```toml
[rabbitmq]
  credential_mode = "static"   # 默认，可省略
  url             = "amqp://guest:guest@localhost:5672/"
```

- ✅ 简单直接，适合开发和小规模部署
- ❌ 密码明文存储在配置/环境变量中，安全性较低

---

### Vault KV 静态凭证（vault-secret）

从 HashiCorp Vault 的 **KV v2 引擎**中读取完整的 RabbitMQ 连接 URL，密码由 Vault 统一管理，应用配置文件中不再出现密码明文。

**配置示例：**

```toml
[rabbitmq]
  credential_mode = "vault-secret"
  vault_path      = "myapp/rabbitmq"   # KV 路径（相对于挂载点）
  vault_url_field = "url"              # Vault 返回数据中的 URL 字段名，默认 "url"

[vault]
  address     = "http://127.0.0.1:8200"
  auth_method = "token"
  token       = "hvs.xxxxxx"
```

**Vault 侧配置（参考）：**

```bash
# 1. 启动 Vault 开发服务
docker run -itd --name=vault \
  --cap-add=IPC_LOCK \
  -p 8200:8200 \
  -e 'VAULT_DEV_ROOT_TOKEN_ID=myroot' \
  -e 'VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200' \
  hashicorp/vault:latest

# 2. 写入 RabbitMQ 连接 URL
docker exec \
  -e VAULT_ADDR='http://127.0.0.1:8200' \
  -e VAULT_TOKEN='myroot' \
  vault vault kv put secret/myapp/rabbitmq \
    url="amqp://user:strongpassword@localhost:5672/vhost"

# 3. 验证读取
docker exec \
  -e VAULT_ADDR='http://127.0.0.1:8200' \
  -e VAULT_TOKEN='myroot' \
  vault vault kv get secret/myapp/rabbitmq
```

> `vault_path` 是相对于挂载点的路径，默认挂载点为 `secret`（对应 vault 模块的 `kv_mount_path`），实际请求路径为 `secret/data/myapp/rabbitmq`，此处只需填 `myapp/rabbitmq`。

**工作流程：**

```
应用启动
  └─→ 连接 Vault（依赖 vault 模块已初始化）
  └─→ 读取 KV 路径中的 url 字段
  └─→ 使用读取到的 URL 建立 RabbitMQ 连接
```

- ✅ 连接 URL（含密码）不出现在应用配置文件中
- ✅ 可以在 Vault 中集中管理和审计凭证访问
- ❌ 当前仅支持静态凭证，Vault 中密码轮换后需重启服务生效

---

## 消息模式

模块封装了三种常用消息模式，通过 `Publisher` 发布，通过 `Consumer` 订阅。

### 队列模式（Direct）

多个消费者竞争消费同一队列，常用于任务分发。

```go
// 发布
pub.Publish(ctx, rabbitmq.NewQueueMessage("task-queue", []byte(`{"task":"process"}`)))

// 消费（竞争消费，多实例时每条消息只被一个实例处理）
sub.DirectSubscribe(ctx, "task-exchange", "task-queue", func(ctx context.Context, msg *rabbitmq.Message) error {
    fmt.Println("received:", string(msg.Body))
    return nil
})
```

### 发布订阅模式（Fanout）

消息广播到所有已绑定队列，每个订阅者都能收到完整消息副本，常用于事件通知。

```go
// 发布
pub.Publish(ctx, rabbitmq.NewFanoutMessage("user-events", []byte(`{"event":"login"}`)))

// 订阅（每个实例都会收到消息）
sub.FanoutSubscribe(ctx, "user-events", func(ctx context.Context, msg *rabbitmq.Message) error {
    fmt.Println("event:", string(msg.Body))
    return nil
})
```

### 主题模式（Topic）

基于通配符路由键匹配，灵活过滤消息，常用于日志分类、分级告警等场景。

| 通配符 | 含义 |
|--------|------|
| `*` | 匹配一个单词 |
| `#` | 匹配零或多个单词 |

```go
// 发布（路由键 order.paid）
pub.Publish(ctx, rabbitmq.NewTopicMessage("logs", "order.paid", []byte(`{"order_id":123}`)))

// 订阅所有 order.* 消息
sub.TopicSubscribe(ctx, "logs", "order.*", func(ctx context.Context, msg *rabbitmq.Message) error {
    fmt.Println("order event:", string(msg.Body))
    return nil
})

// 订阅所有消息
sub.TopicSubscribe(ctx, "logs", "#", func(ctx context.Context, msg *rabbitmq.Message) error {
    fmt.Println("all:", string(msg.Body))
    return nil
})
```

---

## 链路追踪

当 `trace` 为 `true` 且 mcube 的 `trace` 模块启用了 OpenTelemetry 时，每次 `Publish` 和消息消费都会自动创建 Span，并通过消息 Headers 传播 TraceContext。

```toml
[rabbitmq]
  trace = true

[trace]
  enable   = true
  endpoint = "http://jaeger:14268/api/traces"
```

---

## 完整配置参数

| 参数 | 环境变量 | 类型 | 默认值 | 说明 |
|------|----------|------|--------|------|
| `url` | `RABBITMQ_URL` | string | `amqp://guest:guest@localhost:5672/` | RabbitMQ 连接 URL |
| `timeout` | `RABBITMQ_TIMEOUT` | duration | `5s` | 连接超时时间 |
| `heart_beat` | `RABBITMQ_HEART_BEAT` | duration | `10s` | 心跳间隔 |
| `reconect_interval` | `RABBITMQ_RECONNECT_INTERVAL` | duration | `10s` | 断线重连间隔 |
| `max_reconnect_attempts` | `RABBITMQ_MAX_RECONNECT_ATTEMPTS` | int | `0` | 最大重连次数，`0` 为无限重连 |
| `trace` | `RABBITMQ_TRACE` | bool | `false` | 启用 OpenTelemetry 链路追踪 |
| `credential_mode` | `RABBITMQ_CREDENTIAL_MODE` | string | `static` | 凭证模式：`static` / `vault-secret` |
| `vault_path` | `RABBITMQ_VAULT_PATH` | string | — | Vault KV 路径（相对于挂载点） |
| `vault_url_field` | `RABBITMQ_VAULT_URL_FIELD` | string | `url` | Vault 返回数据中的连接 URL 字段名 |
