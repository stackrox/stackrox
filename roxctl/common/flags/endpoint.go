package flags

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"k8s.io/utils/pointer"
)

var (
	endpoint        string
	endpointChanged = pointer.Bool(false)

	serverName    string
	serverNameSet = pointer.Bool(false)
	directGRPC    bool
	directGRPCSet = pointer.Bool(false)
	forceHTTP1    bool
	forceHTTP1Set = pointer.Bool(false)

	plaintext    bool
	plaintextSet = pointer.Bool(false)
	insecure     bool
	insecureSet  = pointer.Bool(false)

	insecureSkipTLSVerify    bool
	insecureSkipTLSVerifySet = pointer.Bool(false)

	caCertFile    string
	caCertFileSet = pointer.Bool(false)

	useKubeContext bool
)

const (
	caCertFileFlagName            = "ca"
	directGRPCFlagName            = "direct-grpc"
	endpointFlagName              = "endpoint"
	forceHTTP1FlagName            = "force-http1"
	insecureFlagName              = "insecure"
	insecureSkipTLSVerifyFlagName = "insecure-skip-tls-verify"
	plaintextFlagName             = "plaintext"
	serverNameFlagName            = "server-name"
	useKubeContextFlagName        = "use-current-k8s-context"
)

var connectionFlags = func() *pflag.FlagSet {
	fs := pflag.NewFlagSet("connection", pflag.ExitOnError)
	fs.StringVarP(&endpoint, endpointFlagName, "e", "localhost:8443",
		"Endpoint for service to contact. Alternatively, set the endpoint via the ROX_ENDPOINT environment variable")
	endpointChanged = &fs.Lookup(endpointFlagName).Changed
	fs.StringVarP(&serverName, serverNameFlagName, "s", "", "TLS ServerName to use for SNI "+
		"(if empty, derived from endpoint). Alternately, set the server name via the ROX_SERVER_NAME environment variable")
	serverNameSet = &fs.Lookup(serverNameFlagName).Changed
	fs.BoolVar(&directGRPC, directGRPCFlagName, false, "Use direct gRPC "+""+
		"(advanced; only use if you encounter connection issues). Alternately, enable by setting the ROX_DIRECT_GRPC_CLIENT "+
		"environment variable to true")
	directGRPCSet = &fs.Lookup(directGRPCFlagName).Changed
	fs.BoolVar(&forceHTTP1, forceHTTP1FlagName, false, "Always use HTTP/1 for all connections "+
		"(advanced; only use if you encounter connection issues). Alternatively, enable by setting the ROX_CLIENT_FORCE_HTTP1 "+
		"environment variable to true")
	forceHTTP1Set = &fs.Lookup(forceHTTP1FlagName).Changed

	fs.BoolVar(&plaintext, plaintextFlagName, false, "Use a plaintext (unencrypted) connection; "+
		"only works in conjunction with --insecure. Alternatively can be enabled by setting the ROX_PLAINTEXT environment variable to true")
	plaintextSet = &fs.Lookup(plaintextFlagName).Changed
	fs.BoolVar(&insecure, insecureFlagName, false, "Enable insecure connection options (DANGEROUS; USE WITH CAUTION). "+
		"Alternatively, enable insecure connection options by setting the ROX_INSECURE_CLIENT environment variable to true")
	insecureSet = &fs.Lookup(insecureFlagName).Changed
	fs.BoolVar(&insecureSkipTLSVerify, insecureSkipTLSVerifyFlagName, false, "Skip TLS certificate validation. "+
		"Alternatively, disable TLS certivicate validation by setting the ROX_INSECURE_CLIENT_SKIP_TLS_VERIFY environment variable to true")
	insecureSkipTLSVerifySet = &fs.Lookup(insecureSkipTLSVerifyFlagName).Changed
	fs.StringVar(&caCertFile, caCertFileFlagName, "", "Path to a custom CA certificate to use (PEM format). "+
		"Alternatively pass the file path using the ROX_CA_CERT_FILE environment variable")
	caCertFileSet = &fs.Lookup(caCertFileFlagName).Changed

	fs.BoolVarP(&useKubeContext, useKubeContextFlagName, "", false,
		"Use the current kubeconfig context to connect to the central service via port-forwarding. "+
			"Alternatively, set "+env.UseCurrentKubeContext.EnvVar()+" environment variable to true")
	return fs
}()

// AddCentralConnectionFlags adds connection-related flags to roxctl.
func AddCentralConnectionFlags(c *cobra.Command) {
	c.PersistentFlags().AddFlagSet(connectionFlags)
	c.MarkFlagsMutuallyExclusive(useKubeContextFlagName, endpointFlagName)

	addCentralAuthFlags(c)
}

// EndpointAndPlaintextSetting returns the Central endpoint to connect to, as well as a bool indicating whether to
// connect in plaintext mode. As connection requires a port it deduces it from provided schema. If schema is not provided
// the givenEndpoint must contain port or error is returned.
func EndpointAndPlaintextSetting() (string, bool, error) {
	endpoint = flagOrSettingValue(endpoint, *endpointChanged, env.EndpointEnv)
	if !strings.Contains(endpoint, "://") {
		if _, _, err := net.SplitHostPort(endpoint); err != nil {
			return "", false, errox.InvalidArgs.CausedBy(err)
		}
		return endpoint, plaintext, nil
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return "", false, errox.InvalidArgs.CausedBy(err)
	}

	if u.Path != "" && u.Path != "/" {
		return "", false, errox.InvalidArgs.New("endpoint URL must not include a path component")
	}

	var usePlaintext bool
	var defaultPort int
	switch u.Scheme {
	case "http":
		defaultPort = 80
		usePlaintext = true
	case "https":
		defaultPort = 443
		usePlaintext = false
	default:
		return "", false, errox.InvalidArgs.Newf("invalid scheme %q in endpoint URL, use either 'http' or 'https'", u.Scheme)
	}

	if *plaintextSet ||
		(!*plaintextSet && env.PlaintextEnv.BooleanSetting() != env.PlaintextEnv.DefaultBooleanSetting()) {
		if booleanFlagOrSettingValue(plaintext, *plaintextSet, env.PlaintextEnv) != usePlaintext {
			return "", false, errox.InvalidArgs.Newf("endpoint URL scheme %q is incompatible with --plaintext=%v setting", u.Scheme, plaintext)
		}
	}

	if u.Port() == "" {
		u.Host = fmt.Sprintf("%s:%d", u.Host, defaultPort)
	}

	return u.Host, usePlaintext, nil
}

// ServerName returns the specified ServerName.
func ServerName() string {
	return flagOrSettingValue(serverName, *serverNameSet, env.ServerEnv)
}

// UseDirectGRPC returns whether to use gRPC directly, i.e., without a proxy.
func UseDirectGRPC() bool {
	return booleanFlagOrSettingValue(directGRPC, *directGRPCSet, env.DirectGRPCEnv) ||
		UseKubeContext()
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

// UseKubeContext tells whether the connections should go through k8s port forwarding.
func UseKubeContext() bool {
	return useKubeContext || env.UseCurrentKubeContext.BooleanSetting()
}
