package main

import (
	"flag"
	"fmt"
	"log"
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
	var cfg config

	flag.StringVar(&configFile, "config", "", "Path to YAML config file")
	flag.IntVar(&cfg.vmCount, "vm-count", 50, "Number of VMs to simulate")
	flag.DurationVar(&cfg.duration, "duration", 0, "Stop after this duration (0 = unbounded)")

	flag.IntVar(&cfg.numPackages, "num-packages", 700, "Number of packages per VM index report")
	flag.IntVar(&cfg.numRepositories, "num-repositories", 0, "Number of repositories per VM index report (0 = use real RHEL repos)")

	flag.UintVar(&cfg.port, "port", 818, "Vsock port for the relay")
	flag.IntVar(&cfg.metricsPort, "metrics-port", 9090, "Expose Prometheus metrics on this port (0 = disabled)")
	flag.DurationVar(&cfg.statsInterval, "stats-interval", 10*time.Second, "Console stats interval")
	flag.DurationVar(&cfg.requestTimeout, "request-timeout", 10*time.Second, "Per-request vsock deadline")
	flag.DurationVar(&cfg.reportInterval, "report-interval", 30*time.Second, "Interval at which each VM sends reports")
	flag.Parse()

	setFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	if configFile != "" {
		yamlCfg, err := loadYAMLConfig(configFile)
		if err != nil {
			log.Fatalf("loading config file: %v", err)
		}
		applyYAMLConfig(&cfg, yamlCfg, setFlags)
	}

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

func applyYAMLConfig(cfg *config, yamlCfg *yamlConfig, setFlags map[string]bool) {
	if !setFlags["vm-count"] && yamlCfg.Loadgen.VmCount > 0 {
		cfg.vmCount = yamlCfg.Loadgen.VmCount
	}
	if !setFlags["num-packages"] && yamlCfg.Loadgen.NumPackages > 0 {
		cfg.numPackages = yamlCfg.Loadgen.NumPackages
	}
	if !setFlags["num-repositories"] && yamlCfg.Loadgen.NumRepositories > 0 {
		cfg.numRepositories = yamlCfg.Loadgen.NumRepositories
	}
	if !setFlags["port"] && yamlCfg.Loadgen.Port > 0 {
		cfg.port = yamlCfg.Loadgen.Port
	}
	if !setFlags["metrics-port"] {
		cfg.metricsPort = yamlCfg.Loadgen.MetricsPort
	}
	if !setFlags["stats-interval"] && yamlCfg.Loadgen.StatsInterval != "" {
		if d, err := time.ParseDuration(yamlCfg.Loadgen.StatsInterval); err == nil {
			cfg.statsInterval = d
		}
	}
	if !setFlags["request-timeout"] && yamlCfg.Loadgen.RequestTimeout != "" {
		if d, err := time.ParseDuration(yamlCfg.Loadgen.RequestTimeout); err == nil {
			cfg.requestTimeout = d
		}
	}
	if !setFlags["report-interval"] && yamlCfg.Loadgen.ReportInterval != "" {
		if d, err := time.ParseDuration(yamlCfg.Loadgen.ReportInterval); err == nil {
			cfg.reportInterval = d
		}
	}
}

func validateConfig(cfg config) {
	if cfg.vmCount <= 0 {
		log.Fatalf("vm-count must be > 0")
	}
	if cfg.vmCount > 100000 {
		log.Fatalf("vm-count must be <= 100000")
	}
	if cfg.reportInterval <= 0 {
		log.Fatalf("report-interval must be > 0")
	}
	if cfg.numPackages <= 0 {
		log.Fatalf("num-packages must be > 0")
	}
}

func createReportGenerator(cfg config) (*vmindexreport.Generator, error) {
	generator := vmindexreport.NewGenerator(cfg.numPackages, cfg.numRepositories)
	log.Printf("Created report generator with %d packages, %d repositories",
		generator.NumPackages(), generator.NumRepositories())
	return generator, nil
}
