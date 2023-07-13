package scannerV4Client

import (
	"strings"

	"github.com/pkg/errors"
	scannerV4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	log = logging.LoggerForModule()
)

// Client is a Scanner gRPC Client.
type Client struct {
	indexerClient scannerV4.IndexerClient
	matcherClient scannerV4.MatcherClient
	conn          *grpc.ClientConn
}

// dial Scanner and return a new Client.
// dial is non-blocking and returns a non-nil error upon configuration error.
func dial(endpoint string) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New("Invalid Scanner endpoint (empty)")
	}

	endpoint = strings.TrimPrefix(endpoint, "https://")
	if strings.Contains(endpoint, "://") {
		return nil, errors.Errorf("ScannerV4 endpoint has unsupported scheme: %s", endpoint)
	}

	tlsConfig, err := clientconn.TLSConfig(mtls.ScannerSubject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize ScannerV4 TLS config")
	}

	// This is non-blocking. If we ever want this to block,
	// then add the grpc.WithBlock() DialOption.
	conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial Scanner")
	}

	log.Infof("Dialing ScannerV4 at %s", endpoint)

	return &Client{
		indexerClient: scannerV4.NewIndexerClient(conn),
		matcherClient: scannerV4.NewMatcherClient(conn),
		conn:          conn,
	}, nil
}

// Close closes the underlying grpc.ClientConn.
func (c *Client) Close() error {
	return c.conn.Close()
}
