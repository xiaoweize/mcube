package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/infraboard/mcube/v2/ioc"
	"github.com/infraboard/mcube/v2/ioc/config/application"
	"github.com/infraboard/mcube/v2/ioc/config/log"
	"github.com/infraboard/mcube/v2/ioc/config/trace"
	"github.com/infraboard/mcube/v2/ioc/config/vault"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

type CREDENTIAL_MODE string

const (
	CREDENTIAL_MODE_STATIC       CREDENTIAL_MODE = "static"       // 静态凭证（配置文件）
	CREDENTIAL_MODE_VAULT_SECRET CREDENTIAL_MODE = "vault-secret" // Vault KV 静态凭证
)

func init() {
	ioc.Config().Registry(defaultConfig)
}

var defaultConfig = &mongoDB{
	Database:  application.Get().GetAppName(),
	AuthDB:    "admin",
	Endpoints: []string{"127.0.0.1:27017"},
	Trace:     true,

	// Vault 凭证默认配置
	CredentialMode:     CREDENTIAL_MODE_STATIC,
	VaultUsernameField: "username",
	VaultPasswordField: "password",
}

type mongoDB struct {
	Endpoints []string `toml:"endpoints" json:"endpoints" yaml:"endpoints" env:"ENDPOINTS" envSeparator:","`
	UserName  string   `toml:"username" json:"username" yaml:"username"  env:"USERNAME"`
	Password  string   `toml:"password" json:"password" yaml:"password"  env:"PASSWORD"`
	Database  string   `toml:"database" json:"database" yaml:"database"  env:"DATABASE"`
	AuthDB    string   `toml:"auth_db" json:"auth_db" yaml:"auth_db"  env:"AUTH_DB"`
	Trace     bool     `toml:"trace" json:"trace" yaml:"trace"  env:"TRACE"`

	// Vault 凭证配置
	CredentialMode CREDENTIAL_MODE `json:"credential_mode" yaml:"credential_mode" toml:"credential_mode" env:"CREDENTIAL_MODE"`
	// VaultPath Vault KV 路径
	VaultPath string `json:"vault_path" yaml:"vault_path" toml:"vault_path" env:"VAULT_PATH"`
	// VaultUsernameField Vault 返回数据中的用户名字段名，默认 "username"
	VaultUsernameField string `json:"vault_username_field" yaml:"vault_username_field" toml:"vault_username_field" env:"VAULT_USERNAME_FIELD"`
	// VaultPasswordField Vault 返回数据中的密码字段名，默认 "password"
	VaultPasswordField string `json:"vault_password_field" yaml:"vault_password_field" toml:"vault_password_field" env:"VAULT_PASSWORD_FIELD"`

	client *mongo.Client
	ioc.ObjectImpl
	log *zerolog.Logger
}

func (m *mongoDB) Name() string {
	return AppName
}

func (i *mongoDB) Priority() int {
	return 698
}

func (m *mongoDB) Init() error {
	m.log = log.Sub(m.Name())

	// 初始化默认值
	m.setDefaults()

	// 根据凭证模式加载凭证
	if err := m.loadCredentials(); err != nil {
		return fmt.Errorf("load credentials: %w", err)
	}

	conn, err := m.getClient()
	if err != nil {
		return err
	}
	m.client = conn
	return nil
}

// setDefaults 设置默认值
func (m *mongoDB) setDefaults() {
	if m.VaultUsernameField == "" {
		m.VaultUsernameField = "username"
	}
	if m.VaultPasswordField == "" {
		m.VaultPasswordField = "password"
	}
	if m.CredentialMode == "" {
		m.CredentialMode = CREDENTIAL_MODE_STATIC
	}
}

// loadCredentials 根据凭证模式加载凭证
func (m *mongoDB) loadCredentials() error {
	switch m.CredentialMode {
	case CREDENTIAL_MODE_STATIC:
		m.log.Info().Msg("using static credentials from config file")
		return nil
	case CREDENTIAL_MODE_VAULT_SECRET:
		return m.loadVaultSecretCredentials()
	default:
		return fmt.Errorf("unsupported credential mode: %s", m.CredentialMode)
	}
}

// loadVaultSecretCredentials 从 Vault KV 加载凭证
func (m *mongoDB) loadVaultSecretCredentials() error {
	if m.VaultPath == "" {
		return fmt.Errorf("vault_path is required for vault-secret mode")
	}

	vaultClient := vault.Client()
	if vaultClient == nil {
		return fmt.Errorf("vault client not initialized, please ensure vault config is enabled")
	}

	resp, err := vault.ReadSecret(context.Background(), m.VaultPath)
	if err != nil {
		return fmt.Errorf("read vault secret from %s: %w", m.VaultPath, err)
	}

	username, ok := resp.Data.Data[m.VaultUsernameField].(string)
	if !ok {
		return fmt.Errorf("field '%s' not found in vault secret at %s", m.VaultUsernameField, m.VaultPath)
	}

	password, ok := resp.Data.Data[m.VaultPasswordField].(string)
	if !ok {
		return fmt.Errorf("field '%s' not found in vault secret at %s", m.VaultPasswordField, m.VaultPath)
	}

	m.UserName = username
	m.Password = password

	m.log.Info().Msgf("loaded credentials from vault KV (path=%s)", m.VaultPath)
	return nil
}

// 关闭数据库连接
func (m *mongoDB) Close(ctx context.Context) {
	if m.client == nil {
		return
	}

	err := m.client.Disconnect(ctx)
	if err != nil {
		m.log.Error().Msgf("close error, %s", err)
	}
}

func (m *mongoDB) GetAuthDB() string {
	if m.AuthDB != "" {
		return m.AuthDB
	}

	return m.Database
}

func (m *mongoDB) GetDB() *mongo.Database {
	return m.client.Database(m.Database)
}

// Client 获取一个全局的mongodb客户端连接
func (m *mongoDB) Client() *mongo.Client {
	return m.client
}

func (m *mongoDB) getClient() (*mongo.Client, error) {
	opts := options.Client()

	if m.UserName != "" && m.Password != "" {
		cred := options.Credential{
			AuthSource: m.GetAuthDB(),
		}

		cred.Username = m.UserName
		cred.Password = m.Password
		cred.PasswordSet = true
		opts.SetAuth(cred)
	}
	opts.SetHosts(m.Endpoints)
	opts.SetConnectTimeout(5 * time.Second)
	if trace.Get().Enable && m.Trace {
		m.log.Info().Msg("enable mongodb trace")
		opts.Monitor = otelmongo.NewMonitor(
			otelmongo.WithCommandAttributeDisabled(true),
		)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*5))
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("new mongodb client error, %s", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("ping mongodb server(%s) error, %s", m.Endpoints, err)
	}

	return client, nil
}
