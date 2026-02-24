package datasource

import (
	"context"
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/hashicorp/vault-client-go/schema"
	"github.com/infraboard/mcube/v2/ioc"
	"github.com/infraboard/mcube/v2/ioc/config/application"
	"github.com/infraboard/mcube/v2/ioc/config/log"
	"github.com/infraboard/mcube/v2/ioc/config/trace"
	"github.com/infraboard/mcube/v2/ioc/config/vault"
	"github.com/rs/zerolog"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func init() {
	ioc.Config().Registry(defaultConfig)
}

var defaultConfig = &dataSource{
	Provider:    PROVIDER_MYSQL,
	Host:        "127.0.0.1",
	Port:        3306,
	DB:          application.Get().Name(),
	AutoMigrate: false,
	Debug:       false,
	Trace:       true,

	// Vault 凭证默认配置
	CredentialMode:      CREDENTIAL_MODE_STATIC,
	VaultUsernameField:  "username",
	VaultPasswordField:  "password",
	VaultAutoRenew:      true,
	VaultRenewThreshold: 0.8,

	SkipDefaultTransaction: false,
	DryRun:                 false,
	PrepareStmt:            true,
}

type dataSource struct {
	ioc.ObjectImpl
	Provider    PROVIDER `json:"provider" yaml:"provider" toml:"provider" env:"PROVIDER"`
	Host        string   `json:"host" yaml:"host" toml:"host" env:"HOST"`
	Port        int      `json:"port" yaml:"port" toml:"port" env:"PORT"`
	DB          string   `json:"database" yaml:"database" toml:"database" env:"DB"`
	Username    string   `json:"username" yaml:"username" toml:"username" env:"USERNAME"`
	Password    string   `json:"password" yaml:"password" toml:"password" env:"PASSWORD"`
	AutoMigrate bool     `json:"auto_migrate" yaml:"auto_migrate" toml:"auto_migrate" env:"AUTO_MIGRATE"`
	Debug       bool     `json:"debug" yaml:"debug" toml:"debug" env:"DEBUG"`
	Trace       bool     `toml:"trace" json:"trace" yaml:"trace"  env:"TRACE"`

	// Vault 凭证配置
	CredentialMode CREDENTIAL_MODE `json:"credential_mode" yaml:"credential_mode" toml:"credential_mode" env:"CREDENTIAL_MODE"`
	// VaultPath Vault 路径：vault-secret 模式为 KV 路径，vault-dynamic 模式为角色名
	VaultPath string `json:"vault_path" yaml:"vault_path" toml:"vault_path" env:"VAULT_PATH"`
	// VaultUsernameField Vault 返回数据中的用户名字段名，默认 "username"
	VaultUsernameField string `json:"vault_username_field" yaml:"vault_username_field" toml:"vault_username_field" env:"VAULT_USERNAME_FIELD"`
	// VaultPasswordField Vault 返回数据中的密码字段名，默认 "password"
	VaultPasswordField string `json:"vault_password_field" yaml:"vault_password_field" toml:"vault_password_field" env:"VAULT_PASSWORD_FIELD"`
	// VaultAutoRenew 是否自动续期动态凭证，默认 true
	VaultAutoRenew bool `json:"vault_auto_renew" yaml:"vault_auto_renew" toml:"vault_auto_renew" env:"VAULT_AUTO_RENEW"`
	// VaultRenewThreshold 续期阈值（租约生命周期的百分比），默认 0.8 (80%)
	VaultRenewThreshold float64 `json:"vault_renew_threshold" yaml:"vault_renew_threshold" toml:"vault_renew_threshold" env:"VAULT_RENEW_THRESHOLD"`

	// GORM perform single create, update, delete operations in transactions by default to ensure database data integrity
	// You can disable it by setting `SkipDefaultTransaction` to true
	SkipDefaultTransaction bool `toml:"skip_default_transaction" json:"skip_default_transaction" yaml:"skip_default_transaction"  env:"SKIP_DEFALT_TRANSACTION"`
	// FullSaveAssociations full save associations
	FullSaveAssociations bool `toml:"full_save_associations" json:"full_save_associations" yaml:"full_save_associations"  env:"FULL_SAVE_ASSOCIATIONS"`
	// DryRun generate sql without execute
	DryRun bool `toml:"dry_run" json:"dry_run" yaml:"dry_run"  env:"DRY_RUN"`
	// PrepareStmt executes the given query in cached statement
	PrepareStmt bool `toml:"prepare_stmt" json:"prepare_stmt" yaml:"prepare_stmt"  env:"PREPARE_STMT"`
	// DisableAutomaticPing
	DisableAutomaticPing bool `toml:"disable_automatic_ping" json:"disable_automatic_ping" yaml:"disable_automatic_ping"  env:"DISABLE_AUTOMATIC_PING"`
	// DisableForeignKeyConstraintWhenMigrating
	DisableForeignKeyConstraintWhenMigrating bool `toml:"disable_foreign_key_constraint_when_migrating" json:"disable_foreign_key_constraint_when_migrating" yaml:"disable_foreign_key_constraint_when_migrating"  env:"DISABLE_FOREIGN_KEY_CONSTRAINT_WHEN_MIGRATING"`
	// IgnoreRelationshipsWhenMigrating
	IgnoreRelationshipsWhenMigrating bool `toml:"ignore_relationships_when_migrating" json:"ignore_relationships_when_migrating" yaml:"ignore_relationships_when_migrating"  env:"IGNORE_RELATIONSHIP_WHEN_MIGRATING"`
	// DisableNestedTransaction disable nested transaction
	DisableNestedTransaction bool `toml:"disable_nested_transaction" json:"disable_nested_transaction" yaml:"disable_nested_transaction"  env:"DISABLE_NESTED_TRANSACTION"`
	// AllowGlobalUpdate allow global update
	AllowGlobalUpdate bool `toml:"allow_global_update" json:"allow_global_update" yaml:"allow_global_update"  env:"ALL_GLOBAL_UPDATE"`
	// QueryFields executes the SQL query with all fields of the table
	QueryFields bool `toml:"query_fields" json:"query_fields" yaml:"query_fields"  env:"QUERY_FIELDS"`
	// CreateBatchSize default create batch size
	CreateBatchSize int `toml:"create_batch_size" json:"create_batch_size" yaml:"create_batch_size"  env:"CREATE_BATCH_SIZE"`
	// TranslateError enabling error translation
	TranslateError bool `toml:"translate_error" json:"translate_error" yaml:"translate_error"  env:"TRANSLATE_ERROR"`

	db  *gorm.DB
	log *zerolog.Logger

	// Vault 动态凭证内部状态
	leaseID       string        // vault-dynamic 模式的 lease ID
	leaseDuration int64         // 租约时长（秒）
	stopRenewal   chan struct{} // 停止续期信号
}

func (m *dataSource) Name() string {
	return AppName
}

func (i *dataSource) Priority() int {
	return 699
}

func (m *dataSource) Init() error {
	m.log = log.Sub(m.Name())

	// 初始化默认值
	if err := m.setDefaults(); err != nil {
		return err
	}

	// 根据凭证模式加载凭证
	if err := m.loadCredentials(); err != nil {
		return fmt.Errorf("load credentials: %w", err)
	}

	db, err := gorm.Open(m.Dialector(), &gorm.Config{
		SkipDefaultTransaction:                   m.SkipDefaultTransaction,
		FullSaveAssociations:                     m.FullSaveAssociations,
		DryRun:                                   m.DryRun,
		PrepareStmt:                              m.PrepareStmt,
		DisableAutomaticPing:                     m.DisableAutomaticPing,
		DisableForeignKeyConstraintWhenMigrating: m.DisableForeignKeyConstraintWhenMigrating,
		IgnoreRelationshipsWhenMigrating:         m.IgnoreRelationshipsWhenMigrating,
		DisableNestedTransaction:                 m.DisableNestedTransaction,
		AllowGlobalUpdate:                        m.AllowGlobalUpdate,
		Logger:                                   newGormCustomLogger(m.log),
	})
	if err != nil {
		return err
	}

	if trace.Get().Enable && m.Trace {
		m.log.Info().Msg("enable gorm trace")
		if err := db.Use(otelgorm.NewPlugin()); err != nil {
			return err
		}
	}

	if m.Debug {
		db = db.Debug()
	}

	m.db = db

	// 启动凭证续期（仅 vault-dynamic 模式）
	if m.CredentialMode.NeedsRenewal() && m.VaultAutoRenew {
		go m.startCredentialRenewal()
	}

	return nil
}

// 关闭数据库连接
func (m *dataSource) Close(ctx context.Context) {
	// 停止续期
	if m.stopRenewal != nil {
		close(m.stopRenewal)
	}

	// 撤销动态凭证租约
	if m.CredentialMode == CREDENTIAL_MODE_VAULT_DYNAMIC && m.leaseID != "" {
		vaultClient := vault.Client()
		if vaultClient != nil {
			if _, err := vaultClient.System.LeasesRevokeLease(ctx, schema.LeasesRevokeLeaseRequest{LeaseId: m.leaseID}); err != nil {
				m.log.Error().Err(err).Msgf("failed to revoke lease %s", m.leaseID)
			} else {
				m.log.Info().Msgf("revoked vault lease %s", m.leaseID)
			}
		}
	}

	// 关闭数据库连接
	if m.db == nil {
		return
	}

	d, err := m.db.DB()
	if err != nil {
		m.log.Error().Msgf("获取db error, %s", err)
		return
	}
	if err := d.Close(); err != nil {
		m.log.Error().Msgf("close db error, %s", err)
	}
}

// 从上下文中获取事物, 如果获取不到则直接返回 无事物的DB对象
func (m *dataSource) GetTransactionOrDB(ctx context.Context) *gorm.DB {
	db := GetTransactionFromCtx(ctx)
	if db != nil {
		return db.WithContext(ctx)
	}
	return m.db.WithContext(ctx)
}

func (m *dataSource) Dialector() gorm.Dialector {
	switch m.Provider {
	case PROVIDER_POSTGRES:
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
			m.Host,
			m.Username,
			m.Password,
			m.DB,
			m.Port,
		)
		return postgres.Open(dsn)
	case PROVIDER_SQLITE:
		return sqlite.Open(m.DB)
	default:
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			m.Username,
			m.Password,
			m.Host,
			m.Port,
			m.DB,
		)
		return mysql.Open(dsn)
	}
}

