package config

import (
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ClairURL           string        `yaml:"clair_url"`
	ClairDBConnString  string        `yaml:"clair_db_connstring"`
	GRPCListenAddr     string        `yaml:"grpc_listen_addr"`
	HTTPListenAddr     string        `yaml:"http_listen_addr"`
	UpdaterListenAddr  string        `yaml:"updater_listen_addr"`
	VulnerabilitiesURL string        `yaml:"vulnerabilities_url"`
	CertsDir           string        `yaml:"certs_dir"`
	Indexer            IndexerConfig `yaml:"indexer"`
	Matcher            MatcherConfig `yaml:"matcher"`
	LogLevel           slog.Level    `yaml:"log_level"`
}

type IndexerConfig struct {
	Database DatabaseConfig `yaml:"database"`
	Enable   bool           `yaml:"enable"`
}

type MatcherConfig struct {
	Database DatabaseConfig `yaml:"database"`
	Enable   bool           `yaml:"enable"`
}

type DatabaseConfig struct {
	ConnString string `yaml:"conn_string"`
}

func Defaults() *Config {
	return &Config{
		ClairURL:           "http://localhost:8080",
		GRPCListenAddr:     ":8443",
		HTTPListenAddr:     ":9443",
		UpdaterListenAddr:  ":9444",
		VulnerabilitiesURL: "https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulnerabilities.zip",
		Indexer:            IndexerConfig{Enable: true},
		Matcher:            MatcherConfig{Enable: true},
		LogLevel:           slog.LevelInfo,
	}
}

func Load(path string) (*Config, error) {
	cfg := Defaults()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}
