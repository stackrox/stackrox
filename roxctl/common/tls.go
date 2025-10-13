package common

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/roxctl/common/flags"
)

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

func tlsConfigOptsForCentral() (*clientconn.TLSConfigOptions, error) {
	_, serverName, _, err := ConnectNames()
	if err != nil {
		return nil, errors.Wrap(err, "parsing central endpoint")
	}

	opts := &clientconn.TLSConfigOptions{
		ServerName:         serverName,
		InsecureSkipVerify: flags.SkipTLSValidation() != nil && *flags.SkipTLSValidation(),
	}

	if !opts.InsecureSkipVerify {
		if opts.RootCAs, err = getCertPool(); err != nil {
			return nil, err
		}
	}

	if flags.UseKubeContext() {
		opts.DialContext = getForwardingDialContext()
	}

	return opts, nil
}

func getCertPool() (*x509.CertPool, error) {
	roots, err := x509.SystemCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "could not get system certs pool")
	}
	if caFile := flags.CAFile(); caFile != "" {
		// Read the CA from the given file.
		if ca, err := os.ReadFile(caFile); err != nil {
			return nil, errors.Wrap(err, "failed to parse CA certificates from file")
		} else if !roots.AppendCertsFromPEM(ca) {
			return nil, errors.Errorf("CA certificates file %s contains no certificates", caFile)
		}
	} else if flags.UseKubeContext() {
		// Read the CA from the central secret.
		_, core, namespace, err := getConfigs()
		if err != nil {
			return nil, err
		}
		ca, err := getCentralCA(context.Background(), core, namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read the central CA from the central-tls secret")
		} else if !roots.AppendCertsFromPEM(ca) {
			return nil, errors.New("central-tls secret contains no certificates")
		}
	}
	return roots, nil
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
