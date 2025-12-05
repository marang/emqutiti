package connections

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"

	"github.com/marang/emqutiti/internal/files"
)

// saveConfig persists profiles and default selection to config.toml.
func saveConfig(profiles []Profile, defaultName string) error {
	saved := LoadState()
	cfg := userConfig{
		DefaultProfileName: defaultName,
		Profiles:           profiles,
		Saved:              saved,
		ProxyAddr:          LoadProxyAddr(),
	}
	return writeConfig(cfg)
}

// savePasswordToKeyring stores a password in the system keyring.
func savePasswordToKeyring(service, username, password string) error {
	return keyring.Set("emqutiti-"+service, username, password)
}

func deletePasswordFromKeyring(service, username string) error {
	if strings.TrimSpace(service) == "" {
		return nil
	}
	if err := keyring.Delete("emqutiti-"+service, username); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	return nil
}

// deleteProfileData removes profile-specific persisted history and traces and
// returns any cleanup errors.
func deleteProfileData(name string) error {
	historyPath := filepath.Join(files.DataDir(name), "history")
	historyErr := os.RemoveAll(historyPath)
	if historyErr != nil {
		log.Printf("Error removing %s: %v", historyPath, historyErr)
	}

	tracesPath := filepath.Join(files.DataDir(name), "traces")
	tracesErr := os.RemoveAll(tracesPath)
	if tracesErr != nil {
		log.Printf("Error removing %s: %v", tracesPath, tracesErr)
	}

	return errors.Join(historyErr, tracesErr)
}

// persistProfileChange applies a profile update, saves config and keyring.
func persistProfileChange(profiles *[]Profile, defaultName string, p Profile, idx int) error {
	plain := p.Password
	hasPassword := strings.TrimSpace(plain) != ""
	canUseKeyring := !p.FromEnv && hasPassword && strings.TrimSpace(p.Name) != "" && strings.TrimSpace(p.Username) != ""
	if canUseKeyring {
		p.Password = "keyring:emqutiti-" + p.Name + "/" + p.Username
	} else {
		if p.FromEnv {
			p.Password = ""
		} else {
			p.Password = plain
		}
	}
	if idx >= 0 && idx < len(*profiles) {
		(*profiles)[idx] = p
	} else {
		*profiles = append(*profiles, p)
	}
	if err := saveConfig(*profiles, defaultName); err != nil {
		return err
	}
	if canUseKeyring {
		if err := savePasswordToKeyring(p.Name, p.Username, plain); err != nil {
			return err
		}
	}
	return nil
}
