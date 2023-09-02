package config

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

var (
	// DefaultConfiguration provides the default values for the scanner configuration.
	DefaultConfiguration = Config{
		HTTPListenAddr: ":9443",
		GRPCListenAddr: ":8443",
		Indexer: IndexerConfig{
			Enable:       true,
			DBConnString: "postgresql:///postgres?host=/var/run/postgresql",
		},
		Matcher: MatcherConfig{
			Enable:       true,
			DBConnString: "postgresql:///postgres?host=/var/run/postgresql",
		},
		// Default is empty.
		MTLS: MTLSConfig{
			CertsDir: "",
		},
		LogLevel: LogLevel(zerolog.InfoLevel),
	}
)

// Config represents the Scanner configuration parameters.
type Config struct {
	Indexer        IndexerConfig `yaml:"indexer"`
	Matcher        MatcherConfig `yaml:"matcher"`
	HTTPListenAddr string        `yaml:"http_listen_addr"`
	GRPCListenAddr string        `yaml:"grpc_listen_addr"`
	MTLS           MTLSConfig    `yaml:"mtls"`
	LogLevel       LogLevel      `yaml:"log_level"`
}

func (c *Config) validate() error {
	err := c.MTLS.validate()
	if err != nil {
		return err
	}
	return nil
}

// IndexerConfig provides Scanner Indexer configuration.
type IndexerConfig struct {
	// Database provides indexer's database configuration.
	DBConnString string `yaml:"db_conn_string"`
	// Enable if false disables the Indexer service.
	Enable bool `yaml:"enable"`
	// GetLayerTimeout timeout duration of GET requests for layers
	GetLayerTimeout Duration `yaml:"get_layer_timeout"`
}

// MatcherConfig provides Scanner Matcher configuration.
type MatcherConfig struct {
	// Database provides matcher's database configuration.
	DBConnString string `yaml:"db_conn_string"`
	// Enable if false disables the Matcher service and vulnerability updater.
	Enable bool `yaml:"enable"`
}

// MTLSConfig configures mutual TLS
type MTLSConfig struct {
	// CertsDir if set changes the prefix to find mTLS certificates and keys
	CertsDir string `yaml:"certs_dir"`
}

func (c *MTLSConfig) validate() error {
	p := c.CertsDir
	if p == "" {
		return nil
	}
	info, err := os.Stat(p)
	if err != nil {
		return fmt.Errorf("could not read mtls.certs_dir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("mtls.certs_dir is not a directory")
	}
	return nil
}

// LogLevel is YAML serializable zerolog.Level
type LogLevel zerolog.Level

// UnmarshalText implements YAML's TextUnmarshaler for LogLevel
func (l *LogLevel) UnmarshalText(level []byte) error {
	levelS := string(level)
	zl, err := zerolog.ParseLevel(levelS)
	if err != nil {
		return fmt.Errorf("unknown log_level string: %q", levelS)
	}
	*l = LogLevel(zl)
	return nil
}

// Duration is YAML serializable time.Duration
type Duration time.Duration

// UnmarshalText implements YAML's TextUnmarshaler for Duration
func (d *Duration) UnmarshalText(dBytes []byte) error {
	dStr := string(dBytes)
	td, err := time.ParseDuration(dStr)
	if err != nil {
		return err
	}
	*d = Duration(td)
	return nil
}

// Load parse and validates Scanner configuration.
func Load(r io.Reader) (*Config, error) {
	yd := yaml.NewDecoder(r)
	yd.KnownFields(true)
	cfg := DefaultConfiguration
	if err := yd.Decode(&cfg); err != nil {
		msg := strings.TrimPrefix(err.Error(), `yaml: `)
		return nil, fmt.Errorf("malformed yaml: %v", msg)
	}
	return &cfg, cfg.validate()
}

// Read load Scanner configuration from a file.
func Read(filename string) (*Config, error) {
	if filename == "" {
		cfg := DefaultConfiguration
		return &cfg, nil
	}
	r, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return Load(r)
}
