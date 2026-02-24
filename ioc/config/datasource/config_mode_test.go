package datasource_test

import (
	"os"
	"testing"

	"github.com/infraboard/mcube/v2/ioc"
	"github.com/infraboard/mcube/v2/ioc/config/datasource"
)

// ===================== 环境变量模式测试 =====================

// TestEnvStaticConfig 测试通过环境变量加载静态凭证配置
func TestEnvStaticConfig(t *testing.T) {
	ds := datasource.Get()
	if ds == nil {
		t.Fatal("datasource config should not be nil")
	}

	// init() 中已通过环境变量设置了这些值
	tests := []struct {
		name   string
		got    any
		expect any
	}{
		{"host", ds.Host, "127.0.0.1"},
		{"port", ds.Port, 3306},
		{"database", ds.DB, "test"},
		{"username", ds.Username, "root"},
		{"password", ds.Password, "123456"},
		{"debug", ds.Debug, true},
	}
	for _, tt := range tests {
		if tt.got != tt.expect {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.expect)
		}
	}
}

// TestEnvCredentialModeDefault 测试未显式设置 credential_mode 时默认为 static
func TestEnvCredentialModeDefault(t *testing.T) {
	ds := datasource.Get()
	if ds.CredentialMode != datasource.CREDENTIAL_MODE_STATIC {
		t.Errorf("credential_mode: got %q, want %q", ds.CredentialMode, datasource.CREDENTIAL_MODE_STATIC)
	}
}

// TestEnvProviderDefault 测试通过环境变量加载时的默认 provider
func TestEnvProviderDefault(t *testing.T) {
	ds := datasource.Get()
	// 默认 provider 为 mysql（init 中未设置 DATASOURCE_PROVIDER）
	if ds.Provider != datasource.PROVIDER_MYSQL {
		t.Errorf("provider: got %q, want %q", ds.Provider, datasource.PROVIDER_MYSQL)
	}
}

// ===================== TOML 文件模式测试 =====================

// loadFromToml 辅助函数：通过 TOML 文件重新加载配置（仅加载配置，不执行 Init）
func loadFromToml(t *testing.T, path string) {
	t.Helper()
	// 清除已有环境变量，避免干扰
	envKeys := []string{
		"DATASOURCE_HOST", "DATASOURCE_PORT", "DATASOURCE_DB",
		"DATASOURCE_USERNAME", "DATASOURCE_PASSWORD", "DATASOURCE_DEBUG",
		"DATASOURCE_PROVIDER", "DATASOURCE_CREDENTIAL_MODE",
		"DATASOURCE_VAULT_PATH", "DATASOURCE_VAULT_USERNAME_FIELD",
		"DATASOURCE_VAULT_PASSWORD_FIELD", "DATASOURCE_VAULT_AUTO_RENEW",
		"DATASOURCE_VAULT_RENEW_THRESHOLD", "DATASOURCE_AUTO_MIGRATE",
		"DATASOURCE_TRACE",
	}
	for _, k := range envKeys {
		os.Unsetenv(k)
	}

	req := ioc.NewLoadConfigRequest()
	req.ForceLoad = true
	req.ConfigEnv.Enabled = false
	req.ConfigFile.Enabled = true
	req.ConfigFile.Paths = []string{path}
	if err := ioc.DefaultStore.LoadConfig(req); err != nil {
		t.Fatalf("load config from %s: %v", path, err)
	}
}

// TestTomlMySQLConfig 测试从 TOML 文件加载 MySQL 配置
func TestTomlMySQLConfig(t *testing.T) {
	loadFromToml(t, "test/mysql.toml")
	ds := datasource.Get()

	tests := []struct {
		name   string
		got    any
		expect any
	}{
		{"provider", ds.Provider, datasource.PROVIDER_MYSQL},
		{"host", ds.Host, "10.0.0.1"},
		{"port", ds.Port, 3306},
		{"database", ds.DB, "mydb"},
		{"username", ds.Username, "toml_user"},
		{"password", ds.Password, "toml_pass"},
		{"auto_migrate", ds.AutoMigrate, true},
		{"debug", ds.Debug, true},
		{"trace", ds.Trace, false},
		{"credential_mode", ds.CredentialMode, datasource.CREDENTIAL_MODE_STATIC},
		{"skip_default_transaction", ds.SkipDefaultTransaction, true},
		{"prepare_stmt", ds.PrepareStmt, false},
	}
	for _, tt := range tests {
		if tt.got != tt.expect {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.expect)
		}
	}
}

