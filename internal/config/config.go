package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds small non-secret prefs. API keys always come from env vars, never stored here.
type Config struct {
	AIEnabled  bool
	AIProvider string
}

// Path returns the config file location under $XDG_CONFIG_HOME (or ~/.config).
func Path() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "idk", "config")
}

// Load reads the config file, tolerating a missing file (returns zero-value Config) or
// malformed lines (skipped).
func Load() Config {
	var cfg Config
	f, err := os.Open(Path())
	if err != nil {
		return cfg
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		switch key {
		case "ai_enabled":
			cfg.AIEnabled = value == "true"
		case "ai_provider":
			cfg.AIProvider = value
		}
	}
	return cfg
}

// Save writes the config file, creating its directory if needed.
func Save(cfg Config) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf("ai_enabled=%t\nai_provider=%s\n", cfg.AIEnabled, cfg.AIProvider)
	return os.WriteFile(path, []byte(content), 0o600)
}
