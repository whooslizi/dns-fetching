package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Provider       string `yaml:"provider"`
	CustomURL      string `yaml:"custom_url,omitempty"`
	Enabled        bool   `yaml:"enabled"`
	ListenAddr     string `yaml:"listen_addr"`
	CacheEnabled   bool   `yaml:"cache_enabled"`
	CacheMaxSize   int    `yaml:"cache_max_size"`
	AutoStart      bool   `yaml:"auto_start"`
	MinimizeToTray bool   `yaml:"minimize_to_tray"`
}

func DefaultConfig() *Config {
	return &Config{
		Provider:       "Cloudflare",
		Enabled:        false,
		ListenAddr:     "127.0.0.1:53",
		CacheEnabled:   true,
		CacheMaxSize:   10000,
		AutoStart:      false,
		MinimizeToTray: true,
	}
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "dns-fetching")
	return dir, os.MkdirAll(dir, 0700)
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			if saveErr := cfg.Save(); saveErr != nil {
				return nil, saveErr
			}
			return cfg, nil
		}
		return nil, err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	// write to tmp then rename to avoid partial writes
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func BackupDir() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	backupDir := filepath.Join(dir, "backup")
	return backupDir, os.MkdirAll(backupDir, 0700)
}