// setDefaults 设置默认值
func (m *dataSource) setDefaults() error {
	// Vault 字段默认值
	if m.VaultUsernameField == "" {
		m.VaultUsernameField = "username"
	}
	if m.VaultPasswordField == "" {
		m.VaultPasswordField = "password"
	}
	if m.VaultRenewThreshold == 0 {
		m.VaultRenewThreshold = 0.8
	}
	if m.VaultRenewThreshold < 0.5 || m.VaultRenewThreshold > 0.95 {
		return fmt.Errorf("vault_renew_threshold must be between 0.5 and 0.95, got %.2f", m.VaultRenewThreshold)
	}

	// 初始化续期停止信号
	if m.stopRenewal == nil {
		m.stopRenewal = make(chan struct{})
	}

	// 处理空字符串，兼容旧配置
	if m.CredentialMode == "" {
		m.CredentialMode = CREDENTIAL_MODE_STATIC
	}

	return nil
}

// loadCredentials 根据凭证模式加载凭证
func (m *dataSource) loadCredentials() error {
	switch m.CredentialMode {
	case CREDENTIAL_MODE_STATIC:
		return m.loadStaticCredentials()

	case CREDENTIAL_MODE_VAULT_SECRET:
		return m.loadVaultSecretCredentials()

	case CREDENTIAL_MODE_VAULT_DYNAMIC:
		return m.loadVaultDynamicCredentials()

	default:
		return fmt.Errorf("unsupported credential mode: %s", m.CredentialMode)
	}
}

