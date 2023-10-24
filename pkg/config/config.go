package config

import (
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	configPath   = "/etc/stackrox/central-config.yaml"
	dbConfigPath = "/etc/ext-db/central-external-db.yaml"
)

var (
	defaultBucketFillFraction = 0.5
	defaultCompactionState    = true
	defaultDBSource           = "host=central-db.stackrox port=5432 user=postgres sslmode=verify-full statement_timeout=600000 pool_min_conns=1 pool_max_conns=90 client_encoding=UTF8"
	defaultDatabase           = "central_active"

	once   sync.Once
	config *Config

	log = logging.CreateLogger(logging.CurrentModule(), 0)
)

// Compaction defines the compaction configuration
type Compaction struct {
	Enabled               *bool    `yaml:"enabled"`
	BucketFillFraction    *float64 `yaml:"bucketFillFraction"`
	FreeFractionThreshold *float64 `yaml:"freeFractionThreshold"`
}

func (c *Compaction) applyDefaults() {
	if c.BucketFillFraction == nil {
		c.BucketFillFraction = &defaultBucketFillFraction
	}
	if c.Enabled == nil {
		c.Enabled = &defaultCompactionState
	}
}

// validate must be called after apply defaults
func (c *Compaction) validate() error {
	errorList := errorhelpers.NewErrorList("validating compaction")
	if *c.BucketFillFraction <= 0 || *c.BucketFillFraction > 1.0 {
		errorList.AddString("fill fraction must be greater than 0 and less than or equal to 1")
	}
	if c.FreeFractionThreshold != nil && (*c.FreeFractionThreshold <= 0 || *c.FreeFractionThreshold > 1.0) {
		errorList.AddString("compaction threshold fraction must be greater than 0 and less than or equal to 1")
	}
	return errorList.ToError()
}

// Maintenance defines the maintenance functions to use when Central starts
type Maintenance struct {
	SafeMode   bool       `yaml:"safeMode"`
	Compaction Compaction `yaml:"compaction"`
	// Allowing force rollback only to this version.
	ForceRollbackVersion string `yaml:"forceRollbackVersion"`
	PolicyEvaluators     string `yaml:"policyEvaluators"`
}

func (m *Maintenance) applyDefaults() {
	m.Compaction.applyDefaults()
}

func (m *Maintenance) validate() error {
	errorList := errorhelpers.NewErrorList("validating maintenance")
	if err := m.Compaction.validate(); err != nil {
		errorList.AddError(err)
	}
	return errorList.ToError()
}

// CentralDB defines the config options to access central-db
type CentralDB struct {
	Source       string `yaml:"source"`
	External     bool   `yaml:"external"`
	DatabaseName string
}

func (c *CentralDB) applyDefaults() {
	c.Source = strings.TrimSpace(c.Source)
	if c.Source == "" {
		c.Source = defaultDBSource
	}

	c.DatabaseName = strings.TrimSpace(c.DatabaseName)
	if c.DatabaseName == "" {
		c.DatabaseName = defaultDatabase
	}
}

func (c *CentralDB) validate() error {
	if c.DatabaseName == "" {
		return errors.New("unexpected empty database name")
	}
	if c.Source == "" {
		return errors.New("unexpected empty database source")
	}
	return nil
}

// Config defines all the configuration for Central
type Config struct {
	Maintenance Maintenance
	CentralDB   CentralDB
}

type yamlConfig interface {
	centralConfig | externalDBConfig
}

// Central Config
type centralConfig struct {
	Maintenance Maintenance `yaml:"maintenance"`
}

// External DB Config
type externalDBConfig struct {
	CentralDB CentralDB `yaml:"centralDB"`
}

func (c *Config) applyDefaults() {
	c.Maintenance.applyDefaults()
	c.CentralDB.applyDefaults()
}

func (c *Config) validate() error {
	errorList := errorhelpers.NewErrorList("validating config")
	if err := c.Maintenance.validate(); err != nil {
		errorList.AddError(err)
	}
	if err := c.CentralDB.validate(); err != nil {
		errorList.AddError(err)
	}
	return errorList.ToError()
}

// readConfig reads a configuration file
func readConfig[T yamlConfig](path string) (*T, error) {
	var conf T
	bytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &conf, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(bytes, &conf); err != nil {
		return nil, err
	}

	return &conf, nil
}

func readConfigs() (*Config, error) {
	centralConf, err := readConfig[centralConfig](configPath)
	if err != nil {
		return nil, err
	}
	dbConf, err := readConfig[externalDBConfig](dbConfigPath)
	if err != nil {
		return nil, err
	}

	conf := Config{Maintenance: centralConf.Maintenance, CentralDB: dbConf.CentralDB}
	conf.applyDefaults()
	if err := conf.validate(); err != nil {
		return nil, err
	}
	return &conf, nil
}

// GetConfig gets the Central configuration file
func GetConfig() *Config {
	once.Do(func() {
		var err error
		config, err = readConfigs()
		if err != nil {
			config = nil
			log.Errorf("Error reading config file: %v", err)
		}
	})
	return config
}
