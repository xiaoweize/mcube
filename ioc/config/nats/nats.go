package nats

import (
	"context"
	"fmt"

	"github.com/infraboard/mcube/v2/ioc"
	"github.com/infraboard/mcube/v2/ioc/config/log"
	"github.com/infraboard/mcube/v2/ioc/config/vault"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

type CREDENTIAL_MODE string

const (
	CREDENTIAL_MODE_STATIC       CREDENTIAL_MODE = "static"       // 静态凭证（配置文件）
	CREDENTIAL_MODE_VAULT_SECRET CREDENTIAL_MODE = "vault-secret" // Vault KV 静态凭证
)

func init() {
	ioc.Config().Registry(&Client{
		URL:             nats.DefaultURL,
		CredentialMode:  CREDENTIAL_MODE_STATIC,
		VaultTokenField: "token",
	})
}

type Client struct {
	ioc.ObjectImpl
	log *zerolog.Logger

	// nats服务地址
	URL string `toml:"url" json:"url" yaml:"url"  env:"URL"`
	// Token
	Token string `toml:"token" json:"token" yaml:"token"  env:"TOKEN"`

	// Vault 凭证配置
	CredentialMode CREDENTIAL_MODE `json:"credential_mode" yaml:"credential_mode" toml:"credential_mode" env:"CREDENTIAL_MODE"`
	// VaultPath Vault KV 路径
	VaultPath string `json:"vault_path" yaml:"vault_path" toml:"vault_path" env:"VAULT_PATH"`
	// VaultTokenField Vault 返回数据中的 Token 字段名，默认 "token"
	VaultTokenField string `json:"vault_token_field" yaml:"vault_token_field" toml:"vault_token_field" env:"VAULT_TOKEN_FIELD"`

	conn *nats.Conn
}

func (c *Client) Name() string {
	return APP_NAME
}

func (i *Client) Priority() int {
	return 695
}

func (c *Client) Options() (opts []nats.Option) {
	if c.Token != "" {
		opts = append(opts, nats.Token(c.Token))
	}
	return
}

func (c *Client) Init() error {
	c.log = log.Sub(c.Name())

	// 初始化默认值
	c.setDefaults()

	// 根据凭证模式加载凭证
	if err := c.loadCredentials(); err != nil {
		return fmt.Errorf("load credentials: %w", err)
	}

	conn, err := nats.Connect(c.URL, c.Options()...)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

// setDefaults 设置默认值
func (c *Client) setDefaults() {
	if c.VaultTokenField == "" {
		c.VaultTokenField = "token"
	}
	if c.CredentialMode == "" {
		c.CredentialMode = CREDENTIAL_MODE_STATIC
	}
}

// loadCredentials 根据凭证模式加载凭证
func (c *Client) loadCredentials() error {
	switch c.CredentialMode {
	case CREDENTIAL_MODE_STATIC:
		c.log.Info().Msg("using static credentials from config file")
		return nil
	case CREDENTIAL_MODE_VAULT_SECRET:
		return c.loadVaultSecretCredentials()
	default:
		return fmt.Errorf("unsupported credential mode: %s", c.CredentialMode)
	}
}

// loadVaultSecretCredentials 从 Vault KV 加载 Token
func (c *Client) loadVaultSecretCredentials() error {
	if c.VaultPath == "" {
		return fmt.Errorf("vault_path is required for vault-secret mode")
	}

	vaultClient := vault.Client()
	if vaultClient == nil {
		return fmt.Errorf("vault client not initialized, please ensure vault config is enabled")
	}

	resp, err := vault.ReadSecret(context.Background(), c.VaultPath)
	if err != nil {
		return fmt.Errorf("read vault secret from %s: %w", c.VaultPath, err)
	}

	token, ok := resp.Data.Data[c.VaultTokenField].(string)
	if !ok {
		return fmt.Errorf("field '%s' not found in vault secret at %s", c.VaultTokenField, c.VaultPath)
	}

	c.Token = token
	c.log.Info().Msgf("loaded nats token from vault KV (path=%s)", c.VaultPath)
	return nil
}

func (c *Client) Close(ctx context.Context) {
	if c.conn != nil {
		c.conn.Drain()
	}
}
