package flags

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
)

var (
	endpoint        string
	endpointChanged *bool

	serverName    string
	serverNameSet *bool
	directGRPC    bool
	directGRPCSet *bool
	forceHTTP1    bool
	forceHTTP1Set *bool

	plaintext    bool
	plaintextSet *bool
	insecure     bool
	insecureSet  *bool

	insecureSkipTLSVerify    bool
	insecureSkipTLSVerifySet *bool

	caCertFile    string
	caCertFileSet *bool
)

const (
	caCertFileFlagName            = "ca"
	directGRPCFlagName            = "direct-grpc"
	forceHTTP1FlagName            = "force-http1"
	insecureFlagName              = "insecure"
	insecureSkipTLSVerifyFlagName = "insecure-skip-tls-verify"
	plaintextFlagName             = "plaintext"
	serverNameFlagName            = "server-name"
)

// AddConnectionFlags adds connection-related flags to roxctl.
func AddConnectionFlags(c *cobra.Command) {
	c.PersistentFlags().StringVarP(&endpoint, "endpoint", "e", "localhost:8443",
		"endpoint for service to contact. Alternatively, set the endpoint via the ROX_ENDPOINT environment variable")
	endpointChanged = &c.PersistentFlags().Lookup("endpoint").Changed
	c.PersistentFlags().StringVarP(&serverName, serverNameFlagName, "s", "", "TLS ServerName to use for SNI "+
		"(if empty, derived from endpoint). Alternately, set the server name via the ROX_SERVER_NAME environment variable")
	serverNameSet = &c.PersistentFlags().Lookup(serverNameFlagName).Changed
	c.PersistentFlags().BoolVar(&directGRPC, directGRPCFlagName, false, "Use direct gRPC "+""+
		"(advanced; only use if you encounter connection issues). Alternately, enable by setting the ROX_DIRECT_GRPC_CLIENT "+
		"environment variable to true")
	directGRPCSet = &c.PersistentFlags().Lookup(directGRPCFlagName).Changed
	c.PersistentFlags().BoolVar(&forceHTTP1, forceHTTP1FlagName, false, "Always use HTTP/1 for all connections "+
		"(advanced; only use if you encounter connection issues). Alternatively, enable by setting the ROX_CLIENT_FORCE_HTTP1 "+
		"environment variable to true")
	forceHTTP1Set = &c.PersistentFlags().Lookup(forceHTTP1FlagName).Changed

	c.PersistentFlags().BoolVar(&plaintext, plaintextFlagName, false, "Use a plaintext (unencrypted) connection; "+
		"only works in conjunction with --insecure. Alternatively can be enabled by setting the ROX_PLAINTEXT environment variable to true")
	plaintextSet = &c.PersistentFlags().Lookup(plaintextFlagName).Changed
	c.PersistentFlags().BoolVar(&insecure, insecureFlagName, false, "Enable insecure connection options (DANGEROUS; USE WITH CAUTION). "+
		"Alternatively, enable insecure connection options by setting the ROX_INSECURE_CLIENT environment variable to true")
	insecureSet = &c.PersistentFlags().Lookup(insecureFlagName).Changed
	c.PersistentFlags().BoolVar(&insecureSkipTLSVerify, insecureSkipTLSVerifyFlagName, false, "Skip TLS certificate validation. "+
		"Alternatively, disable TLS certivicate validation by setting the ROX_INSECURE_CLIENT_SKIP_TLS_VERIFY environment variable to true")
	insecureSkipTLSVerifySet = &c.PersistentFlags().Lookup(insecureSkipTLSVerifyFlagName).Changed
	c.PersistentFlags().StringVar(&caCertFile, caCertFileFlagName, "", "Path to a custom CA certificate to use (PEM format). "+
		"Alternatively pass the file path using the ROX_CA_CERT_FILE environment variable")
	caCertFileSet = &c.PersistentFlags().Lookup(caCertFileFlagName).Changed
}

// EndpointAndPlaintextSetting returns the Central endpoint to connect to, as well as a bool indicating whether to
// connect in plaintext mode.
func EndpointAndPlaintextSetting() (string, bool, error) {
	endpoint = flagOrSettingValue(endpoint, *endpointChanged, env.EndpointEnv)
	if !strings.Contains(endpoint, "://") {
		return endpoint, plaintext, nil
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return "", false, errors.Wrap(err, "malformed endpoint URL")
	}

	if u.Path != "" && u.Path != "/" {
		return "", false, errox.InvalidArgs.New("endpoint URL must not include a path component")
	}

	var usePlaintext bool
	switch u.Scheme {
	case "http":
		usePlaintext = true
	case "https":
		usePlaintext = false
	default:
		return "", false, errox.InvalidArgs.Newf("invalid scheme %q in endpoint URL, the scheme should be: http(s)://<endpoint>:<port>", u.Scheme)
	}

	if *plaintextSet ||
		(!*plaintextSet && env.PlaintextEnv.BooleanSetting() != env.PlaintextEnv.DefaultBooleanSetting()) {
		if booleanFlagOrSettingValue(plaintext, *plaintextSet, env.PlaintextEnv) != usePlaintext {
			return "", false, errox.InvalidArgs.Newf("endpoint URL scheme %q is incompatible with --plaintext=%v setting", u.Scheme, plaintext)
		}
	}

	return u.Host, usePlaintext, nil
}

// ServerName returns the specified ServerName.
func ServerName() string {
	return flagOrSettingValue(serverName, *serverNameSet, env.ServerEnv)
}

// UseDirectGRPC returns whether to use gRPC directly, i.e., without a proxy.
func UseDirectGRPC() bool {
	return booleanFlagOrSettingValue(directGRPC, *directGRPCSet, env.DirectGRPCEnv)
}

// ForceHTTP1 indicates that the HTTP/1 should be used for all outgoing connections.
func ForceHTTP1() bool {
	return booleanFlagOrSettingValue(forceHTTP1, *forceHTTP1Set, env.ClientForceHTTP1Env)
}

// UseInsecure returns whether to use insecure connection behavior.
func UseInsecure() bool {
	return booleanFlagOrSettingValue(insecure, *insecureSet, env.InsecureClientEnv)
}

// SkipTLSValidation returns a bool that indicates the value of the `--insecure-skip-tls-verify` flag, with `nil`
// indicating that it was left at its default value.
func SkipTLSValidation() *bool {
	if !*insecureSkipTLSVerifySet {
		if env.InsecureClientSkipTLSVerifyEnv.BooleanSetting() == env.InsecureClientSkipTLSVerifyEnv.DefaultBooleanSetting() {
			return nil
		}
		envSetting := env.InsecureClientSkipTLSVerifyEnv.BooleanSetting()
		return &envSetting
	}
	return &insecureSkipTLSVerify
}

// CAFile returns the file for custom CA certificates.
func CAFile() string {
	return flagOrSettingValue(caCertFile, *caCertFileSet, env.CACertFileEnv)
}

// CentralURL returns the URL for the central instance based on the endpoint flags.
func CentralURL() (*url.URL, error) {
	endpoint, plaintext, err := EndpointAndPlaintextSetting()
	if err != nil {
		return nil, err
	}

	scheme := "https"
	if plaintext {
		scheme = "http"
	}
	rawURL := scheme + "://" + endpoint
	baseURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing central URL %s", rawURL)
	}
	return baseURL, nil
}
