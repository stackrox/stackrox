package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/stackrox/rox/pkg/fixtures/vmindexreport"
	"gopkg.in/yaml.v3"
)

type config struct {
	vmCount  int
	duration time.Duration

	numPackages     int
	numRepositories int

	port           uint
	metricsPort    int
	statsInterval  time.Duration
	requestTimeout time.Duration
	reportInterval time.Duration
}

// yamlConfig represents the structure of the YAML config file.
type yamlConfig struct {
	Loadgen struct {
		VmCount         int    `yaml:"vmCount"`
		NumPackages     int    `yaml:"numPackages"`
		NumRepositories int    `yaml:"numRepositories"`
		StatsInterval   string `yaml:"statsInterval"`
		Port            uint   `yaml:"port"`
		MetricsPort     int    `yaml:"metricsPort"`
		RequestTimeout  string `yaml:"requestTimeout,omitempty"`
		ReportInterval  string `yaml:"reportInterval,omitempty"`
	} `yaml:"loadgen"`
}

func parseConfig() config {
	var configFile string
	flag.StringVar(&configFile, "config", "", "Path to YAML config file (required)")
	flag.Parse()

	if configFile == "" {
		log.Error("--config flag is required")
		os.Exit(1)
	}

	yamlCfg, err := loadYAMLConfig(configFile)
	if err != nil {
		log.Errorf("loading config file: %v", err)
		os.Exit(1)
	}

	cfg := configFromYAML(yamlCfg)
	validateConfig(cfg)
	return cfg
}

func loadYAMLConfig(path string) (*yamlConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	var cfg yamlConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	return &cfg, nil
}

func configFromYAML(yamlCfg *yamlConfig) config {
	cfg := config{
		vmCount:         yamlCfg.Loadgen.VmCount,
		numPackages:     yamlCfg.Loadgen.NumPackages,
		numRepositories: yamlCfg.Loadgen.NumRepositories,
		port:            yamlCfg.Loadgen.Port,
		metricsPort:     yamlCfg.Loadgen.MetricsPort,
	}

	// Apply defaults
	if cfg.vmCount == 0 {
		cfg.vmCount = 50
	}
	if cfg.numPackages == 0 {
		cfg.numPackages = 700
	}
	if cfg.port == 0 {
		cfg.port = 818
	}

	// Parse durations with defaults
	if yamlCfg.Loadgen.StatsInterval != "" {
		if d, err := time.ParseDuration(yamlCfg.Loadgen.StatsInterval); err == nil {
			cfg.statsInterval = d
		}
	}
	if cfg.statsInterval == 0 {
		cfg.statsInterval = 10 * time.Second
	}

	if yamlCfg.Loadgen.RequestTimeout != "" {
		if d, err := time.ParseDuration(yamlCfg.Loadgen.RequestTimeout); err == nil {
			cfg.requestTimeout = d
		}
	}
	if cfg.requestTimeout == 0 {
		cfg.requestTimeout = 10 * time.Second
	}

	if yamlCfg.Loadgen.ReportInterval != "" {
		if d, err := time.ParseDuration(yamlCfg.Loadgen.ReportInterval); err == nil {
			cfg.reportInterval = d
		}
	}
	if cfg.reportInterval == 0 {
		cfg.reportInterval = 30 * time.Second
	}

	return cfg
}

func validateConfig(cfg config) {
	if cfg.vmCount <= 0 {
		log.Error("vmCount must be > 0")
		os.Exit(1)
	}
	if cfg.vmCount > 100000 {
		log.Error("vmCount must be <= 100000")
		os.Exit(1)
	}
	if cfg.reportInterval <= 0 {
		log.Error("reportInterval must be > 0")
		os.Exit(1)
	}
	if cfg.numPackages <= 0 {
		log.Error("numPackages must be > 0")
		os.Exit(1)
	}
}

func createReportGenerator(cfg config) (*vmindexreport.Generator, error) {
	generator := vmindexreport.NewGenerator(cfg.numPackages, cfg.numRepositories)
	log.Infof("Created report generator with %d packages, %d repositories",
		generator.NumPackages(), generator.NumRepositories())
	return generator, nil
}
