package connections

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/zalando/go-keyring"

	"github.com/marang/emqutiti/internal/files"
)

func TestProfileBrokerURL(t *testing.T) {
	for _, schema := range []string{"mqtt", "mqtts"} {
		p := Profile{Schema: schema, Host: "example.com", Port: 1883}
		want := schema + "://example.com:1883"
		if got := p.BrokerURL(); got != want {
			t.Fatalf("BrokerURL() = %q", got)
		}
	}
}

// TestApplyEnvVars ensures environment variables override profile fields.
func TestApplyEnvVars(t *testing.T) {
	p := Profile{Name: "test", FromEnv: true}
	prefix := EnvPrefix(p.Name)
	os.Setenv(prefix+"HOST", "example.com")
	os.Setenv(prefix+"PORT", "1884")
	os.Setenv(prefix+"SSL_TLS", "true")
	t.Cleanup(func() {
		os.Unsetenv(prefix + "HOST")
		os.Unsetenv(prefix + "PORT")
		os.Unsetenv(prefix + "SSL_TLS")
	})

	ApplyEnvVars(&p)

	if p.Host != "example.com" {
		t.Errorf("Host not set: %v", p.Host)
	}
	if p.Port != 1884 {
		t.Errorf("Port not set: %v", p.Port)
	}
	if !p.SSL {
		t.Errorf("SSL not set: %v", p.SSL)
	}
}

func TestLoadConfig(t *testing.T) {
	keyring.MockInit()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	data := `[[profiles]]
name = "p1"
username = "u1"
password = "keyring:svc/u1"

[[profiles]]
name = "env"
from_env = true
`
	if err := os.WriteFile(cfgPath, []byte(data), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := keyring.Set("svc", "u1", "secret"); err != nil {
		t.Fatalf("keyring set: %v", err)
	}
	os.Setenv("EMQUTITI_ENV_HOST", "example.com")
	os.Setenv("EMQUTITI_ENV_PORT", "1884")
	t.Cleanup(func() {
		os.Unsetenv("EMQUTITI_ENV_HOST")
		os.Unsetenv("EMQUTITI_ENV_PORT")
	})

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if len(cfg.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(cfg.Profiles))
	}
	if cfg.Profiles[0].Password != "secret" {
		t.Errorf("keyring password not resolved: %v", cfg.Profiles[0].Password)
	}
	p := cfg.Profiles[1]
	if p.Host != "example.com" || p.Port != 1884 {
		t.Errorf("env vars not applied: %+v", p)
	}
}

