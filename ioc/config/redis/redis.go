package redis

import (
	"context"
	"fmt"

	"github.com/infraboard/mcube/v2/ioc"
	"github.com/infraboard/mcube/v2/ioc/config/log"
	"github.com/infraboard/mcube/v2/ioc/config/trace"
	"github.com/infraboard/mcube/v2/ioc/config/vault"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type CREDENTIAL_MODE string

const (
	CREDENTIAL_MODE_STATIC       CREDENTIAL_MODE = "static"       // 静态凭证（配置文件）
	CREDENTIAL_MODE_VAULT_SECRET CREDENTIAL_MODE = "vault-secret" // Vault KV 静态凭证
)

func init() {
	ioc.Config().Registry(defaultConfig)
}

var defaultConfig = &Redis{
	DB:        0,
	Endpoints: []string{"127.0.0.1:6379"},
	Trace:     true,

	// Vault 凭证默认配置
	CredentialMode:     CREDENTIAL_MODE_STATIC,
	VaultUsernameField: "username",
	VaultPasswordField: "password",
}

type Redis struct {
	ioc.ObjectImpl
	Endpoints []string `toml:"endpoints" json:"endpoints" yaml:"endpoints" env:"ENDPOINTS" envSeparator:","`
	DB        int      `toml:"db" json:"db" yaml:"db"  env:"DB"`
	UserName  string   `toml:"username" json:"username" yaml:"username"  env:"USERNAME"`
	Password  string   `toml:"password" json:"password" yaml:"password"  env:"PASSWORD"`
	Trace     bool     `toml:"trace" json:"trace" yaml:"trace"  env:"TRACE"`
	Metric    bool     `toml:"metric" json:"metric" yaml:"metric"  env:"METRIC"`

	// Vault 凭证配置
	CredentialMode CREDENTIAL_MODE `json:"credential_mode" yaml:"credential_mode" toml:"credential_mode" env:"CREDENTIAL_MODE"`
	// VaultPath Vault KV 路径
	VaultPath string `json:"vault_path" yaml:"vault_path" toml:"vault_path" env:"VAULT_PATH"`
	// VaultUsernameField Vault 返回数据中的用户名字段名，默认 "username"
	VaultUsernameField string `json:"vault_username_field" yaml:"vault_username_field" toml:"vault_username_field" env:"VAULT_USERNAME_FIELD"`
	// VaultPasswordField Vault 返回数据中的密码字段名，默认 "password"
	VaultPasswordField string `json:"vault_password_field" yaml:"vault_password_field" toml:"vault_password_field" env:"VAULT_PASSWORD_FIELD"`

	client redis.UniversalClient
	log    *zerolog.Logger
}

func (m *Redis) Name() string {
	return AppName
}

func (i *Redis) Priority() int {
	return 697
}

// https://opentelemetry.io/ecosystem/registry/?s=redis&component=&language=go
// https://github.com/redis/go-redis/tree/master/extra/redisotel
func (m *Redis) Init() error {
	m.log = log.Sub(m.Name())

	// 初始化默认值
	m.setDefaults()

	// 根据凭证模式加载凭证
	if err := m.loadCredentials(); err != nil {
		return fmt.Errorf("load credentials: %w", err)
	}

	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    m.Endpoints,
		DB:       m.DB,
		Username: m.UserName,
		Password: m.Password,
	})

	if trace.Get().Enable && m.Trace {
		m.log.Info().Msg("enable redis trace")
		if err := redisotel.InstrumentTracing(rdb); err != nil {
			return err
		}
	}

	if m.Metric {
		if err := redisotel.InstrumentMetrics(rdb); err != nil {
			return err
		}
	}

	m.client = rdb
	return nil
}

// 关闭数据库连接
func (m *Redis) Close(ctx context.Context) {
	if m.client == nil {
		return
	}

	err := m.client.Close()
	if err != nil {
		m.log.Error().Msgf("close redis client error, %s", err)
	}
}

// setDefaults 设置默认值
func (m *Redis) setDefaults() {
	if m.VaultUsernameField == "" {
		m.VaultUsernameField = "username"
	}
	if m.VaultPasswordField == "" {
		m.VaultPasswordField = "password"
	}
	// 处理空字符串，兼容旧配置
	if m.CredentialMode == "" {
		m.CredentialMode = CREDENTIAL_MODE_STATIC
	}
}

// loadCredentials 根据凭证模式加载凭证
func (m *Redis) loadCredentials() error {
	switch m.CredentialMode {
	case CREDENTIAL_MODE_STATIC:
		return m.loadStaticCredentials()
	case CREDENTIAL_MODE_VAULT_SECRET:
		return m.loadVaultSecretCredentials()
	default:
		return fmt.Errorf("unsupported credential mode: %s", m.CredentialMode)
	}
}

// loadStaticCredentials 加载静态凭证
func (m *Redis) loadStaticCredentials() error {
	if m.Password == "" {
		m.log.Warn().Msg("password is empty for static credential mode")
	}
	m.log.Info().Msg("using static credentials from config file")
	return nil
}

// loadVaultSecretCredentials 从 Vault KV 加载静态凭证
func (m *Redis) loadVaultSecretCredentials() error {
	if m.VaultPath == "" {
		return fmt.Errorf("vault_path is required for vault-secret mode")
	}

	vaultClient := vault.Client()
	if vaultClient == nil {
		return fmt.Errorf("vault client not initialized, please ensure vault config is enabled")
	}

	ctx := context.Background()
	resp, err := vault.ReadSecret(ctx, m.VaultPath)
	if err != nil {
		return fmt.Errorf("read vault secret from %s: %w", m.VaultPath, err)
	}

	// 提取 username（可选）
	if username, ok := resp.Data.Data[m.VaultUsernameField].(string); ok {
		m.UserName = username
	}

	// 提取 password
	password, ok := resp.Data.Data[m.VaultPasswordField].(string)
	if !ok {
		return fmt.Errorf("field '%s' not found in vault secret at %s", m.VaultPasswordField, m.VaultPath)
	}
	m.Password = password

	m.log.Info().Msgf("loaded credentials from vault KV (path=%s)", m.VaultPath)
	return nil
}
