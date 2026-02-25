# Redis 配置模块

基于 [go-redis](https://github.com/redis/go-redis) 的 Redis 客户端配置模块，集成到 mcube IoC 容器中，支持单节点、集群、哨兵等多种部署模式（通过 `UniversalClient` 自动适配），以及静态凭证、Vault KV 静态凭证两种凭证加载模式。

## 目录

- [快速开始](#快速开始)
- [配置方式](#配置方式)
  - [环境变量](#环境变量)
  - [TOML 文件](#toml-文件)
  - [配置优先级](#配置优先级)
- [凭证模式](#凭证模式)
  - [静态凭证（static）](#静态凭证static)
  - [Vault KV 静态凭证（vault-secret）](#vault-kv-静态凭证vault-secret)
- [链路追踪](#链路追踪)
- [完整配置参数](#完整配置参数)

---

## 快速开始

**1. 导入包（触发自动注册）**

```go
import (
    "github.com/infraboard/mcube/v2/ioc"
    _ "github.com/infraboard/mcube/v2/ioc/config/redis" // 仅注册，不使用
)
```

**2. 初始化并获取客户端**

```go
func main() {
    // 方式一：开发环境，从配置文件加载
    ioc.DevelopmentSetupWithPath("etc/application.toml")

    // 方式二：生产环境，从环境变量加载
    ioc.ConfigIocObject(ioc.NewLoadConfigRequest())

    // 获取 redis.UniversalClient 对象
    rdb := redis.Client()
    fmt.Println(rdb)
}
```

**3. 最简 TOML 配置（`etc/application.toml`）**

```toml
[redis]
  endpoints = ["127.0.0.1:6379"]
  password  = "123456"
```

---

## 配置方式

### 环境变量

所有配置项均可通过 `REDIS_` 前缀的环境变量设置。

```bash
export REDIS_ENDPOINTS=127.0.0.1:6379        # 多个节点用逗号分隔
export REDIS_DB=0
export REDIS_USERNAME=
export REDIS_PASSWORD=123456
export REDIS_TRACE=true
export REDIS_METRIC=false
```

凭证模式相关：

```bash
export REDIS_CREDENTIAL_MODE=static          # static | vault-secret
export REDIS_VAULT_PATH=myapp/redis
export REDIS_VAULT_USERNAME_FIELD=username
export REDIS_VAULT_PASSWORD_FIELD=password
```

### TOML 文件

```toml
[redis]
  endpoints = ["127.0.0.1:6379"]   # Redis 节点列表，多节点时自动切换为集群/哨兵模式
  db        = 0                    # 数据库编号，集群模式下无效
  username  = ""                   # ACL 用户名（Redis 6.0+）
  password  = ""                   # 密码
  trace     = true                 # 启用 OpenTelemetry 链路追踪
  metric    = false                # 启用 Prometheus 指标采集

  # 凭证模式（可选，默认 static）
  credential_mode      = "static"
  vault_path           = ""
  vault_username_field = "username"
  vault_password_field = "password"
```

### 配置优先级

```
环境变量 > TOML/YAML 文件 > 代码默认值
```

---

## 凭证模式

通过 `credential_mode` 字段（或 `REDIS_CREDENTIAL_MODE` 环境变量）控制 Redis 凭证的来源，支持两种模式。

### 静态凭证（static）

**默认模式**。直接从配置文件或环境变量中读取 `username` 和 `password`。

```toml
[redis]
  credential_mode = "static"   # 默认，可省略
  password        = "123456"
```

- ✅ 简单直接，适合开发和小规模部署
- ❌ 密码明文存储在配置文件中，安全性较低
- ❌ 轮换密码需要重启服务

---

### Vault KV 静态凭证（vault-secret）

从 HashiCorp Vault 的 **KV v2 引擎**中读取 Redis 密码，凭证由 Vault 统一管理，应用配置文件中不再出现密码明文。

**配置示例：**

```toml
[redis]
  credential_mode      = "vault-secret"
  endpoints            = ["127.0.0.1:6379"]
  vault_path           = "myapp/redis"      # KV 路径（相对于挂载点）
  vault_username_field = "username"          # Vault 返回数据中的用户名字段，默认 "username"（可选）
  vault_password_field = "password"          # Vault 返回数据中的密码字段，默认 "password"

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

# 2. 启用 KV v2 引擎（开发模式已自动启用）
docker exec \
  -e VAULT_ADDR='http://127.0.0.1:8200' \
  -e VAULT_TOKEN='myroot' \
  vault vault secrets enable -path=secret kv-v2

# 3. 写入 Redis 凭证
docker exec \
  -e VAULT_ADDR='http://127.0.0.1:8200' \
  -e VAULT_TOKEN='myroot' \
  vault vault kv put secret/myapp/redis \
    password=my-redis-password

# 若 Redis 开启了 ACL，也可写入用户名
docker exec \
  -e VAULT_ADDR='http://127.0.0.1:8200' \
  -e VAULT_TOKEN='myroot' \
  vault vault kv put secret/myapp/redis \
    username=myuser \
    password=my-redis-password

# 4. 验证读取
docker exec \
  -e VAULT_ADDR='http://127.0.0.1:8200' \
  -e VAULT_TOKEN='myroot' \
  vault vault kv get secret/myapp/redis
```

> `vault_path` 填写的是相对于挂载点的路径。默认挂载点为 `secret`（对应 vault 模块的 `kv_mount_path` 配置项），实际请求路径为 `secret/data/myapp/redis`，此处只需填 `myapp/redis`。

**注意事项：**

- `username` 字段是**可选**的。对于未开启 ACL 的 Redis，只需在 Vault 中存储 `password`。
- `password` 字段是**必须**的，若 Vault 中找不到该字段会返回错误。
- 当前仅支持静态凭证（vault-secret），不支持动态凭证。Vault 中的密码轮换后需要重启服务生效。

**工作流程：**

```
应用启动
  └─→ 连接 Vault（依赖 vault 模块已初始化）
  └─→ 读取 KV 路径中的 password（以及可选的 username）
  └─→ 使用读取到的凭证创建 Redis 客户端
```

- ✅ 密码不出现在应用配置文件中
- ✅ 可以在 Vault 中集中管理密码
- ❌ 密码仍然是静态的，轮换后需要重启服务

---

## 链路追踪

当 `trace` 配置项为 `true` 且 mcube 的 `trace` 模块也启用了 OpenTelemetry 时，所有 Redis 命令会自动创建 Span，记录：

- 命令名称
- 执行耗时
- 节点地址

```toml
[redis]
  trace = true    # 开启 Redis trace，默认 true

[trace]
  enable   = true
  endpoint = "http://jaeger:14268/api/traces"
```

> trace 集成基于 [redisotel](https://github.com/redis/go-redis/tree/master/extra/redisotel)，与 mcube 的 `trace` 模块共享同一 TracerProvider。

**Prometheus 指标（可选）：**

```toml
[redis]
  metric = true   # 开启 Prometheus 指标，默认 false
```

开启后会通过 `redisotel` 自动上报连接池大小、命令执行次数、延迟分布等指标。

---

## 完整配置参数

| 参数 | 环境变量 | 类型 | 默认值 | 说明 |
|------|----------|------|--------|------|
| `endpoints` | `REDIS_ENDPOINTS` | []string | `["127.0.0.1:6379"]` | Redis 节点列表，逗号分隔 |
| `db` | `REDIS_DB` | int | `0` | 数据库编号（集群模式无效） |
| `username` | `REDIS_USERNAME` | string | — | ACL 用户名（Redis 6.0+） |
| `password` | `REDIS_PASSWORD` | string | — | 密码 |
| `trace` | `REDIS_TRACE` | bool | `true` | 启用 OpenTelemetry 链路追踪 |
| `metric` | `REDIS_METRIC` | bool | `false` | 启用 Prometheus 指标采集 |
| `credential_mode` | `REDIS_CREDENTIAL_MODE` | string | `static` | 凭证模式：`static` / `vault-secret` |
| `vault_path` | `REDIS_VAULT_PATH` | string | — | Vault KV 路径（相对于挂载点） |
| `vault_username_field` | `REDIS_VAULT_USERNAME_FIELD` | string | `username` | Vault 返回数据中的用户名字段（可选） |
| `vault_password_field` | `REDIS_VAULT_PASSWORD_FIELD` | string | `password` | Vault 返回数据中的密码字段 |