// TestTomlPostgresConfig 测试从 TOML 文件加载 PostgreSQL 配置
func TestTomlPostgresConfig(t *testing.T) {
	loadFromToml(t, "test/postgres.toml")
	ds := datasource.Get()

	tests := []struct {
		name   string
		got    any
		expect any
	}{
		{"provider", ds.Provider, datasource.PROVIDER_POSTGRES},
		{"host", ds.Host, "10.0.0.2"},
		{"port", ds.Port, 5432},
		{"database", ds.DB, "pgdb"},
		{"username", ds.Username, "pg_user"},
		{"password", ds.Password, "pg_pass"},
		{"debug", ds.Debug, false},
		{"trace", ds.Trace, true},
	}
	for _, tt := range tests {
		if tt.got != tt.expect {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.expect)
		}
	}
}

// TestTomlSQLiteConfig 测试从 TOML 文件加载 SQLite 配置
func TestTomlSQLiteConfig(t *testing.T) {
	loadFromToml(t, "test/sqlite.toml")
	ds := datasource.Get()

	tests := []struct {
		name   string
		got    any
		expect any
	}{
		{"provider", ds.Provider, datasource.PROVIDER_SQLITE},
		{"database", ds.DB, "test.db"},
		{"debug", ds.Debug, true},
		{"trace", ds.Trace, false},
	}
	for _, tt := range tests {
		if tt.got != tt.expect {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.expect)
		}
	}
}

// ===================== Vault 模式配置测试 =====================

// TestTomlVaultSecretConfig 测试从 TOML 文件加载 vault-secret 模式配置
func TestTomlVaultSecretConfig(t *testing.T) {
	loadFromToml(t, "test/vault_secret.toml")
	ds := datasource.Get()

	tests := []struct {
		name   string
		got    any
		expect any
	}{
		{"provider", ds.Provider, datasource.PROVIDER_MYSQL},
		{"host", ds.Host, "10.0.0.3"},
		{"credential_mode", ds.CredentialMode, datasource.CREDENTIAL_MODE_VAULT_SECRET},
		{"vault_path", ds.VaultPath, "secret/data/myapp/db"},
		{"vault_username_field", ds.VaultUsernameField, "db_user"},
		{"vault_password_field", ds.VaultPasswordField, "db_pass"},
	}
	for _, tt := range tests {
		if tt.got != tt.expect {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.expect)
		}
	}

	// vault-secret 模式下 IsVaultMode 应返回 true, NeedsRenewal 应返回 false
	if !ds.CredentialMode.IsVaultMode() {
		t.Error("vault-secret should be vault mode")
	}
	if ds.CredentialMode.NeedsRenewal() {
		t.Error("vault-secret should not need renewal")
	}
}

// TestTomlVaultDynamicConfig 测试从 TOML 文件加载 vault-dynamic 模式配置
func TestTomlVaultDynamicConfig(t *testing.T) {
	loadFromToml(t, "test/vault_dynamic.toml")
	ds := datasource.Get()

	tests := []struct {
		name   string
		got    any
		expect any
	}{
		{"provider", ds.Provider, datasource.PROVIDER_POSTGRES},
		{"host", ds.Host, "10.0.0.4"},
		{"port", ds.Port, 5432},
		{"credential_mode", ds.CredentialMode, datasource.CREDENTIAL_MODE_VAULT_DYNAMIC},
		{"vault_path", ds.VaultPath, "my-role"},
		{"vault_auto_renew", ds.VaultAutoRenew, true},
		{"vault_renew_threshold", ds.VaultRenewThreshold, 0.75},
	}
	for _, tt := range tests {
		if tt.got != tt.expect {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.expect)
		}
	}

	// vault-dynamic 模式下 IsVaultMode 和 NeedsRenewal 都应返回 true
	if !ds.CredentialMode.IsVaultMode() {
		t.Error("vault-dynamic should be vault mode")
	}
	if !ds.CredentialMode.NeedsRenewal() {
		t.Error("vault-dynamic should need renewal")
	}
}

