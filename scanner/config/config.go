package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/internal/version"
)

// MatcherReadiness labels the different readiness strategies Scanner can use.
type MatcherReadiness string

const (
	// ReadinessDatabase makes the matcher ready when the database connection is established.
	ReadinessDatabase MatcherReadiness = "database"
	// ReadinessVulnerability makes the matcher ready when the vulnerabilities are loaded at least once.
	ReadinessVulnerability MatcherReadiness = "vulnerability"
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
			GetLayerTimeout:    time.Minute,
			RepositoryToCPEURL: "https://security.access.redhat.com/data/metrics/repository-to-cpe.json",
			NameToReposURL:     "https://security.access.redhat.com/data/metrics/container-name-repos-map.json",
		},
		Matcher: MatcherConfig{
			Enable:    true,
			Readiness: ReadinessDatabase,
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
		LogLevel: slog.LevelInfo,
	}
)

// Config represents the Scanner configuration parameters.
type Config struct {
	// StackRoxServices indicates the Scanner is deployed alongside StackRox services.
	StackRoxServices bool          `mapstructure:"stackrox_services"`
	Indexer          IndexerConfig `mapstructure:"indexer"`
	Matcher          MatcherConfig `mapstructure:"matcher"`
	HTTPListenAddr   string        `mapstructure:"http_listen_addr"`
	GRPCListenAddr   string        `mapstructure:"grpc_listen_addr"`
	MTLS             MTLSConfig    `mapstructure:"mtls"`
	Proxy            ProxyConfig   `mapstructure:"proxy"`
	LogLevel         slog.Level    `mapstructure:"log_level"`
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
	Database Database `mapstructure:"database"`
	// Enable if false disables the Indexer service.
	Enable bool `mapstructure:"enable"`
	// GetLayerTimeout specifies the timeout duration of GET requests for layers
	GetLayerTimeout time.Duration `mapstructure:"get_layer_timeout"`
	// RepositoryToCPEURL specifies the URL to query for repository-to-cpe.json.
	RepositoryToCPEURL string `mapstructure:"repository_to_cpe_url"`
	// RepositoryToCPEURL specifies the location of the seed repository-to-cpe.json.
	RepositoryToCPEFile string `mapstructure:"repository_to_cpe_file"`
	// NameToReposURL specifies the URL to query for container-name-repos-map.json.
	NameToReposURL string `mapstructure:"name_to_repos_url"`
	// NameToReposFile specifies the location of the seed container-name-repos-map.json.
	NameToReposFile string `mapstructure:"name_to_repos_file"`
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
	Database Database `mapstructure:"database"`
	// Enable if false disables the Matcher service and vulnerability updater.
	Enable bool `mapstructure:"enable"`
	// IndexerAddr forces the matcher to retrieve index reports from a remote indexer
	// instance at the specified address, instead of the local indexer (when the
	// indexer is enabled).
	IndexerAddr string `mapstructure:"indexer_addr"`
	// VulnerabilitiesURL specifies the URL to query for vulnerabilities.
	VulnerabilitiesURL string `mapstructure:"vulnerabilities_url"`
	// EnableRCVulnBundle, when true, attempts to use an RC vulnerability bundle before falling back to the GA bundle.
	EnableRCVulnBundle bool `mapstructure:"enable_rc_vuln_bundle"`
	// VulnerabilitiesURLs internal list of candidate URLs for vulnerability bundles, ordered by preference.
	VulnerabilitiesURLs []string `mapstructure:"-"`
	// RemoteIndexerEnabled internal and generated flag, true when the remote indexer is enabled.
	RemoteIndexerEnabled bool `mapstructure:"-"`
	// VulnerabilityVersion allows overwriting the default version.Version and
	// version.VulnerabilityVersion (normally defined by the go build command).
	VulnerabilityVersion string `mapstructure:"vulnerability_version"`
	// Readiness determine the readiness type for the Matcher.
	Readiness MatcherReadiness `mapstructure:"readiness"`
	// VulnBundleAllowlist, when non-empty, restricts which vulnerability bundles
	// are imported on each update cycle. An empty list imports all bundles.
	// Bundle names are specified without file extension (e.g. "alpine", "nvd").
	// For the full list of bundle names see the vulnerability updater exporter.
	//
	// When populated, names not in the list are skipped during import. This reduces
	// the vulnerabilities loaded into the database, which speeds up update cycles
	// but means vulnerabilities from excluded bundles will not be detected,
	// leading to potentially incomplete scan results.
	VulnBundleAllowlist []string `mapstructure:"vuln_bundle_allowlist"`
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

	// Replace version placeholders.
	roxVer, vulnVer := c.resolveVersions()
	u := strings.ReplaceAll(c.VulnerabilitiesURL, "ROX_VERSION", roxVer)
	c.VulnerabilitiesURLs = make([]string, 0, 2)
	if c.EnableRCVulnBundle {
		// Prioritize RC-based vulnerability URL.
		c.VulnerabilitiesURLs = append(c.VulnerabilitiesURLs, strings.ReplaceAll(u, "ROX_VULNERABILITY_VERSION", vulnVer+"-rc"))
	}
	c.VulnerabilitiesURLs = append(c.VulnerabilitiesURLs, strings.ReplaceAll(u, "ROX_VULNERABILITY_VERSION", vulnVer))

	if c.Readiness == "" {
		return errors.New("readiness: cannot be empty")
	}

	switch c.Readiness {
	case ReadinessDatabase, ReadinessVulnerability:
	default:
		return fmt.Errorf("readiness: invalid readiness type %q", c.Readiness)
	}

	return nil
}

