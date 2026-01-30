package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// distributionConfig represents a normal distribution with bounds.
type distributionConfig struct {
	Mean   float64 `yaml:"mean"`
	Stddev float64 `yaml:"stddev"`
	Min    float64 `yaml:"min"`
	Max    float64 `yaml:"max"`
}

// packageDistribution wraps distributionConfig for package counts.
type packageDistribution struct {
	distributionConfig
}

// intervalDistribution wraps distributionConfig for time intervals.
type intervalDistribution struct {
	distributionConfig
	// Parsed duration values (computed from mean/stddev/min/max in seconds)
	meanDuration   time.Duration
	stddevDuration time.Duration
	minDuration    time.Duration
	maxDuration    time.Duration
}

type config struct {
	vmCount  int
	duration time.Duration

	packageDist      packageDistribution
	intervalDist     intervalDistribution
	specificPackage  string // If set, use only this package (e.g., "vim-minimal", "basesystem")

	port           uint
	metricsPort    int
	statsInterval  time.Duration
	requestTimeout time.Duration
}

// yamlConfig represents the structure of the YAML config file.
type yamlConfig struct {
	Loadgen struct {
		VmCount         int         `yaml:"vmCount"`
		NumPackages     int         `yaml:"numPackages,omitempty"`     // Legacy scalar format
		Packages        interface{} `yaml:"packages,omitempty"`        // Can be int (legacy) or map (distribution)
		ReportInterval  interface{} `yaml:"reportInterval,omitempty"`  // Can be string (legacy) or map (distribution)
		SpecificPackage string      `yaml:"specificPackage,omitempty"` // Use specific package (e.g., "vim-minimal", "basesystem")
		StatsInterval   string      `yaml:"statsInterval"`
		Port            uint        `yaml:"port"`
		MetricsPort     int         `yaml:"metricsPort"`
		RequestTimeout  string      `yaml:"requestTimeout,omitempty"`
		Duration        string      `yaml:"duration,omitempty"`
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
		port:            yamlCfg.Loadgen.Port,
		metricsPort:     yamlCfg.Loadgen.MetricsPort,
		specificPackage: yamlCfg.Loadgen.SpecificPackage,
	}

	// Apply defaults
	if cfg.vmCount == 0 {
		cfg.vmCount = 50
	}
	if cfg.port == 0 {
		cfg.port = 818
	}

	// Parse packages: support both legacy scalar (numPackages) and new distribution format
	if yamlCfg.Loadgen.NumPackages > 0 {
		// Legacy scalar format: convert to distribution with stddev=0
		mean := float64(yamlCfg.Loadgen.NumPackages)
		cfg.packageDist = packageDistribution{
			distributionConfig: distributionConfig{
				Mean:   mean,
				Stddev: 0,
				Min:    mean,
				Max:    mean,
			},
		}
		log.Infof("Using legacy numPackages=%d (converted to distribution with mean=%.0f, stddev=0)", yamlCfg.Loadgen.NumPackages, mean)
	} else if yamlCfg.Loadgen.Packages != nil {
		// New distribution format
		pkgMap, ok := yamlCfg.Loadgen.Packages.(map[string]interface{})
		if !ok {
			log.Error("packages must be a map with mean, stddev, min, max")
			os.Exit(1)
		}
		pkgDist, err := parsePackageDistribution(pkgMap)
		if err != nil {
			log.Errorf("parsing packages distribution: %v", err)
			os.Exit(1)
		}
		cfg.packageDist = pkgDist
	} else {
		// Default fallback
		log.Warn("No packages configuration found, using default: mean=700, stddev=0")
		cfg.packageDist = packageDistribution{
			distributionConfig: distributionConfig{
				Mean:   700,
				Stddev: 0,
				Min:    700,
				Max:    700,
			},
		}
	}

	// Parse reportInterval: support both legacy scalar string and new distribution format
	if yamlCfg.Loadgen.ReportInterval != nil {
		// Check if it's a string (legacy) or map (distribution)
		if reportIntervalStr, ok := yamlCfg.Loadgen.ReportInterval.(string); ok {
			// Legacy scalar format: convert to distribution with stddev=0
			meanDur, err := time.ParseDuration(reportIntervalStr)
			if err != nil {
				log.Errorf("parsing reportInterval: %v", err)
				os.Exit(1)
			}
			meanSeconds := meanDur.Seconds()
			cfg.intervalDist = intervalDistribution{
				distributionConfig: distributionConfig{
					Mean:   meanSeconds,
					Stddev: 0,
					Min:    meanSeconds,
					Max:    meanSeconds,
				},
				meanDuration:   meanDur,
				stddevDuration: 0,
				minDuration:    meanDur,
				maxDuration:    meanDur,
			}
			log.Infof("Using legacy reportInterval=%s (converted to distribution with mean=%.2fs, stddev=0)", reportIntervalStr, meanSeconds)
		} else if reportIntervalMap, ok := yamlCfg.Loadgen.ReportInterval.(map[string]interface{}); ok {
			// New distribution format
			intervalDist, err := parseIntervalDistribution(reportIntervalMap)
			if err != nil {
				log.Errorf("parsing reportInterval distribution: %v", err)
				os.Exit(1)
			}
			cfg.intervalDist = intervalDist
		} else {
			log.Error("reportInterval must be a duration string (e.g., '60s') or a map with mean, stddev, min, max")
			os.Exit(1)
		}
	} else {
		// Default fallback
		log.Warn("No reportInterval configuration found, using default: mean=30s, stddev=0")
		defaultDur := 30 * time.Second
		cfg.intervalDist = intervalDistribution{
			distributionConfig: distributionConfig{
				Mean:   defaultDur.Seconds(),
				Stddev: 0,
				Min:    defaultDur.Seconds(),
				Max:    defaultDur.Seconds(),
			},
			meanDuration:   defaultDur,
			stddevDuration: 0,
			minDuration:    defaultDur,
			maxDuration:    defaultDur,
		}
	}

	// Parse durations with defaults
	if yamlCfg.Loadgen.StatsInterval != "" {
		d, err := time.ParseDuration(yamlCfg.Loadgen.StatsInterval)
		if err != nil {
			log.Errorf("parsing statsInterval: %v", err)
			os.Exit(1)
		}
		cfg.statsInterval = d
	}
	if cfg.statsInterval == 0 {
		cfg.statsInterval = 10 * time.Second
	}

	if yamlCfg.Loadgen.RequestTimeout != "" {
		d, err := time.ParseDuration(yamlCfg.Loadgen.RequestTimeout)
		if err != nil {
			log.Errorf("parsing requestTimeout: %v", err)
			os.Exit(1)
		}
		cfg.requestTimeout = d
	}
	if cfg.requestTimeout == 0 {
		cfg.requestTimeout = 10 * time.Second
	}

	// Parse optional duration (0 means run indefinitely)
	if yamlCfg.Loadgen.Duration != "" {
		d, err := time.ParseDuration(yamlCfg.Loadgen.Duration)
		if err != nil {
			log.Errorf("parsing duration: %v", err)
			os.Exit(1)
		}
		cfg.duration = d
	}

	return cfg
}

