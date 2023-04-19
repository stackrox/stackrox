package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"gopkg.in/yaml.v3"
)

// Store provides the ability to read / write configurations for roxctl from / to a configuration file.
//
//go:generate mockgen-wrapper
type Store interface {
	Write(cfg *RoxctlConfig) error
	Read() (*RoxctlConfig, error)
}

// RoxctlConfig contains all configurations available for roxctl.
type RoxctlConfig struct {
	CentralConfigs CentralConfigs `json:"centrals"`
}

// GetCentralConfigs retrieves all central configs. In case RoxctlConfig is nil, nil will be returned.
func (r *RoxctlConfig) GetCentralConfigs() CentralConfigs {
	if r == nil {
		return nil
	}
	return r.CentralConfigs
}

// CentralConfigs is the list of configurations per central.
type CentralConfigs map[string]*CentralConfig

// GetCentralConfig retrieves a CentralConfig for a given host. If no central config is specified, nil will be returned.
func (c CentralConfigs) GetCentralConfig(host string) *CentralConfig {
	if c == nil {
		return nil
	}
	return c[host]
}

// CentralConfig contains all configurations available for a single central. Currently, it only holds access information.
type CentralConfig struct {
	AccessConfig *CentralAccessConfig `json:"access,omitempty"`
}

// GetAccess retrieves the access configuration for a central.
func (c *CentralConfig) GetAccess() *CentralAccessConfig {
	if c == nil {
		return nil
	}
	return c.AccessConfig
}

// CentralAccessConfig contains all configurations for access to a single central.
type CentralAccessConfig struct {
	AccessToken  string     `json:"accessToken,omitempty"`
	IssuedAt     *time.Time `json:"issuedAt,omitempty"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	RefreshToken string     `json:"refreshToken"`
}

// NewConfigStore initializes a config.Store that will be capable of reading and writing configuration to a configuration
// file.
func NewConfigStore() (Store, error) {
	path, err := determineConfigPath()
	if err != nil {
		return nil, err
	}
	store := &configStore{
		path: path,
	}
	return store, nil
}

type configStore struct {
	path string
}

func (c *configStore) Read() (*RoxctlConfig, error) {
	fileContents, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RoxctlConfig{CentralConfigs: map[string]*CentralConfig{}}, nil
		}
		return nil, errors.Wrapf(err, "reading config from file %s", c.path)
	}

	var cfg RoxctlConfig
	if err := yaml.Unmarshal(fileContents, &cfg); err != nil {
		return nil, errors.Wrap(err, "parsing config")
	}
	return &cfg, nil
}

func (c *configStore) Write(cfg *RoxctlConfig) error {
	rawConfig, err := yaml.Marshal(cfg)
	if err != nil {
		return errors.Wrap(err, "unmarshalling config to YAML")
	}
	if err := os.WriteFile(c.path, rawConfig, 0644); err != nil {
		return errors.Wrapf(err, "writing config to file %s", c.path)
	}
	return nil
}

// determineConfigPath determines the configuration path of roxctl.
// This will be determined with the following priority:
// 1. ROX_CONFIG_DIR environment variable value.
// 2. XDG_RUNTIME_DIR environment variable value.
// 3. $HOME/.roxctl/config location.
func determineConfigPath() (string, error) {
	if path := env.ConfigDirEnv.Setting(); path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", errors.Wrap(err, "validating path for ROX_CONFIG_DIR")
		}
		return path, nil
	}

	if path := os.Getenv("XDG_RUNTIME_DIR"); path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", errors.Wrap(err, "validating path for XDG_RUNTIME_DIR")
		}
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "determining home directory")
	}
	path := filepath.Join(homeDir, ".roxctl", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0775); err != nil {
		return "", errors.Wrapf(err, "creating config directory %s", path)
	}
	return path, nil
}
