# MongoDB 配置模块

基于 [mongo-driver](https://github.com/mongodb/mongo-go-driver) 的 MongoDB 客户端配置模块，集成到 mcube IoC 容器中，支持 OpenTelemetry 链路追踪，以及静态凭证、Vault KV 静态凭证两种凭证加载模式。

## 目录

- [开发环境搭建](#开发环境搭建)
- [快速开始](#快速开始)
- [配置方式](#配置方式)
  - [环境变量](#环境变量)
  - [TOML 文件](#toml-文件)
- [凭证模式](#凭证模式)
  - [静态凭证（static）](#静态凭证static)
  - [Vault KV 静态凭证（vault-secret）](#vault-kv-静态凭证vault-secret)
- [链路追踪](#链路追踪)
- [完整配置参数](#完整配置参数)

---

## 开发环境搭建

```sh
docker run -d \
  --name mongodb \
  -p 27017:27017 \
  -e MONGO_INITDB_ROOT_USERNAME=root \
  -e MONGO_INITDB_ROOT_PASSWORD=123456 \
  mongo:7
```

---

## 快速开始

**1. 导入包（触发自动注册）**

```go
import (
    "github.com/infraboard/mcube/v2/ioc"
    _ "github.com/infraboard/mcube/v2/ioc/config/mongo"
)
```

**2. 初始化并获取客户端**

```go
func main() {
    // 从配置文件或环境变量加载
    ioc.DevelopmentSetupWithPath("etc/application.toml")

    // 获取 *mongo.Database 对象
    db := mongo.DB()

    // 或获取原始 *mongo.Client
    client := mongo.Client()
}
```

**3. 最简 TOML 配置（`etc/application.toml`）**

```toml
[mongo]
  endpoints = ["127.0.0.1:27017"]
  username  = "root"
  password  = "123456"
```

---

## 配置方式

### 环境变量

所有配置项均可通过 `MONGO_` 前缀的环境变量设置。

```bash
export MONGO_ENDPOINTS=127.0.0.1:27017     # 多个节点用逗号分隔
export MONGO_USERNAME=root
export MONGO_PASSWORD=123456
export MONGO_DATABASE=myapp
export MONGO_AUTH_DB=admin
export MONGO_TRACE=true

# 凭证模式相关
export MONGO_CREDENTIAL_MODE=static        # static | vault-secret
export MONGO_VAULT_PATH=myapp/mongo
export MONGO_VAULT_USERNAME_FIELD=username
export MONGO_VAULT_PASSWORD_FIELD=password
```

### TOML 文件

```toml
[mongo]
  endpoints = ["127.0.0.1:27017"]   # MongoDB 节点列表
  username  = "root"                # 用户名
  password  = "123456"              # 密码
  database  = "myapp"               # 默认数据库，默认取应用名
  auth_db   = "admin"               # 认证数据库，默认 "admin"
  trace     = true                  # 启用 OpenTelemetry 链路追踪

  # 凭证模式（可选，默认 static）
  credential_mode      = "static"
  vault_path           = ""
  vault_username_field = "username"
  vault_password_field = "password"
```

---

## 凭证模式

通过 `credential_mode` 字段（或 `MONGO_CREDENTIAL_MODE` 环境变量）控制 MongoDB 凭证的来源。

### 静态凭证（static）

**默认模式**。直接从配置文件或环境变量中读取 `username` 和 `password`。

```toml
[mongo]
  credential_mode = "static"   # 默认，可省略
  username        = "root"
  password        = "123456"
```

- ✅ 简单直接，适合开发和小规模部署
- ❌ 密码明文存储在配置文件中，安全性较低
- ❌ 轮换密码需要重启服务

---

### Vault KV 静态凭证（vault-secret）

从 HashiCorp Vault 的 **KV v2 引擎**中读取 MongoDB 用户名和密码，凭证由 Vault 统一管理，应用配置文件中不再出现密码明文。

**配置示例：**

```toml
[mongo]
  credential_mode      = "vault-secret"
  endpoints            = ["127.0.0.1:27017"]
  vault_path           = "myapp/mongo"      # KV 路径（相对于挂载点）
  vault_username_field = "username"          # Vault 返回数据中的用户名字段，默认 "username"
  vault_password_field = "password"          # Vault 返回数据中的密码字段，默认 "password"

[vault]
  address     = "http://127.0.0.1:8200"
  auth_method = "token"
  token       = "myroot"
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

# 2. 写入 MongoDB 凭证
docker exec \
  -e VAULT_ADDR='http://127.0.0.1:8200' \
  -e VAULT_TOKEN='myroot' \
  vault vault kv put secret/myapp/mongo \
    username=root \
    password=123456

# 3. 验证读取
docker exec \
  -e VAULT_ADDR='http://127.0.0.1:8200' \
  -e VAULT_TOKEN='myroot' \
  vault vault kv get secret/myapp/mongo
```

> `vault_path` 填写的是相对于挂载点的路径。默认挂载点为 `secret`（对应 vault 模块的 `kv_mount_path` 配置项），实际请求路径为 `secret/data/myapp/mongo`，此处只需填 `myapp/mongo`。

**工作流程：**

```
应用启动
  └─→ 连接 Vault（依赖 vault 模块已初始化）
  └─→ 读取 KV 路径中的 username / password
  └─→ 使用读取到的凭证创建 MongoDB 客户端
```

- ✅ 密码不出现在应用配置文件中
- ✅ 可以在 Vault 中集中管理密码
- ❌ 密码仍然是静态的，轮换后需要重启服务生效

---

## 链路追踪

当 `trace` 为 `true` 且 mcube 的 `trace` 模块启用了 OpenTelemetry 时，所有 MongoDB 命令会自动创建 Span。

```toml
[mongo]
  trace = true

[trace]
  enable   = true
  endpoint = "http://jaeger:14268/api/traces"
```

> trace 集成基于 [otelmongo](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo)，命令内容默认不记录（`WithCommandAttributeDisabled(true)`），可防止敏感数据泄露到追踪系统。

---

## 完整配置参数

| 参数 | 环境变量 | 类型 | 默认值 | 说明 |
|------|----------|------|--------|------|
| `endpoints` | `MONGO_ENDPOINTS` | []string | `["127.0.0.1:27017"]` | MongoDB 节点列表，逗号分隔 |
| `username` | `MONGO_USERNAME` | string | — | 用户名 |
| `password` | `MONGO_PASSWORD` | string | — | 密码 |
| `database` | `MONGO_DATABASE` | string | 应用名 | 默认数据库名 |
| `auth_db` | `MONGO_AUTH_DB` | string | `admin` | 认证数据库 |
| `trace` | `MONGO_TRACE` | bool | `true` | 启用 OpenTelemetry 链路追踪 |
| `credential_mode` | `MONGO_CREDENTIAL_MODE` | string | `static` | 凭证模式：`static` / `vault-secret` |
| `vault_path` | `MONGO_VAULT_PATH` | string | — | Vault KV 路径（相对于挂载点） |
| `vault_username_field` | `MONGO_VAULT_USERNAME_FIELD` | string | `username` | Vault 返回数据中的用户名字段 |
| `vault_password_field` | `MONGO_VAULT_PASSWORD_FIELD` | string | `password` | Vault 返回数据中的密码字段 |
