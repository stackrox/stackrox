// Package config parses and validates all CLI flags and environment variables
// for the CO -> ACS importer tool.
package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/stackrox/co-acs-importer/internal/models"
)

const (
	defaultTokenEnv    = "ACS_API_TOKEN"
	defaultPasswordEnv = "ACS_PASSWORD"
	defaultTimeout     = 30 * time.Second
	defaultMaxRetries  = 5
)

// ParseAndValidate parses flags from args (typically os.Args[1:]), resolves
// environment variables, and validates the resulting Config.
// It uses a dedicated FlagSet so it is safe to call from tests.
func ParseAndValidate(args []string) (*models.Config, error) {
	fs := flag.NewFlagSet("co-acs-importer", flag.ContinueOnError)

	// IMP-CLI-001
	acsEndpoint := fs.String("acs-endpoint", os.Getenv("ACS_ENDPOINT"), "ACS endpoint URL (https://). Also read from ACS_ENDPOINT env var.")

	// IMP-CLI-023 / IMP-CLI-026
	acsAuthMode := fs.String("acs-auth-mode", "", "Auth mode: token (default) or basic. (IMP-CLI-023, IMP-CLI-026)")

	// IMP-CLI-002 / token mode
	acsTokenEnv := fs.String("acs-token-env", defaultTokenEnv, "Env var name that holds the ACS API token (token mode).")

	// IMP-CLI-024 / basic mode
	acsUsername := fs.String("acs-username", os.Getenv("ACS_USERNAME"), "ACS username for basic auth. Also read from ACS_USERNAME env var.")
	acsPasswordEnv := fs.String("acs-password-env", defaultPasswordEnv, "Env var name that holds the ACS password (basic mode).")

	// IMP-CLI-003
	kubeContext := fs.String("source-kubecontext", "", "Kubernetes context to use as source cluster (default: current context).")

	// IMP-CLI-004
	coNamespace := fs.String("co-namespace", "", "Namespace to read Compliance Operator resources from.")
	coAllNamespaces := fs.Bool("co-all-namespaces", false, "Read Compliance Operator resources from all namespaces.")

	// IMP-CLI-005
	acsClusterID := fs.String("acs-cluster-id", "", "ACS cluster ID that all imported scan configs target.")

	// IMP-CLI-007
	dryRun := fs.Bool("dry-run", false, "Disable all ACS write operations.")

	// IMP-CLI-008
	reportJSON := fs.String("report-json", "", "Write structured JSON report to this file path.")

	// IMP-CLI-009
	requestTimeout := fs.Duration("request-timeout", defaultTimeout, "HTTP request timeout (e.g. 30s).")

	// IMP-CLI-010
	maxRetries := fs.Int("max-retries", defaultMaxRetries, "Maximum number of retries for ACS API calls (min 0).")

	// IMP-CLI-011
	caCertFile := fs.String("ca-cert-file", "", "Path to CA certificate file for TLS verification.")

	// IMP-CLI-012
	insecureSkipVerify := fs.Bool("insecure-skip-verify", false, "Skip TLS certificate verification (not recommended for production).")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("flag parse error: %w", err)
	}

	cfg := &models.Config{
		ACSEndpoint:        *acsEndpoint,
		TokenEnv:           *acsTokenEnv,
		Username:           *acsUsername,
		PasswordEnv:        *acsPasswordEnv,
		KubeContext:        *kubeContext,
		CONamespace:        *coNamespace,
		COAllNamespaces:    *coAllNamespaces,
		ACSClusterID:       *acsClusterID,
		DryRun:             *dryRun,
		ReportJSON:         *reportJSON,
		RequestTimeout:     *requestTimeout,
		MaxRetries:         *maxRetries,
		CACertFile:         *caCertFile,
		InsecureSkipVerify: *insecureSkipVerify,
	}

	// IMP-CLI-026: default auth mode to token when not explicitly set.
	switch models.AuthMode(*acsAuthMode) {
	case "":
		cfg.AuthMode = models.AuthModeToken
	case models.AuthModeToken, models.AuthModeBasic:
		cfg.AuthMode = models.AuthMode(*acsAuthMode)
	default:
		return nil, fmt.Errorf(
			"invalid --acs-auth-mode %q: must be %q or %q (IMP-CLI-023)",
			*acsAuthMode, models.AuthModeToken, models.AuthModeBasic,
		)
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// validate checks all cross-field invariants after flags and env vars are resolved.
func validate(cfg *models.Config) error {
	// IMP-CLI-001: endpoint required.
	if cfg.ACSEndpoint == "" {
		return errors.New("--acs-endpoint (or ACS_ENDPOINT env var) is required (IMP-CLI-001)")
	}

	// IMP-CLI-013: endpoint must be https://.
	if !strings.HasPrefix(cfg.ACSEndpoint, "https://") {
		return fmt.Errorf("--acs-endpoint must start with https:// (got %q) (IMP-CLI-013)", cfg.ACSEndpoint)
	}

	// Strip trailing slash for consistency.
	cfg.ACSEndpoint = strings.TrimRight(cfg.ACSEndpoint, "/")

	// IMP-CLI-014 / IMP-CLI-025: validate auth material for the chosen mode.
	switch cfg.AuthMode {
	case models.AuthModeToken:
		token := os.Getenv(cfg.TokenEnv)
		if token == "" {
			return fmt.Errorf(
				"token auth mode requires a non-empty token in env var %q (IMP-CLI-014, IMP-CLI-025)\n"+
					"Fix: set %s=<your-api-token> before running",
				cfg.TokenEnv, cfg.TokenEnv,
			)
		}
	case models.AuthModeBasic:
		if cfg.Username == "" {
			return errors.New(
				"basic auth mode requires --acs-username (or ACS_USERNAME env var) to be non-empty (IMP-CLI-025)\n" +
					"Fix: pass --acs-username=<user> or set ACS_USERNAME=<user>",
			)
		}
		password := os.Getenv(cfg.PasswordEnv)
		if password == "" {
			return fmt.Errorf(
				"basic auth mode requires a non-empty password in env var %q (IMP-CLI-025)\n"+
					"Fix: set %s=<your-password> before running",
				cfg.PasswordEnv, cfg.PasswordEnv,
			)
		}
	}

	// IMP-CLI-004: must have exactly one of --co-namespace or --co-all-namespaces.
	if cfg.CONamespace == "" && !cfg.COAllNamespaces {
		return errors.New(
			"one of --co-namespace <ns> or --co-all-namespaces is required (IMP-CLI-004)",
		)
	}
	if cfg.CONamespace != "" && cfg.COAllNamespaces {
		return errors.New(
			"--co-namespace and --co-all-namespaces are mutually exclusive (IMP-CLI-004)",
		)
	}

	// IMP-CLI-005: cluster ID required.
	if cfg.ACSClusterID == "" {
		return errors.New("--acs-cluster-id is required (IMP-CLI-005)")
	}

	// IMP-CLI-010: max retries must be non-negative.
	if cfg.MaxRetries < 0 {
		return fmt.Errorf("--max-retries must be >= 0, got %d (IMP-CLI-010)", cfg.MaxRetries)
	}

	return nil
}
