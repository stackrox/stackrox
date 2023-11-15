package config

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stackrox/rox/pkg/utils"
	"gopkg.in/yaml.v3"
)

var (
	// defaultConfiguration provides the default values for the scanner configuration.
	defaultConfiguration = Config{
		HTTPListenAddr: "127.0.0.1:9443",
		GRPCListenAddr: "127.0.0.1:8443",
		Indexer: IndexerConfig{
			Enable: true,
			Database: Database{
				ConnString:   "host=/var/run/postgresql",
				PasswordFile: "",
			},

			GetLayerTimeout: Duration(time.Minute),
		},
		Matcher: MatcherConfig{
			Enable: true,
			Database: Database{
				ConnString:   "host=/var/run/postgresql",
				PasswordFile: "",
			},
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
	if err := c.MTLS.validate(); err != nil {
		return fmt.Errorf("mtls: %w", err)
	}
	if c.HTTPListenAddr == "" {
		return errors.New("http_listen_addr is empty")
	}
	if c.GRPCListenAddr == "" {
		return errors.New("grpc_listen_addr is empty")
	}
	if err := c.Indexer.validate(); err != nil {
		return fmt.Errorf("indexer: %w", err)
	}
	if err := c.Matcher.validate(); err != nil {
		return fmt.Errorf("matcher: %w", err)
	}
	return nil
}

// IndexerConfig provides Scanner Indexer configuration.
type IndexerConfig struct {
	// Database provides indexer's database configuration.
	Database Database `yaml:"database"`
	// Enable if false disables the Indexer service.
	Enable bool `yaml:"enable"`
	// GetLayerTimeout timeout duration of GET requests for layers
	GetLayerTimeout Duration `yaml:"get_layer_timeout"`
}

func (c *IndexerConfig) validate() error {
	if !c.Enable {
		return nil
	}
	if err := c.Database.validate(); err != nil {
		return fmt.Errorf("database: %w", err)
	}
	return nil
}

// MatcherConfig provides Scanner Matcher configuration.
type MatcherConfig struct {
	// Database provides matcher's database configuration.
	Database Database `yaml:"database"`
	// Enable if false disables the Matcher service and vulnerability updater.
	Enable bool `yaml:"enable"`
	// IndexerAddr forces the matcher to retrieve index reports from a remote indexer
	// instance at the specified address, instead of the local indexer (when the
	// indexer is enabled).
	IndexerAddr string `yaml:"indexer_addr"`
	// RemoteIndexerEnabled internal and generated flag, true when the remote indexer is enabled.
	RemoteIndexerEnabled bool
}

func (c *MatcherConfig) validate() error {
	if !c.Enable {
		return nil
	}
	if err := c.Database.validate(); err != nil {
		return fmt.Errorf("database: %w", err)
	}
	c.RemoteIndexerEnabled = c.IndexerAddr != ""
	if c.RemoteIndexerEnabled {
		_, _, err := net.SplitHostPort(c.IndexerAddr)
		if err != nil {
			return fmt.Errorf("indexer_addr: failed to parse address: %w", err)
		}
	}
	return nil
}

// Database provides database configuration for scanner backends.
type Database struct {
	// ConnString provides database DSN configuration.
	ConnString string `yaml:"conn_string"`
	// PasswordFile specifies the database password by reading from a file,
	// only valid for the password to be specified in a file if not in
	// the ConnString.
	PasswordFile string `yaml:"password_file"`
}

func (d *Database) validate() error {
	if d.ConnString == "" {
		return errors.New("conn_string: empty is not allowed")
	}
	if strings.HasPrefix(d.ConnString, "postgres://") || strings.HasPrefix(d.ConnString, "postgresql://") {
		return errors.New("conn_string: URLs are not supported, use DSN")
	}
	cfg, err := pgxpool.ParseConfig(d.ConnString)
	if err != nil {
		return fmt.Errorf("conn_string: invalid: %w", err)
	}
	if cfg.ConnConfig.Password != "" && d.PasswordFile != "" {
		return errors.New("specify either password in conn_string or password file, but not both")
	}
	// TODO Technically this should be in Unmarshal(), it's here for convenience.
	if d.PasswordFile != "" {
		pw, err := os.ReadFile(d.PasswordFile)
		if err != nil {
			return fmt.Errorf("invalid password file %q: %w", d.PasswordFile, err)
		}
		d.ConnString = fmt.Sprintf("%s password=%s", d.ConnString, pw)
	}
	return nil
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
		return fmt.Errorf("could not read certs_dir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("certs_dir is not a directory: %s", p)
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
	cfg := defaultConfiguration
	if err := yd.Decode(&cfg); err != nil {
		msg := strings.TrimPrefix(err.Error(), `yaml: `)
		return nil, fmt.Errorf("malformed yaml: %v", msg)
	}
	return &cfg, cfg.validate()
}

// Read loads Scanner configuration from a file.
func Read(filename string) (*Config, error) {
	if filename == "" {
		cfg := defaultConfiguration
		return &cfg, nil
	}
	r, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer utils.IgnoreError(r.Close)
	return Load(r)
}