// Database provides database configuration for scanner backends.
type Database struct {
	// ConnString provides database DSN configuration.
	ConnString string `mapstructure:"conn_string"`
	// PasswordFile specifies the database password by reading from a file,
	// only valid for the password to be specified in a file if not in
	// the ConnString.
	PasswordFile string `mapstructure:"password_file"`
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
	CertsDir string `mapstructure:"certs_dir"`
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
	ConfigDir  string `mapstructure:"config_dir"`
	ConfigFile string `mapstructure:"config_file"`
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

// LogLevel is YAML serializable slog.Level
type LogLevel slog.Level

// UnmarshalText implements YAML's TextUnmarshaler for LogLevel
func (l *LogLevel) UnmarshalText(level []byte) error {
	sl, err := parseSlogLevel(string(level))
	if err != nil {
		return err
	}
	*l = LogLevel(sl)
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

func parseSlogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "trace", "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error", "fatal", "panic":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log_level string: %q", s)
	}
}

// stringToSlogLevelFunc returns a DecodeHookFunc that converts
// strings to slog.Level. This hook is used with mapstructure to enable
// automatic conversion of string log level values in configuration files
// to slog.Level types during unmarshaling.
func stringToSlogLevelFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(slog.LevelInfo) {
			return data, nil
		}
		return parseSlogLevel(data.(string))
	}
}

// Load loads Scanner configuration from the environment, and merge with a
// configuration file unless its reader is nil.
func Load(r io.Reader) (*Config, error) {
	v := viper.New()
	// Our config is in YAML.
	v.SetConfigType("yaml")
	// Allow env vars, but use `_` rather than `.` as a field separator.
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("SCANNER_V4")
	v.AutomaticEnv()
	// Decode the default configuration into a configuration map using mapstruct, so
	// we can initialize Viper's default keys (using MergeConfigMap).
	cfgMap := make(map[string]any)
	if err := mapstructure.Decode(defaultConfiguration, &cfgMap); err != nil {
		return nil, fmt.Errorf("decoding default config: %w", err)
	}
	if err := v.MergeConfigMap(cfgMap); err != nil {
		return nil, fmt.Errorf("merging default config: %w", err)
	}
	if r != nil {
		// Merge the values from the configuration file, if provided.
		if err := v.MergeConfig(r); err != nil {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	}
	cfg := defaultConfiguration
	if err := v.UnmarshalExact(&cfg, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
		stringToSlogLevelFunc(),
	))); err != nil {
		return nil, fmt.Errorf("loading config file: %w", err)
	}
	if cfg.StackRoxServices {
		cfg.Indexer.StackRoxServices = true
		cfg.Matcher.StackRoxServices = true
	}
	return &cfg, cfg.validate()
}

// Read loads Scanner configuration from the environment, and merge with a
// configuration file unless its filename is empty.
func Read(filename string) (*Config, error) {
	var r io.ReadCloser
	if filename != "" {
		var err error
		r, err = os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer utils.IgnoreError(r.Close)
	}
	return Load(r)
}
