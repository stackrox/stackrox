package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
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
	CentralConfigs CentralConfigs `yaml:"centrals"`
}

// GetCentralConfigs retrieves all central configs. In case RoxctlConfig is nil, nil will be returned.
func (r *RoxctlConfig) GetCentralConfigs() CentralConfigs {
	if r == nil {
		return nil
	}
	return r.CentralConfigs
}

// CentralURL is the URL of central.
type CentralURL = string

// CentralConfigs is the list of configurations per central.
type CentralConfigs map[CentralURL]*CentralConfig

// GetCentralConfig retrieves a CentralConfig for a given central URL. If no central config is specified,
// nil will be returned.
func (c CentralConfigs) GetCentralConfig(centralURL CentralURL) *CentralConfig {
	if c == nil {
		return nil
	}
	return c[centralURL]
}

// CentralConfig contains all configurations available for a single central. Currently, it only holds access information.
type CentralConfig struct {
	AccessConfig *CentralAccessConfig `yaml:"access,omitempty"`
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
	AccessToken  string     `yaml:"accessToken,omitempty"`
	IssuedAt     *time.Time `yaml:"issuedAt,omitempty"`
	ExpiresAt    *time.Time `yaml:"expiresAt,omitempty"`
	RefreshToken string     `yaml:"refreshToken,omitempty"`
}

// NewConfigStore initializes a config.Store that will be capable of reading and writing configuration to a configuration
// file.
func NewConfigStore() (Store, error) {
	path, err := determineConfigDir()
	if err != nil {
		return nil, err
	}
	path, err = ensureRoxctlConfigFilePathExists(path)
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
		return errors.Wrap(err, "marshalling config to YAML")
	}
	if err := os.WriteFile(c.path, rawConfig, 0600); err != nil {
		return errors.Wrapf(err, "writing config to file %s", c.path)
	}
	return nil
}

// determineConfigDir determines the configuration path of roxctl.
// This will be determined with the following priority:
// 1. ROX_CONFIG_DIR environment variable value.
// 2. XDG_RUNTIME_DIR environment variable value.
// 3. $HOME/.roxctl location.
func determineConfigDir() (string, error) {
	if path := env.ConfigDirEnv.Setting(); path != "" {
		fi, err := os.Stat(path)
		if err != nil {
			return "", errors.Wrap(err, "validating path for ROX_CONFIG_DIR")
		}
		if !fi.IsDir() {
			return "", errox.InvalidArgs.Newf("Path %s for ROX_CONFIG_DIR is a file, but should be a directory",
				path)
		}
		return path, nil
	}

	if path := os.Getenv("XDG_RUNTIME_DIR"); path != "" {
		fi, err := os.Stat(path)
		if err != nil {
			return "", errors.Wrap(err, "validating path for XDG_RUNTIME_DIR")
		}
		if !fi.IsDir() {
			return "", errox.InvalidArgs.Newf("Path %s for ROX_CONFIG_DIR is a file, but should be a directory",
				path)
		}
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "determining home directory")
	}
	path := filepath.Join(homeDir, ".roxctl")
	if err := os.MkdirAll(path, 0700); err != nil {
		return "", errors.Wrapf(err, "creating config directory %s", path)
	}
	return path, nil
}

// ensureRoxctlConfigFilePathExists will ensure that the file roxctl-config.yaml exists in the given path.
func ensureRoxctlConfigFilePathExists(configDir string) (string, error) {
	configFilePath := filepath.Join(configDir, "roxctl-config.yaml")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", errors.Wrapf(err, "creating roxctl config file %s", configDir)
	}
	return configFilePath, nil
}
