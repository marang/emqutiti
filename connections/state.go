package connections

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ConnectionSnapshot holds topics, payloads, and layout heights for a
// connection in config.toml.
type ConnectionSnapshot struct {
	Topics        []TopicSnapshot   `toml:"topics"`
	Payloads      []PayloadSnapshot `toml:"payloads"`
	MessageHeight int               `toml:"message_height,omitempty"`
	TopicsHeight  int               `toml:"topics_height,omitempty"`
	HistoryHeight int               `toml:"history_height,omitempty"`
}

// userConfig represents the structure stored in config.toml.
type userConfig struct {
	DefaultProfileName string                        `toml:"default_profile"`
	Profiles           []Profile                     `toml:"profiles"`
	Saved              map[string]ConnectionSnapshot `toml:"saved"`
	ProxyAddr          string                        `toml:"proxy_addr"`
}

// LoadState retrieves saved topics and payloads from config.toml.
func LoadState() map[string]ConnectionSnapshot {
	fp, err := DefaultUserConfigFile()
	if err != nil {
		return map[string]ConnectionSnapshot{}
	}
	var cfg userConfig
	if _, err := toml.DecodeFile(fp, &cfg); err != nil {
		return map[string]ConnectionSnapshot{}
	}
	if cfg.Saved == nil {
		return map[string]ConnectionSnapshot{}
	}
	return cfg.Saved
}

// writeConfig writes the entire configuration back to disk.
func writeConfig(cfg userConfig) error {
	fp, err := DefaultUserConfigFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fp), os.ModePerm); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return err
	}
	if err := os.WriteFile(fp, buf.Bytes(), 0644); err != nil {
		return err
	}
	return nil
}

// SaveState updates only the Saved section in config.toml.
func SaveState(data map[string]ConnectionSnapshot) error {
	fp, err := DefaultUserConfigFile()
	if err != nil {
		return err
	}
	var cfg userConfig
	toml.DecodeFile(fp, &cfg) // ignore errors for new files
	cfg.Saved = data
	return writeConfig(cfg)
}
