# NATS 配置模块

基于 [nats.go](https://github.com/nats-io/nats.go) 的 NATS 客户端配置模块，集成到 mcube IoC 容器中，支持 Token 认证，以及静态凭证、Vault KV 静态凭证两种凭证加载模式。

## 目录

- [开发环境搭建](#开发环境搭建)
- [快速开始](#快速开始)
- [配置方式](#配置方式)
  - [环境变量](#环境变量)
  - [TOML 文件](#toml-文件)
- [凭证模式](#凭证模式)
  - [静态凭证（static）](#静态凭证static)
  - [Vault KV 静态凭证（vault-secret）](#vault-kv-静态凭证vault-secret)
- [完整配置参数](#完整配置参数)

---

## 开发环境搭建

```sh
docker run -d \
  --name nats \
  -p 4222:4222 \
  -p 8222:8222 \
  nats:latest
```

启用 Token 认证：

```sh
docker run -d \
  --name nats \
  -p 4222:4222 \
  nats:latest --auth my-nats-token
```

通过 http://localhost:8222 访问监控界面（需加 `-p 8222:8222`）。

---

## 快速开始

**1. 导入包（触发自动注册）**

```go
import (
    "github.com/infraboard/mcube/v2/ioc"
    _ "github.com/infraboard/mcube/v2/ioc/config/nats"
)
```

**2. 初始化并使用**

```go
func main() {
    ioc.DevelopmentSetupWithPath("etc/application.toml")

    conn := nats.Get()

    // 发布消息
    conn.Publish("event_bus", []byte("hello"))
    conn.Flush()

    // 订阅消息
    conn.Subscribe("event_bus", func(msg *nats.Msg) {
        fmt.Println(string(msg.Data))
    })
}
```

**3. 最简 TOML 配置（`etc/application.toml`）**

```toml
[nats]
  url = "nats://127.0.0.1:4222"
```

---

## 配置方式

### 环境变量

所有配置项均可通过 `NATS_` 前缀的环境变量设置。

```bash
export NATS_URL=nats://127.0.0.1:4222
export NATS_TOKEN=my-nats-token

# 凭证模式相关
export NATS_CREDENTIAL_MODE=static       # static | vault-secret
export NATS_VAULT_PATH=myapp/nats
export NATS_VAULT_TOKEN_FIELD=token
```

### TOML 文件

```toml
[nats]
  url   = "nats://127.0.0.1:4222"   # NATS 服务地址，默认 nats://127.0.0.1:4222
  token = ""                         # Token 认证（静态模式下填写）

  # 凭证模式（可选，默认 static）
  credential_mode   = "static"
  vault_path        = ""
  vault_token_field = "token"
```

---

## 凭证模式

通过 `credential_mode` 字段（或 `NATS_CREDENTIAL_MODE` 环境变量）控制 NATS Token 凭证的来源。

### 静态凭证（static）

**默认模式**。直接从配置文件或环境变量中读取 `token`。未配置 Token 时以匿名方式连接。

```toml
[nats]
  credential_mode = "static"   # 默认，可省略
  url             = "nats://127.0.0.1:4222"
  token           = "my-nats-token"
```

- ✅ 简单直接，适合开发和小规模部署
- ❌ Token 明文存储在配置文件中，安全性较低

---

### Vault KV 静态凭证（vault-secret）

从 HashiCorp Vault 的 **KV v2 引擎**中读取 NATS Token，凭证由 Vault 统一管理，应用配置文件中不再出现 Token 明文。

**配置示例：**

```toml
[nats]
  credential_mode   = "vault-secret"
  url               = "nats://127.0.0.1:4222"
  vault_path        = "myapp/nats"   # KV 路径（相对于挂载点）
  vault_token_field = "token"        # Vault 返回数据中的 Token 字段名，默认 "token"

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

# 2. 写入 NATS Token
docker exec \
  -e VAULT_ADDR='http://127.0.0.1:8200' \
  -e VAULT_TOKEN='myroot' \
  vault vault kv put secret/myapp/nats \
    token=my-nats-auth-token

# 3. 验证读取
docker exec \
  -e VAULT_ADDR='http://127.0.0.1:8200' \
  -e VAULT_TOKEN='myroot' \
  vault vault kv get secret/myapp/nats
```

> `vault_path` 填写的是相对于挂载点的路径。默认挂载点为 `secret`（对应 vault 模块的 `kv_mount_path` 配置项），实际请求路径为 `secret/data/myapp/nats`，此处只需填 `myapp/nats`。

**工作流程：**

```
应用启动
  └─→ 连接 Vault（依赖 vault 模块已初始化）
  └─→ 读取 KV 路径中的 token 字段
  └─→ 使用读取到的 Token 建立 NATS 连接
```

- ✅ Token 不出现在应用配置文件中
- ✅ 可以在 Vault 中集中管理 Token
- ❌ Token 仍然是静态的，轮换后需要重启服务生效

---

## 完整配置参数

| 参数 | 环境变量 | 类型 | 默认值 | 说明 |
|------|----------|------|--------|------|
| `url` | `NATS_URL` | string | `nats://127.0.0.1:4222` | NATS 服务地址 |
| `token` | `NATS_TOKEN` | string | — | Token 认证凭证 |
| `credential_mode` | `NATS_CREDENTIAL_MODE` | string | `static` | 凭证模式：`static` / `vault-secret` |
| `vault_path` | `NATS_VAULT_PATH` | string | — | Vault KV 路径（相对于挂载点） |
| `vault_token_field` | `NATS_VAULT_TOKEN_FIELD` | string | `token` | Vault 返回数据中的 Token 字段名 |

