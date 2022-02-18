package scannerconn

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/mtls"
)

type Conn struct {

}

func Dial(endpoint string) (*Conn, error) {
	if endpoint == "" {
		return nil, errors.New("Invalid Scanner endpoint (empty)")
	}

	endpoint = strings.TrimPrefix(endpoint, "https://")
	if strings.Contains(endpoint, "://") {
		return nil, errors.Errorf("Scanner endpoint has unsupported scheme: %s", endpoint)
	}

	tlsConfig, err := clientconn.TLSConfig(mtls.ScannerSubject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize Scanner TLS config")
	}


}