func TestLoadConfigKeyringError(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	data := `[[profiles]]
name = "p1"
username = "u1"
password = "keyring:svc/u1"
`
	if err := os.WriteFile(cfgPath, []byte(data), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := LoadConfig(cfgPath); err == nil {
		t.Fatal("expected error for missing keyring entry")
	}
}

func TestSavePasswordToKeyring(t *testing.T) {
	keyring.MockInit()
	if err := savePasswordToKeyring("svc", "user", "pw"); err != nil {
		t.Fatalf("savePasswordToKeyring: %v", err)
	}
	got, err := keyring.Get("emqutiti-svc", "user")
	if err != nil || got != "pw" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestDeletePasswordFromKeyring(t *testing.T) {
	keyring.MockInit()
	if err := savePasswordToKeyring("svc", "user", "pw"); err != nil {
		t.Fatalf("savePasswordToKeyring: %v", err)
	}
	if err := deletePasswordFromKeyring("svc", "user"); err != nil {
		t.Fatalf("deletePasswordFromKeyring: %v", err)
	}
	if _, err := keyring.Get("emqutiti-svc", "user"); err == nil || !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("expected missing keyring entry, got %v", err)
	}
}

func TestSaveConfig(t *testing.T) {
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	profiles := []Profile{{Name: "a"}}
	if err := saveConfig(profiles, "a"); err != nil {
		t.Fatalf("saveConfig: %v", err)
	}
	cfgPath, _ := DefaultUserConfigFile()
	var cfg userConfig
	if _, err := toml.DecodeFile(cfgPath, &cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.DefaultProfileName != "a" || len(cfg.Profiles) != 1 || cfg.Profiles[0].Name != "a" {
		t.Fatalf("unexpected cfg: %#v", cfg)
	}
}

func TestDeleteProfileData(t *testing.T) {
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	base := files.DataDir("p1")
	os.MkdirAll(filepath.Join(base, "history"), 0755)
	os.MkdirAll(filepath.Join(base, "traces"), 0755)

	deleteProfileData("p1")

	if _, err := os.Stat(filepath.Join(base, "history")); !os.IsNotExist(err) {
		t.Fatalf("history not deleted")
	}
	if _, err := os.Stat(filepath.Join(base, "traces")); !os.IsNotExist(err) {
		t.Fatalf("traces not deleted")
	}
}

func TestPersistProfileChange(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	profiles := []Profile{}
	p := Profile{Name: "test", Username: "user", Password: "secret"}
	if err := persistProfileChange(&profiles, "test", p, -1); err != nil {
		t.Fatalf("persistProfileChange: %v", err)
	}

	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	if profiles[0].Password != "keyring:emqutiti-test/user" {
		t.Fatalf("password not rewritten: %v", profiles[0].Password)
	}

	cfgPath, _ := DefaultUserConfigFile()
	var cfg userConfig
	if _, err := toml.DecodeFile(cfgPath, &cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(cfg.Profiles) != 1 || cfg.Profiles[0].Password != "keyring:emqutiti-test/user" {
		t.Fatalf("config not written: %#v", cfg)
	}
	if cfg.DefaultProfileName != "test" {
		t.Fatalf("default not saved: %#v", cfg)
	}
	pw, err := keyring.Get("emqutiti-test", "user")
	if err != nil || pw != "secret" {
		t.Fatalf("keyring not saved: %q %v", pw, err)
	}
}

func TestPersistProfileChangeWithoutPassword(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	profiles := []Profile{}
	p := Profile{Name: "test", Username: "user"}
	if err := persistProfileChange(&profiles, "test", p, -1); err != nil {
		t.Fatalf("persistProfileChange: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	if profiles[0].Password != "" {
		t.Fatalf("expected empty password, got %q", profiles[0].Password)
	}

	cfgPath, _ := DefaultUserConfigFile()
	var cfg userConfig
	if _, err := toml.DecodeFile(cfgPath, &cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.Profiles[0].Password != "" {
		t.Fatalf("expected empty password in config, got %q", cfg.Profiles[0].Password)
	}
	if _, err := keyring.Get("emqutiti-test", "user"); err == nil {
		t.Fatalf("expected no keyring entry when password is empty")
	}
}

func TestPersistProfileChangeWriteError(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, ".config")
	if err := os.WriteFile(cfgFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	profiles := []Profile{}
	p := Profile{Name: "test", Username: "user", Password: "secret"}
	if err := persistProfileChange(&profiles, "test", p, -1); err == nil {
		t.Fatal("expected error")
	}
}

func TestPersistProfileChangeSkipsKeyringWithoutUsername(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	profiles := []Profile{}
	p := Profile{Name: "test", Password: "secret"}
	if err := persistProfileChange(&profiles, "test", p, -1); err != nil {
		t.Fatalf("persistProfileChange: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	if profiles[0].Password != "secret" {
		t.Fatalf("expected plain password to persist when username empty, got %q", profiles[0].Password)
	}
	if _, err := keyring.Get("emqutiti-test", ""); err == nil {
		t.Fatalf("expected no keyring entry for empty username")
	}
	cfgPath, _ := DefaultUserConfigFile()
	var cfg userConfig
	if _, err := toml.DecodeFile(cfgPath, &cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.Profiles[0].Password != "secret" {
		t.Fatalf("expected plain password in config, got %q", cfg.Profiles[0].Password)
	}
}

func TestDeleteConnectionRemovesPassword(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	if err := savePasswordToKeyring("p1", "u1", "secret"); err != nil {
		t.Fatalf("savePasswordToKeyring: %v", err)
	}

	mgr := NewConnectionsModel()
	mgr.Profiles = []Profile{{Name: "p1", Username: "u1", Password: "secret"}}
	mgr.Statuses = map[string]string{}
	mgr.Errors = map[string]string{}
	mgr.DeleteConnection(0)

	if len(mgr.Profiles) != 0 {
		t.Fatalf("expected profile removed, got %d", len(mgr.Profiles))
	}
	if _, err := keyring.Get("emqutiti-p1", "u1"); err == nil {
		t.Fatalf("expected password removed from keyring")
	}
}

func TestDeleteConnectionClearsDefault(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	mgr := NewConnectionsModel()
	mgr.Profiles = []Profile{{Name: "p1"}, {Name: "p2"}}
	mgr.DefaultProfileName = "p1"
	mgr.Statuses = map[string]string{}
	mgr.Errors = map[string]string{}

	mgr.DeleteConnection(0)

	if mgr.DefaultProfileName != "p2" {
		t.Fatalf("expected default to move to remaining profile, got %q", mgr.DefaultProfileName)
	}
	cfgPath, _ := DefaultUserConfigFile()
	var cfg userConfig
	if _, err := toml.DecodeFile(cfgPath, &cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.DefaultProfileName != "p2" {
		t.Fatalf("expected config default p2, got %q", cfg.DefaultProfileName)
	}
}

func TestApplyDefaultPassword(t *testing.T) {
	t.Setenv("EMQUTITI_DEFAULT_PASSWORD", "envpw")

	t.Run("sets password when empty", func(t *testing.T) {
		p := Profile{}
		ApplyDefaultPassword(&p)
		if p.Password != "envpw" {
			t.Fatalf("expected envpw, got %q", p.Password)
		}
	})

	t.Run("does not override existing password", func(t *testing.T) {
		p := Profile{Password: "orig"}
		ApplyDefaultPassword(&p)
		if p.Password != "orig" {
			t.Fatalf("expected orig, got %q", p.Password)
		}
	})

	t.Run("ignores env when from env", func(t *testing.T) {
		p := Profile{FromEnv: true}
		ApplyDefaultPassword(&p)
		if p.Password != "" {
			t.Fatalf("expected empty password, got %q", p.Password)
		}
	})
}