// loadStaticCredentials 加载静态凭证
func (m *dataSource) loadStaticCredentials() error {
	if m.Username == "" {
		m.log.Warn().Msg("username is empty for static credential mode")
	}
	if m.Password == "" {
		m.log.Warn().Msg("password is empty for static credential mode")
	}
	m.log.Info().Msg("using static credentials from config file")
	return nil
}

// loadVaultSecretCredentials 从 Vault KV 加载静态凭证
func (m *dataSource) loadVaultSecretCredentials() error {
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

	// 提取 username 和 password
	username, ok := resp.Data.Data[m.VaultUsernameField].(string)
	if !ok {
		return fmt.Errorf("field '%s' not found in vault secret at %s", m.VaultUsernameField, m.VaultPath)
	}

	password, ok := resp.Data.Data[m.VaultPasswordField].(string)
	if !ok {
		return fmt.Errorf("field '%s' not found in vault secret at %s", m.VaultPasswordField, m.VaultPath)
	}

	m.Username = username
	m.Password = password

	m.log.Info().Msgf("loaded credentials from vault KV (path=%s)", m.VaultPath)
	return nil
}

// loadVaultDynamicCredentials 从 Vault Database 引擎生成动态凭证
func (m *dataSource) loadVaultDynamicCredentials() error {
	if m.VaultPath == "" {
		return fmt.Errorf("vault_path (role name) is required for vault-dynamic mode")
	}

	vaultClient := vault.Client()
	if vaultClient == nil {
		return fmt.Errorf("vault client not initialized, please ensure vault config is enabled")
	}

	ctx := context.Background()
	resp, err := vault.GenerateDatabaseCredentials(ctx, m.VaultPath)
	if err != nil {
		return fmt.Errorf("generate vault database credentials (role=%s): %w", m.VaultPath, err)
	}

	// 提取凭证
	username, ok := resp.Data["username"].(string)
	if !ok {
		return fmt.Errorf("username not found in vault database credentials response")
	}

	password, ok := resp.Data["password"].(string)
	if !ok {
		return fmt.Errorf("password not found in vault database credentials response")
	}

	m.Username = username
	m.Password = password

	// 提取租约信息
	m.leaseID = resp.LeaseID
	m.leaseDuration = int64(resp.LeaseDuration)

	m.log.Info().Msgf("generated vault dynamic credentials (role=%s, lease_id=%s, ttl=%ds)",
		m.VaultPath, m.leaseID, m.leaseDuration)

	return nil
}

