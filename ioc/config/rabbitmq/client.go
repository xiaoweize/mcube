package rabbitmq

import (
	"context"
	"fmt"
	"time"

	"github.com/infraboard/mcube/v2/ioc"
	"github.com/infraboard/mcube/v2/ioc/config/log"
	"github.com/infraboard/mcube/v2/ioc/config/vault"
	"github.com/rs/zerolog"
)

func init() {
	ioc.Config().Registry(&Client{
		RabbitConnConfig: RabbitConnConfig{
			URL:               "amqp://guest:guest@localhost:5672/",
			Timeout:           5 * time.Second,
			Heartbeat:         10 * time.Second,
			ReconnectInterval: 10 * time.Second,
		},
	})
}

type Client struct {
	ioc.ObjectImpl

	// 连接配置
	RabbitConnConfig

	conn *RabbitConn
	log  *zerolog.Logger
}

func (c *Client) Name() string {
	return APP_NAME
}

func (i *Client) Priority() int {
	return 695
}

func (c *Client) Init() error {
	c.log = log.Sub(APP_NAME)

	// 初始化默认值
	c.setDefaults()

	// 根据凭证模式加载凭证
	if err := c.loadCredentials(); err != nil {
		return fmt.Errorf("load credentials: %w", err)
	}

	conn, err := NewRabbitConn(c.RabbitConnConfig)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

// setDefaults 设置默认值
func (c *Client) setDefaults() {
	if c.VaultURLField == "" {
		c.VaultURLField = "url"
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

// loadVaultSecretCredentials 从 Vault KV 加载连接 URL
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

	url, ok := resp.Data.Data[c.VaultURLField].(string)
	if !ok {
		return fmt.Errorf("field '%s' not found in vault secret at %s", c.VaultURLField, c.VaultPath)
	}

	c.URL = url
	c.log.Info().Msgf("loaded rabbitmq url from vault KV (path=%s)", c.VaultPath)
	return nil
}

func (c *Client) Close(ctx context.Context) {
	if c.conn != nil {
		c.conn.Close()
	}
}
