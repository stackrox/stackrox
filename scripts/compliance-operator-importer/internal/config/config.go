// Package config parses and validates all CLI flags and environment variables
// for the CO -> ACS importer tool.
package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/stackrox/co-acs-importer/internal/models"
)

// ErrHelpRequested is returned by ParseAndValidate when --help is passed.
// Callers should treat this as a successful exit (code 0).
var ErrHelpRequested = errors.New("help requested")

// uuidPattern matches a standard UUID (8-4-4-4-12 hex).
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

const (
	defaultTimeout     = 30 * time.Second
	defaultMaxRetries  = 5
	defaultCONamespace = "openshift-compliance"
	defaultUsername    = "admin"
)

// repeatableStringFlag is a custom flag type for collecting multiple values.
type repeatableStringFlag struct {
	values *[]string
}

func (f *repeatableStringFlag) String() string {
	if f.values == nil {
		return ""
	}
	return strings.Join(*f.values, ",")
}

func (f *repeatableStringFlag) Set(value string) error {
	*f.values = append(*f.values, value)
	return nil
}

// ParseAndValidate parses flags from args (typically os.Args[1:]), resolves
// environment variables, and validates the resulting Config.
// It uses a dedicated FlagSet so it is safe to call from tests.
func ParseAndValidate(args []string) (*models.Config, error) {
	fs := flag.NewFlagSet("co-acs-scan-importer", flag.ContinueOnError)

	// Override default Usage with structured help text.
	fs.Usage = func() { printUsage(fs) }

	// --- ACS connection (IMP-CLI-001) ---
	endpoint := fs.String("endpoint", os.Getenv("ROX_ENDPOINT"),
		"ACS Central endpoint URL.\n"+
			"Bare hostnames get https:// prepended automatically.\n"+
			"Also read from the ROX_ENDPOINT environment variable.")

	// --- ACS authentication (IMP-CLI-024) ---
	username := fs.String("username", "",
		"Username for basic auth (default \"admin\").\n"+
			"Also read from ROX_ADMIN_USER environment variable.")

	// --- Compliance Operator namespace ---
	coNamespace := fs.String("co-namespace", defaultCONamespace,
		"Namespace containing Compliance Operator resources.\n"+
			"Overridden by --co-all-namespaces.")
	coAllNamespaces := fs.Bool("co-all-namespaces", false,
		"Read Compliance Operator resources from all namespaces.")

	// --- Import behavior ---
	dryRun := fs.Bool("dry-run", false,
		"Preview all actions without making any changes to ACS.\n"+
			"The report is still generated.")
	overwriteExisting := fs.Bool("overwrite-existing", false,
		"Update existing ACS scan configurations instead of skipping them.\n"+
			"Without this flag, existing configs are skipped with a warning.")
	reportJSON := fs.String("report-json", "",
		"Write a structured JSON report to this file path.")

	// --- HTTP / TLS ---
	requestTimeout := fs.Duration("request-timeout", defaultTimeout,
		"Timeout for each HTTP request to ACS (e.g. 30s, 1m).")
	maxRetries := fs.Int("max-retries", defaultMaxRetries,
		"Maximum retry attempts for transient ACS API failures (429, 502, 503, 504).")
	caCertFile := fs.String("ca-cert-file", "",
		"Path to a PEM-encoded CA certificate bundle for TLS verification.")
	insecureSkipVerify := fs.Bool("insecure-skip-verify", false,
		"Skip TLS certificate verification. Not recommended for production.")

	// --- Multi-cluster mode ---
	var kubeconfigs []string
	var kubecontexts []string
	var clusterValues []string
	fs.Var(&repeatableStringFlag{values: &kubeconfigs}, "kubeconfig",
		"Path to a kubeconfig file (repeatable). Each file represents one source cluster.\n"+
			"The current context in each file is used. Mutually exclusive with --kubecontext.")
	fs.Var(&repeatableStringFlag{values: &kubecontexts}, "kubecontext",
		"Kubernetes context name (repeatable). Use \"all\" to iterate every context.\n"+
			"Operates on the active kubeconfig (set via KUBECONFIG env var or ~/.kube/config).\n"+
			"Mutually exclusive with --kubeconfig.")
	fs.Var(&repeatableStringFlag{values: &clusterValues}, "cluster",
		"ACS cluster identification (repeatable). Accepts three forms:\n"+
			"  UUID: used directly as the ACS cluster ID (single-cluster).\n"+
			"  name: resolved via GET /v1/clusters (single-cluster).\n"+
			"  ctx=name-or-uuid: maps a kubeconfig context to an ACS cluster (multi-cluster).\n"+
			"Omit to auto-discover the ACS cluster ID.")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, ErrHelpRequested
		}
		return nil, fmt.Errorf("flag parse error: %w", err)
	}

	// Resolve username: flag > env > default.
	resolvedUsername := *username
	if resolvedUsername == "" {
		resolvedUsername = os.Getenv("ROX_ADMIN_USER")
	}
	if resolvedUsername == "" {
		resolvedUsername = defaultUsername
	}

	cfg := &models.Config{
		ACSEndpoint:        *endpoint,
		Username:           resolvedUsername,
		CONamespace:        *coNamespace,
		COAllNamespaces:    *coAllNamespaces,
		DryRun:             *dryRun,
		ReportJSON:         *reportJSON,
		RequestTimeout:     *requestTimeout,
		MaxRetries:         *maxRetries,
		CACertFile:         *caCertFile,
		InsecureSkipVerify: *insecureSkipVerify,
		OverwriteExisting:  *overwriteExisting,
		Kubeconfigs:        kubeconfigs,
		Kubecontexts:       kubecontexts,
	}

	// Classify --cluster values into overrides vs single-cluster shorthand.
	if err := classifyClusterValues(clusterValues, cfg); err != nil {
		return nil, err
	}

	// IMP-CLI-002: auto-infer auth mode from env vars.
	if err := inferAuthMode(cfg); err != nil {
		return nil, err
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// classifyClusterValues processes --cluster flag values:
//   - ctx=value → ClusterOverrides (for multi-cluster mode)
//   - UUID → ACSClusterID (single-cluster shorthand)
//   - name → ClusterNameLookup (single-cluster shorthand, resolved at runtime)
func classifyClusterValues(values []string, cfg *models.Config) error {
	var overrides []string
	var shorthands []string

	for _, v := range values {
		if strings.Contains(v, "=") {
			overrides = append(overrides, v)
		} else {
			shorthands = append(shorthands, v)
		}
	}

	if len(shorthands) > 1 {
		return fmt.Errorf("at most one --cluster shorthand (UUID or name) allowed, got %d: %v", len(shorthands), shorthands)
	}

	cfg.ClusterOverrides = overrides

	if len(shorthands) == 1 {
		v := shorthands[0]
		if uuidPattern.MatchString(v) {
			cfg.ACSClusterID = v
		} else {
			cfg.ClusterNameLookup = v
		}
	}

	return nil
}

// inferAuthMode sets cfg.AuthMode based on which env vars are present (IMP-CLI-002).
//   - ROX_API_TOKEN set → token mode
//   - ROX_ADMIN_PASSWORD set → basic mode
//   - both set → ambiguous error (IMP-CLI-025)
//   - neither set → error with help text (IMP-CLI-025)
func inferAuthMode(cfg *models.Config) error {
	hasToken := os.Getenv("ROX_API_TOKEN") != ""
	hasPassword := os.Getenv("ROX_ADMIN_PASSWORD") != ""

	switch {
	case hasToken && hasPassword:
		return errors.New(
			"ambiguous auth: both ROX_API_TOKEN and ROX_ADMIN_PASSWORD are set\n" +
				"Fix: unset one of them to select a single auth mode",
		)
	case hasToken:
		cfg.AuthMode = models.AuthModeToken
	case hasPassword:
		cfg.AuthMode = models.AuthModeBasic
	default:
		return errors.New(
			"no auth credentials found\n" +
				"Fix: set ROX_API_TOKEN for token auth, or ROX_ADMIN_PASSWORD for basic auth",
		)
	}
	return nil
}

// validate checks all cross-field invariants after flags and env vars are resolved.
func validate(cfg *models.Config) error {
	if cfg.ACSEndpoint == "" {
		return errors.New("--endpoint is required (or set ROX_ENDPOINT)")
	}

	// IMP-CLI-013: auto-prepend https:// for bare hostnames; reject http://.
	if strings.HasPrefix(cfg.ACSEndpoint, "http://") {
		return fmt.Errorf("--endpoint must not use http:// (got %q)\nFix: use https:// or omit the scheme", cfg.ACSEndpoint)
	}
	if !strings.HasPrefix(cfg.ACSEndpoint, "https://") {
		cfg.ACSEndpoint = "https://" + cfg.ACSEndpoint
	}

	// Strip trailing slash for consistency.
	cfg.ACSEndpoint = strings.TrimRight(cfg.ACSEndpoint, "/")

	// Auth material validation (IMP-CLI-014).
	switch cfg.AuthMode {
	case models.AuthModeToken:
		if os.Getenv("ROX_API_TOKEN") == "" {
			return errors.New(
				"ROX_API_TOKEN is empty\n" +
					"Fix: export ROX_API_TOKEN=<your-api-token>",
			)
		}
	case models.AuthModeBasic:
		if os.Getenv("ROX_ADMIN_PASSWORD") == "" {
			return errors.New(
				"ROX_ADMIN_PASSWORD is empty\n" +
					"Fix: export ROX_ADMIN_PASSWORD=<your-password>",
			)
		}
	}

	if cfg.COAllNamespaces {
		cfg.CONamespace = "" // --co-all-namespaces overrides any namespace setting
	}

	if len(cfg.Kubeconfigs) > 0 && len(cfg.Kubecontexts) > 0 {
		return errors.New("--kubeconfig and --kubecontext are mutually exclusive")
	}

	// In single-cluster mode without explicit --cluster, enable auto-discovery.
	isMultiClusterMode := len(cfg.Kubeconfigs) > 0 || len(cfg.Kubecontexts) > 0
	if !isMultiClusterMode && cfg.ACSClusterID == "" && cfg.ClusterNameLookup == "" {
		cfg.AutoDiscoverClusterID = true
	}

	if cfg.MaxRetries < 0 {
		return fmt.Errorf("--max-retries must be >= 0 (got %d)", cfg.MaxRetries)
	}

	return nil
}

// printUsage writes structured help text to stderr.
func printUsage(fs *flag.FlagSet) {
	w := os.Stderr
	fmt.Fprint(w, `co-acs-scan-importer - Import Compliance Operator scan schedules into ACS

DESCRIPTION
  Reads ScanSettingBinding resources from one or more Kubernetes clusters
  running the Compliance Operator and creates equivalent scan configurations
  in Red Hat Advanced Cluster Security (ACS) via the v2 API.

  The importer auto-discovers the ACS cluster ID for each source cluster
  by reading the admission-control ConfigMap, falling back to OpenShift
  ClusterVersion metadata or the Helm effective cluster name secret.

USAGE
  # Single cluster (current kubeconfig context, auto-discovers ACS cluster ID):
  co-acs-scan-importer \
    --endpoint central.example.com \
    --dry-run

  # Multi-cluster with separate kubeconfig files:
  co-acs-scan-importer \
    --kubeconfig /path/to/cluster-a.kubeconfig \
    --kubeconfig /path/to/cluster-b.kubeconfig \
    --endpoint central.example.com

  # Multi-cluster with merged kubeconfig and named contexts:
  KUBECONFIG=a.yaml:b.yaml:c.yaml co-acs-scan-importer \
    --kubecontext cluster-a \
    --kubecontext cluster-b \
    --endpoint central.example.com

  # All contexts in a merged kubeconfig:
  co-acs-scan-importer \
    --kubecontext all \
    --endpoint central.example.com

  # Update existing ACS scan configs instead of skipping them:
  co-acs-scan-importer \
    --kubeconfig /path/to/cluster.kubeconfig \
    --endpoint central.example.com \
    --overwrite-existing

  # Basic auth (for development/testing):
  ROX_ADMIN_PASSWORD=secret co-acs-scan-importer \
    --endpoint central.example.com \
    --username admin \
    --insecure-skip-verify

AUTHENTICATION
  Auth mode is auto-inferred from environment variables:
  - Set ROX_API_TOKEN for API token auth (production).
  - Set ROX_ADMIN_PASSWORD for basic auth (development/testing).
  - Setting both is an error (ambiguous).
  - Setting neither is an error.

MULTI-CLUSTER NOTES
  When clusters are spread across multiple kubeconfig files, use the
  --kubeconfig flag once per file. Each file's current context is used.

  When a single merged kubeconfig contains all clusters with unique context
  names, use --kubecontext to select them (or "all" to use every context).
  Merge kubeconfigs via: KUBECONFIG=a.yaml:b.yaml:c.yaml

  --kubeconfig and --kubecontext are mutually exclusive.

  ScanSettingBindings with the same name across multiple clusters are merged
  into a single ACS scan configuration targeting all matched clusters. The
  importer verifies that profiles and schedules match across clusters and
  reports an error if they differ.

AUTO-DISCOVERY
  In multi-cluster mode, the ACS cluster ID is auto-discovered for each
  source cluster using the following chain (first success wins):

  1. admission-control ConfigMap "cluster-id" key (namespace: stackrox)
  2. OpenShift ClusterVersion spec.clusterID matched against ACS provider metadata
  3. helm-effective-cluster-name secret matched against ACS cluster name

  Use --cluster ctx=name-or-uuid to override auto-discovery for a
  specific context.

EXIT CODES
  0  All bindings processed successfully (or nothing to do).
  1  Fatal error (bad config, auth failure, connectivity issue).
  2  Partial success (some bindings failed; see report for details).

ENVIRONMENT VARIABLES
  ROX_ENDPOINT          ACS Central URL (alternative to --endpoint).
  ROX_API_TOKEN         API token for token auth mode.
  ROX_ADMIN_PASSWORD    Password for basic auth mode.
  ROX_ADMIN_USER        Username for basic auth (default "admin").
  KUBECONFIG            Colon-separated list of kubeconfig file paths.

FLAGS
`)
	fs.PrintDefaults()
}
