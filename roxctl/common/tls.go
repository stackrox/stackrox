package common

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/tlscheck"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const errorMsg = `The remote endpoint failed TLS validation: %v

  Do one of the following:
  1. Obtain a valid certificate for your Central instance/Load Balancer.
  2. Use the --ca option to specify a custom CA certificate (PEM format).
     This certificate can be obtained by running "roxctl central cert".
  3. Use the --insecure-skip-tls-verify option to suppress this error
     and not validate TLS certificates (NOT RECOMMENDED).
`

type grpcPermissionDenied struct{ error }

func (p *grpcPermissionDenied) GRPCStatus() *status.Status {
	return status.New(codes.PermissionDenied, p.Error())
}

type insecureVerifierWithError struct {
	logger logger.Logger
}

func (v *insecureVerifierWithError) VerifyPeerCertificate(leaf *x509.Certificate, chainRest []*x509.Certificate, conf *tls.Config) error {
	verifyOpts := x509.VerifyOptions{
		DNSName:       conf.ServerName,
		Intermediates: tlscheck.NewCertPool(chainRest...),
		Roots:         conf.RootCAs,
	}
	v.logger.InfofLn("trying to verify cert for %s, signed by %s (CA %v)", leaf.Subject, leaf.Issuer, leaf.IsCA)
	for i, c := range chainRest {
		v.logger.InfofLn("%d cert in chain %s, signed by %s (CA %v)", i, c.Subject, c.Issuer, c.IsCA)
	}
	if _, err := leaf.Verify(verifyOpts); err != nil {
		v.logger.ErrfLn(errorMsg, err)
		return &grpcPermissionDenied{err}
	}
	return nil
}

// ConnectNames returns the endpoint and (SNI) server name given by the
// --endpoint and --server-name flags respectively and information about plaintext.
// If no server name is given, an appropriate name is derived from the given endpoint.
func ConnectNames() (string, string, bool, error) {
	endpoint, usePlaintext, err := flags.EndpointAndPlaintextSetting()
	if err != nil {
		return "", "", false, errors.Wrap(err, "could not get endpoint")
	}
	if flags.UseKubeContext() {
		endpoint, _, err = GetForwardingEndpoint()
		if err != nil {
			return "", "", false, errors.Wrap(err,
				"could not get endpoint forwarding to the central service in the current k8s context")
		}
	}
	serverName, err := getServerName(endpoint)
	if err != nil {
		return "", "", false, errors.Wrap(err, "could not get server name")
	}
	return endpoint, serverName, usePlaintext, nil
}

func getServerName(endpoint string) (string, error) {
	if serverName := flags.ServerName(); serverName != "" {
		return serverName, nil
	}
	serverName, _, _, err := netutil.ParseEndpoint(endpoint)
	if err != nil {
		return "", errors.Wrap(err, "could not parse endpoint")
	}
	return serverName, nil
}

func tlsConfigOptsForCentral(logger logger.Logger) (*clientconn.TLSConfigOptions, error) {
	_, serverName, _, err := ConnectNames()
	if err != nil {
		return nil, errors.Wrap(err, "parsing central endpoint")
	}

	var dialContext func(ctx context.Context, addr string) (net.Conn, error)

	skipVerify := flags.SkipTLSValidation() != nil && *flags.SkipTLSValidation()
	var roots *x509.CertPool
	var customVerifier tlscheck.TLSCertVerifier
	var ca []byte
	if skipVerify {
		if flags.CAFile() != "" {
			logger.WarnfLn("--insecure-skip-tls-verify has no effect when --ca is set")
		}
	} else {
		customVerifier = &insecureVerifierWithError{
			logger: logger,
		}
		if flags.CAFile() != "" {
			var err error
			if ca, err = os.ReadFile(flags.CAFile()); err != nil {
				return nil, errors.Wrap(err, "failed to parse CA certificates from file")
			}
		} else {
			// Read the CA from the central secret.
			if flags.UseKubeContext() {
				_, core, namespace, err := getConfigs()
				if err != nil {
					return nil, err
				}
				var warn error
				// Proceed with no CA on error. Return the error as warning later.
				ca, warn = getCentralCA(context.Background(), core, namespace)
				if warn != nil {
					logger.WarnfLn("Failed to read the central CA: %v", warn)
				}
			}
		}
	}

	if ca != nil {
		roots = x509.NewCertPool()
		if !roots.AppendCertsFromPEM(ca) {
			return nil, errors.Errorf("CA certificates file %s contains no certificates!", flags.CAFile())
		}
	}
	if flags.UseKubeContext() {
		dialContext = getForwardingDialContext()
	}

	return &clientconn.TLSConfigOptions{
		ServerName:         serverName,
		InsecureSkipVerify: skipVerify,
		CustomCertVerifier: customVerifier,
		RootCAs:            roots,
		DialContext:        dialContext,
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