// startCredentialRenewal 启动凭证自动续期（仅用于 vault-dynamic 模式）
func (m *dataSource) startCredentialRenewal() {
	if m.leaseDuration <= 0 {
		m.log.Warn().Msg("invalid lease duration, credential renewal disabled")
		return
	}

	// 在租约阈值时续期
	renewInterval := time.Duration(float64(m.leaseDuration)*m.VaultRenewThreshold) * time.Second

	m.log.Info().Msgf("credential auto-renewal enabled (interval=%s, threshold=%.0f%%)",
		renewInterval, m.VaultRenewThreshold*100)

	ticker := time.NewTicker(renewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.renewCredentials(); err != nil {
				m.log.Error().Err(err).Msg("failed to renew credentials")
				// 续期失败，尝试重新生成凭证
				if err := m.regenerateCredentials(); err != nil {
					m.log.Error().Err(err).Msg("failed to regenerate credentials - database connection may fail")
				}
			}

		case <-m.stopRenewal:
			m.log.Info().Msg("credential renewal stopped")
			return
		}
	}
}

// renewCredentials 续期现有凭证租约
func (m *dataSource) renewCredentials() error {
	ctx := context.Background()
	client := vault.Client()

	resp, err := client.System.LeasesRenewLease(ctx, schema.LeasesRenewLeaseRequest{
		LeaseId:   m.leaseID,
		Increment: fmt.Sprintf("%ds", m.leaseDuration),
	})
	if err != nil {
		return fmt.Errorf("renew lease %s: %w", m.leaseID, err)
	}

	m.leaseDuration = int64(resp.LeaseDuration)
	m.log.Info().Msgf("credentials renewed (lease_id=%s, new_ttl=%ds)", m.leaseID, m.leaseDuration)

	return nil
}

// regenerateCredentials 重新生成凭证并重连数据库
func (m *dataSource) regenerateCredentials() error {
	m.log.Warn().Msg("attempting to regenerate credentials")

	// 重新生成凭证
	if err := m.loadVaultDynamicCredentials(); err != nil {
		return fmt.Errorf("regenerate credentials: %w", err)
	}

	// 关闭旧连接
	if m.db != nil {
		if sqlDB, err := m.db.DB(); err == nil {
			sqlDB.Close()
		}
	}

	// 创建新连接
	db, err := gorm.Open(m.Dialector(), &gorm.Config{
		SkipDefaultTransaction:                   m.SkipDefaultTransaction,
		FullSaveAssociations:                     m.FullSaveAssociations,
		DryRun:                                   m.DryRun,
		PrepareStmt:                              m.PrepareStmt,
		DisableAutomaticPing:                     m.DisableAutomaticPing,
		DisableForeignKeyConstraintWhenMigrating: m.DisableForeignKeyConstraintWhenMigrating,
		IgnoreRelationshipsWhenMigrating:         m.IgnoreRelationshipsWhenMigrating,
		DisableNestedTransaction:                 m.DisableNestedTransaction,
		AllowGlobalUpdate:                        m.AllowGlobalUpdate,
		Logger:                                   newGormCustomLogger(m.log),
	})
	if err != nil {
		return fmt.Errorf("reconnect database with new credentials: %w", err)
	}

	if trace.Get().Enable && m.Trace {
		if err := db.Use(otelgorm.NewPlugin()); err != nil {
			return fmt.Errorf("enable trace on new connection: %w", err)
		}
	}

	if m.Debug {
		db = db.Debug()
	}

	m.db = db
	m.log.Info().Msg("database reconnected with regenerated credentials")

	return nil
}