// ===================== 环境变量覆盖 TOML 文件测试 =====================

// TestEnvOverridesToml 测试环境变量覆盖 TOML 文件配置
func TestEnvOverridesToml(t *testing.T) {
	// 先设置环境变量
	os.Setenv("DATASOURCE_HOST", "env-host")
	os.Setenv("DATASOURCE_USERNAME", "env_user")
	os.Setenv("DATASOURCE_PASSWORD", "env_pass")
	defer func() {
		os.Unsetenv("DATASOURCE_HOST")
		os.Unsetenv("DATASOURCE_USERNAME")
		os.Unsetenv("DATASOURCE_PASSWORD")
	}()

	// 同时启用文件和环境变量加载
	req := ioc.NewLoadConfigRequest()
	req.ForceLoad = true
	req.ConfigEnv.Enabled = true
	req.ConfigFile.Enabled = true
	req.ConfigFile.Paths = []string{"test/mysql.toml"}
	if err := ioc.DefaultStore.LoadConfig(req); err != nil {
		t.Fatalf("load config: %v", err)
	}

	ds := datasource.Get()

	// 环境变量应覆盖 TOML 文件中的值
	if ds.Host != "env-host" {
		t.Errorf("host should be overridden by env: got %q, want %q", ds.Host, "env-host")
	}
	if ds.Username != "env_user" {
		t.Errorf("username should be overridden by env: got %q, want %q", ds.Username, "env_user")
	}
	if ds.Password != "env_pass" {
		t.Errorf("password should be overridden by env: got %q, want %q", ds.Password, "env_pass")
	}

	// 未被环境变量覆盖的值应保持 TOML 文件中的值
	if ds.Port != 3306 {
		t.Errorf("port should retain toml value: got %d, want %d", ds.Port, 3306)
	}
	if ds.DB != "mydb" {
		t.Errorf("database should retain toml value: got %q, want %q", ds.DB, "mydb")
	}
}

// ===================== CREDENTIAL_MODE 枚举方法测试 =====================

// TestCredentialModeFlags 测试 CREDENTIAL_MODE 上的辅助方法
func TestCredentialModeFlags(t *testing.T) {
	tests := []struct {
		mode         datasource.CREDENTIAL_MODE
		isVault      bool
		needsRenewal bool
	}{
		{datasource.CREDENTIAL_MODE_STATIC, false, false},
		{datasource.CREDENTIAL_MODE_VAULT_SECRET, true, false},
		{datasource.CREDENTIAL_MODE_VAULT_DYNAMIC, true, true},
	}
	for _, tt := range tests {
		if got := tt.mode.IsVaultMode(); got != tt.isVault {
			t.Errorf("CREDENTIAL_MODE(%q).IsVaultMode(): got %v, want %v", tt.mode, got, tt.isVault)
		}
		if got := tt.mode.NeedsRenewal(); got != tt.needsRenewal {
			t.Errorf("CREDENTIAL_MODE(%q).NeedsRenewal(): got %v, want %v", tt.mode, got, tt.needsRenewal)
		}
	}
}

// ===================== Provider 枚举测试 =====================

// TestProviderValues 测试 PROVIDER 常量值正确
func TestProviderValues(t *testing.T) {
	tests := []struct {
		provider datasource.PROVIDER
		expect   string
	}{
		{datasource.PROVIDER_MYSQL, "mysql"},
		{datasource.PROVIDER_POSTGRES, "postgres"},
		{datasource.PROVIDER_SQLITE, "sqlite"},
	}
	for _, tt := range tests {
		if string(tt.provider) != tt.expect {
			t.Errorf("PROVIDER: got %q, want %q", tt.provider, tt.expect)
		}
	}
}
