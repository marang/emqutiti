package files

import (
	"os"
	"path/filepath"
)

// DataDir returns the base data directory for the given profile.
// If the profile is empty, "default" is used.
// The directory is placed under ~/.config/emqutiti/data by default.
// When EMQUTITI_HOME is set, that directory is used as the base instead.
func DataDir(profile string) string {
	if profile == "" {
		profile = "default"
	}
	if override := os.Getenv("EMQUTITI_HOME"); override != "" {
		return filepath.Join(override, "data", profile)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("data", profile)
	}
	return filepath.Join(home, ".config", "emqutiti", "data", profile)
}

// EnsureDir creates the directory with 0755 permissions if it does not exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
