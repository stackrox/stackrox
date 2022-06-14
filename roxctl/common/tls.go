package common

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/clientconn"
	"github.com/stackrox/stackrox/pkg/mtls"
	"github.com/stackrox/stackrox/pkg/netutil"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/roxctl/common/flags"
	"github.com/stackrox/stackrox/roxctl/common/logger"
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
	logger           logger.Logger
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
			v.logger.WarnfLn(warningMsg)
			v.logger.ErrfLn("Certificate validation error:", err.Error())
		})
	}
	return nil
}

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

func tlsConfigOptsForCentral(logger logger.Logger) (*clientconn.TLSConfigOptions, error) {
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
			logger.WarnfLn("--insecure-skip-tls-verify has no effect when --ca is set")
		}
	} else {
		if flags.SkipTLSValidation() == nil {
			customVerifier = &insecureVerifierWithWarning{
				logger: logger,
			}
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

func tlsConfigForCentral(logger logger.Logger) (*tls.Config, error) {
	opts, err := tlsConfigOptsForCentral(logger)
	if err != nil {
		return nil, err
	}
	conf, err := clientconn.TLSConfig(mtls.CentralSubject, *opts)
	if err != nil {
		return nil, errors.Wrap(err, "invalid TLS config")
	}
	return conf, nil
}
