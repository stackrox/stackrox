package common

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/roxctl/common/flags"
)

const warningMsg = `The remote endpoint failed TLS validation. This will be a fatal error in future releases.
Please do one of the following at your earliest convenience:
  1. Obtain a valid certificate for your Central instance/Load Balancer.
  2. Use the --ca option to specify a custom CA certificate (PEM format). This Certificate can be obtained by
     running "roxctl central cert".
  3. Update all your roxctl usages to pass the --insecure-skip-tls-verify option, in order to
     suppress this warning and retain the old behavior of not validating TLS certificates in
     the future (NOT RECOMMENDED).

`

type insecureVerifierWithWarning struct {
	printWarningOnce sync.Once
}

func (v *insecureVerifierWithWarning) VerifyPeerCertificate(leaf *x509.Certificate, chainRest []*x509.Certificate, conf *tls.Config) error {
	verifyOpts := x509.VerifyOptions{
		DNSName:       conf.ServerName,
		Intermediates: clientconn.NewCertPool(chainRest...),
		Roots:         conf.RootCAs,
	}

	_, err := leaf.Verify(verifyOpts)
	if err != nil {
		v.printWarningOnce.Do(func() {
			CLIEnvironment().Logger().WarnfLn(warningMsg)
			CLIEnvironment().Logger().ErrfLn("Certificate validation error:", err.Error())
		})
	}
	return nil
}

var (
	warningVerifierInstance insecureVerifierWithWarning
)

// ConnectNames returns the endpoint and (SNI) server name given by the
// --endpoint and --server-name flags respectively. If no server name is given,
// an appropriate name is derived from the given endpoint.
func ConnectNames() (string, string, error) {
	endpoint, _, err := flags.EndpointAndPlaintextSetting()
	if err != nil {
		return "", "", errors.Wrap(err, "could not get endpoint")
	}
	serverName := flags.ServerName()
	if serverName == "" {
		var err error
		serverName, _, _, err = netutil.ParseEndpoint(endpoint)
		if err != nil {
			return "", "", errors.Wrap(err, "could not parse endpoint")
		}
	}
	return endpoint, serverName, nil
}

func tlsConfigOptsForCentral() (*clientconn.TLSConfigOptions, error) {
	_, serverName, err := ConnectNames()
	if err != nil {
		return nil, errors.Wrap(err, "parsing central endpoint")
	}

	skipVerify := false
	var roots *x509.CertPool
	var customVerifier clientconn.TLSCertVerifier
	if flags.CAFile() != "" {
		caPEMData, err := os.ReadFile(flags.CAFile())
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse CA certificates from file")
		}
		roots = x509.NewCertPool()
		if !roots.AppendCertsFromPEM(caPEMData) {
			return nil, errors.Errorf("CA certificates file %s contains no certificates!", flags.CAFile())
		}
		if flags.SkipTLSValidation() != nil && *flags.SkipTLSValidation() {
			CLIEnvironment().Logger().WarnfLn("--insecure-skip-tls-verify has no effect when --ca is set")
		}
	} else {
		if flags.SkipTLSValidation() == nil {
			customVerifier = &warningVerifierInstance
		} else if *flags.SkipTLSValidation() {
			skipVerify = true
		}
	}

	return &clientconn.TLSConfigOptions{
		ServerName:         serverName,
		InsecureSkipVerify: skipVerify,
		CustomCertVerifier: customVerifier,
		RootCAs:            roots,
	}, nil
}

func tlsConfigForCentral() (*tls.Config, error) {
	opts, err := tlsConfigOptsForCentral()
	if err != nil {
		return nil, err
	}
	conf, err := clientconn.TLSConfig(mtls.CentralSubject, *opts)
	if err != nil {
		return nil, errors.Wrap(err, "invalid TLS config")
	}
	return conf, nil
}
