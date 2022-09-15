package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sigs.k8s.io/yaml"
)

// Config is the config struct
type Config struct {
	Hosts map[string]*HostConfig `json:"hosts"`
}

// GetHosts gets hosts
func (c *Config) GetHosts() map[string]*HostConfig {
	if c == nil {
		return nil
	}
	return c.Hosts
}

// HostConfig is the host config
type HostConfig struct {
	Access *HostAccessConfig `json:"access,omitempty"`
}

// GetAccess gets access
func (c *HostConfig) GetAccess() *HostAccessConfig {
	if c == nil {
		return nil
	}
	return c.Access
}

// HostAccessConfig is
type HostAccessConfig struct {
	Token     string     `json:"token,omitempty"`
	IssuedAt  *time.Time `json:"issuedAt,omitempty"`
	ExpiresAt time.Time  `json:"expiresAt,omitempty"`

	RefreshToken string `json:"refreshToken,omitempty"`
}

// GetToken gets
func (c *HostAccessConfig) GetToken() string {
	if c == nil {
		return ""
	}
	return c.Token
}

func configPath() (string, error) {
	if path := os.Getenv("ROXCONFIG"); path != "" {
		return path, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(homeDir, ".roxctl", "config.yaml"), nil
}

// Load loads
func Load() (*Config, error) {
	cfgPath, err := configPath()
	if err != nil {
		return nil, fmt.Errorf("could not load config: %w", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("could not load config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}
	return &cfg, nil
}

// Store stores
func Store(cfg *Config) error {
	yamlBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not marshal config to YAML: %w", err)
	}
	cfgPath, err := configPath()
	if err != nil {
		return fmt.Errorf("could not store config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		return fmt.Errorf("could not ensure directory for config file %s exists", cfgPath)
	}
	if _, err := os.Stat(cfgPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not stat config file %s: %w", cfgPath, err)
	} else if err == nil {
		if err := os.Chmod(cfgPath, 0600); err != nil {
			return fmt.Errorf("could not ensure mode for config file %s: %w", cfgPath, err)
		}
	}
	if err := os.WriteFile(cfgPath, yamlBytes, 0600); err != nil {
		return fmt.Errorf("could not write config to file %s: %w", cfgPath, err)
	}
	return nil
}