// toFloat64 converts an interface{} to float64, handling both int and float64 types from YAML
func toFloat64(v interface{}, fieldName string) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("%s must be a number, got %T", fieldName, v)
	}
}

func parsePackageDistribution(m map[string]interface{}) (packageDistribution, error) {
	var dist packageDistribution

	// Extract float64 values
	if meanVal, ok := m["mean"]; ok {
		mean, err := toFloat64(meanVal, "packages.mean")
		if err != nil {
			return dist, err
		}
		dist.Mean = mean
	} else {
		return dist, fmt.Errorf("packages.mean is required")
	}

	if stddevVal, ok := m["stddev"]; ok {
		stddev, err := toFloat64(stddevVal, "packages.stddev")
		if err != nil {
			return dist, err
		}
		dist.Stddev = stddev
	} else {
		return dist, fmt.Errorf("packages.stddev is required")
	}

	if minVal, ok := m["min"]; ok {
		min, err := toFloat64(minVal, "packages.min")
		if err != nil {
			return dist, err
		}
		dist.Min = min
	} else {
		return dist, fmt.Errorf("packages.min is required")
	}

	if maxVal, ok := m["max"]; ok {
		max, err := toFloat64(maxVal, "packages.max")
		if err != nil {
			return dist, err
		}
		dist.Max = max
	} else {
		return dist, fmt.Errorf("packages.max is required")
	}

	return dist, nil
}

