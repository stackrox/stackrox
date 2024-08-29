package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/internal/version"
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
			GetLayerTimeout:    Duration(time.Minute),
			RepositoryToCPEURL: "https://security.access.redhat.com/data/metrics/repository-to-cpe.json",
			NameToReposURL:     "https://security.access.redhat.com/data/metrics/container-name-repos-map.json",
		},
		Matcher: MatcherConfig{
			Enable: true,
			Database: Database{
				ConnString:   "host=/var/run/postgresql",
				PasswordFile: "",
			},
			VulnerabilitiesURL: "https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulnerabilities.zip",
		},
		// Default is empty.
		MTLS: MTLSConfig{
			CertsDir: "",
		},
		Proxy: ProxyConfig{
			ConfigFile: "config.yaml",
		},
		LogLevel: LogLevel(zerolog.InfoLevel),
	}
)

// Config represents the Scanner configuration parameters.
type Config struct {
	// StackRoxServices indicates the Scanner is deployed alongside StackRox services.
	StackRoxServices bool          `yaml:"stackrox_services"`
	Indexer          IndexerConfig `yaml:"indexer"`
	Matcher          MatcherConfig `yaml:"matcher"`
	HTTPListenAddr   string        `yaml:"http_listen_addr"`
	GRPCListenAddr   string        `yaml:"grpc_listen_addr"`
	MTLS             MTLSConfig    `yaml:"mtls"`
	Proxy            ProxyConfig   `yaml:"proxy"`
	LogLevel         LogLevel      `yaml:"log_level"`
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

	if err := c.Proxy.validate(); err != nil {
		return fmt.Errorf("proxy: %w", err)
	}

	return nil
}

// IndexerConfig provides Scanner Indexer configuration.
type IndexerConfig struct {
	// StackRoxServices specifies whether Indexer is deployed alongside StackRox services.
	StackRoxServices bool
	// Database provides indexer's database configuration.
	Database Database `yaml:"database"`
	// Enable if false disables the Indexer service.
	Enable bool `yaml:"enable"`
	// GetLayerTimeout specifies the timeout duration of GET requests for layers
	GetLayerTimeout Duration `yaml:"get_layer_timeout"`
	// RepositoryToCPEURL specifies the URL to query for repository-to-cpe.json.
	RepositoryToCPEURL string `yaml:"repository_to_cpe_url"`
	// RepositoryToCPEURL specifies the location of the seed repository-to-cpe.json.
	RepositoryToCPEFile string `yaml:"repository_to_cpe_file"`
	// NameToReposURL specifies the URL to query for container-name-repos-map.json.
	NameToReposURL string `yaml:"name_to_repos_url"`
	// NameToReposFile specifies the location of the seed container-name-repos-map.json.
	NameToReposFile string `yaml:"name_to_repos_file"`
}

func (c *IndexerConfig) validate() error {
	if !c.Enable {
		return nil
	}

	if err := c.Database.validate(); err != nil {
		return fmt.Errorf("database: %w", err)
	}

	if c.RepositoryToCPEURL == "" {
		return errors.New("repository_to_cpe_url: cannot be empty")
	}
	if _, err := url.Parse(c.RepositoryToCPEURL); err != nil {
		return fmt.Errorf("repository_to_cpe_url: invalid URL: %w", err)
	}
	if c.RepositoryToCPEFile != "" {
		if _, err := os.Stat(c.RepositoryToCPEFile); err != nil {
			return fmt.Errorf("repository_to_cpe_file: %w", err)
		}
	}

	if c.NameToReposURL == "" {
		return errors.New("name_to_repos_url: cannot be empty")
	}
	if _, err := url.Parse(c.NameToReposURL); err != nil {
		return fmt.Errorf("name_to_repos_url: invalid URL: %w", err)
	}
	if c.NameToReposFile != "" {
		if _, err := os.Stat(c.NameToReposFile); err != nil {
			return fmt.Errorf("name_to_repos_file: %w", err)
		}
	}

	return nil
}

// MatcherConfig provides Scanner Matcher configuration.
type MatcherConfig struct {
	// StackRoxServices specifies whether Matcher is deployed alongside StackRox services.
	StackRoxServices bool
	// Database provides matcher's database configuration.
	Database Database `yaml:"database"`
	// Enable if false disables the Matcher service and vulnerability updater.
	Enable bool `yaml:"enable"`
	// IndexerAddr forces the matcher to retrieve index reports from a remote indexer
	// instance at the specified address, instead of the local indexer (when the
	// indexer is enabled).
	IndexerAddr string `yaml:"indexer_addr"`
	// VulnerabilitiesURL specifies the URL to query for vulnerabilities.
	VulnerabilitiesURL string `yaml:"vulnerabilities_url"`
	// RemoteIndexerEnabled internal and generated flag, true when the remote indexer is enabled.
	RemoteIndexerEnabled bool
	VulnerabilityVersion string `yaml:"vulnerability_version"`
}

// resolveVersions returns values for ROX_VERSION and ROX_VULNERABILITY_VERSION
// based on the current build information and version string name.  If the user
// has explicitly set a VulnerabilityVersion in the configuration, it overrides
// the default values to maintain backward compatibility with pre-existing
// configs.
func (c *MatcherConfig) resolveVersions() (roxVer, vulnVer string) {
	roxVer = "dev"
	vulnVer = "dev"
	// Rely on buildinfo first to check release builds. It is defined by
	// build tags and has stronger reliability.
	if buildinfo.ReleaseBuild {
		roxVer = version.Version
		vulnVer = version.VulnerabilityVersion
	}
	if c.VulnerabilityVersion != "" {
		// We overwrite ROX_VERSION to not break existing
		// configurations, acknowledging that the configuration name
		// VulnerabilityVersion is confusing.
		roxVer = c.VulnerabilityVersion
		vulnVer = c.VulnerabilityVersion
	}
	return
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

	if c.VulnerabilitiesURL == "" {
		return errors.New("vulnerabilities_url: cannot be empty")
	}
	if _, err := url.Parse(c.VulnerabilitiesURL); err != nil {
		return fmt.Errorf("vulnerabilities_url: invalid URL: %w", err)
	}

	roxVer, vulnVer := c.resolveVersions()
	c.VulnerabilitiesURL = strings.ReplaceAll(c.VulnerabilitiesURL, "ROX_VERSION", roxVer)
	c.VulnerabilitiesURL = strings.ReplaceAll(c.VulnerabilitiesURL, "ROX_VULNERABILITY_VERSION", vulnVer)

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

// ProxyConfig configures HTTP proxies.
type ProxyConfig struct {
	ConfigDir  string `yaml:"config_dir"`
	ConfigFile string `yaml:"config_file"`
}

func (c *ProxyConfig) validate() error {
	dir := c.ConfigDir
	if dir == "" {
		return nil
	}

	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("could not read config_dir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("config_dir is not a directory: %s", dir)
	}

	if c.ConfigFile == "" {
		return errors.New("config_file: cannot be empty")
	}

	path := filepath.Join(dir, c.ConfigFile)
	info, err = os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// When the proxy is configured to be a Kubernetes secret,
			// the file will not exist if the secret does not exist.
			// Just allow this and log it, as the proxy config watcher will handle it.
			log.Printf("config_file %q does not exist, continuing...", path)
			return nil
		}
		return fmt.Errorf("could not read config_file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("config_file is a directory: %s", path)
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
	if cfg.StackRoxServices {
		cfg.Indexer.StackRoxServices = true
		cfg.Matcher.StackRoxServices = true
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
