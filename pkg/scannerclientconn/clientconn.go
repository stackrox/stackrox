package scannerclientconn

import (
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Dial creates a client connection to Scanner at the given endpoint.
func Dial(endpoint string, dialOpts DialOptions) (*grpc.ClientConn, error) {
	if endpoint == "" {
		return nil, errors.New("Invalid Scanner endpoint (empty)")
	}

	endpoint = strings.TrimPrefix(endpoint, "https://")
	if strings.Contains(endpoint, "://") {
		return nil, errors.Errorf("Scanner endpoint has unsupported scheme: %s", endpoint)
	}

	creds := insecure.NewCredentials()
	if dialOpts.TLSConfig != nil {
		creds = credentials.NewTLS(dialOpts.TLSConfig)
	}

	return grpc.Dial(endpoint, grpc.WithTransportCredentials(creds))
}