func parseIntervalDistribution(m map[string]interface{}) (intervalDistribution, error) {
	var dist intervalDistribution
	var err error

	// Extract duration strings and parse them
	if meanVal, ok := m["mean"]; ok {
		if meanStr, ok := meanVal.(string); ok {
			meanDur, err := time.ParseDuration(meanStr)
			if err != nil {
				return dist, fmt.Errorf("reportInterval.mean: %w", err)
			}
			dist.Mean = meanDur.Seconds()
			dist.meanDuration = meanDur
		} else {
			return dist, fmt.Errorf("reportInterval.mean must be a duration string (e.g., '60s'), got %T", meanVal)
		}
	} else {
		return dist, fmt.Errorf("reportInterval.mean is required")
	}

	if stddevVal, ok := m["stddev"]; ok {
		if stddevStr, ok := stddevVal.(string); ok {
			stddevDur, err := time.ParseDuration(stddevStr)
			if err != nil {
				return dist, fmt.Errorf("reportInterval.stddev: %w", err)
			}
			dist.Stddev = stddevDur.Seconds()
			dist.stddevDuration = stddevDur
		} else {
			return dist, fmt.Errorf("reportInterval.stddev must be a duration string (e.g., '20s'), got %T", stddevVal)
		}
	} else {
		return dist, fmt.Errorf("reportInterval.stddev is required")
	}

	if minVal, ok := m["min"]; ok {
		if minStr, ok := minVal.(string); ok {
			minDur, err := time.ParseDuration(minStr)
			if err != nil {
				return dist, fmt.Errorf("reportInterval.min: %w", err)
			}
			dist.Min = minDur.Seconds()
			dist.minDuration = minDur
		} else {
			return dist, fmt.Errorf("reportInterval.min must be a duration string (e.g., '30s'), got %T", minVal)
		}
	} else {
		return dist, fmt.Errorf("reportInterval.min is required")
	}

	if maxVal, ok := m["max"]; ok {
		if maxStr, ok := maxVal.(string); ok {
			maxDur, err := time.ParseDuration(maxStr)
			if err != nil {
				return dist, fmt.Errorf("reportInterval.max: %w", err)
			}
			dist.Max = maxDur.Seconds()
			dist.maxDuration = maxDur
		} else {
			return dist, fmt.Errorf("reportInterval.max must be a duration string (e.g., '300s'), got %T", maxVal)
		}
	} else {
		return dist, fmt.Errorf("reportInterval.max is required")
	}

	return dist, err
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

	// Validate package distribution
	pkgDist := cfg.packageDist
	if pkgDist.Stddev < 0 {
		log.Error("packages.stddev must be >= 0")
		os.Exit(1)
	}
	if pkgDist.Min < 0 {
		log.Error("packages.min must be >= 0")
		os.Exit(1)
	}
	if pkgDist.Max < pkgDist.Min {
		log.Error("packages.max must be >= packages.min")
		os.Exit(1)
	}
	// Allow min == max for legacy scalar values (stddev=0)
	if pkgDist.Max > pkgDist.Min && (pkgDist.Mean < pkgDist.Min || pkgDist.Mean > pkgDist.Max) {
		log.Warnf("packages.mean (%.2f) is outside [%.2f, %.2f] range", pkgDist.Mean, pkgDist.Min, pkgDist.Max)
	}

	// Validate interval distribution
	intervalDist := cfg.intervalDist
	if intervalDist.Stddev < 0 {
		log.Error("reportInterval.stddev must be >= 0")
		os.Exit(1)
	}
	if intervalDist.Min < 0 {
		log.Error("reportInterval.min must be >= 0")
		os.Exit(1)
	}
	if intervalDist.Max < intervalDist.Min {
		log.Error("reportInterval.max must be >= reportInterval.min")
		os.Exit(1)
	}
	if intervalDist.minDuration < time.Second {
		log.Error("reportInterval.min must be >= 1s")
		os.Exit(1)
	}
	// Allow min == max for legacy scalar values (stddev=0)
	if intervalDist.Max > intervalDist.Min && (intervalDist.Mean < intervalDist.Min || intervalDist.Mean > intervalDist.Max) {
		log.Warnf("reportInterval.mean (%.2fs) is outside [%.2fs, %.2fs] range",
			intervalDist.Mean, intervalDist.Min, intervalDist.Max)
	}
}
