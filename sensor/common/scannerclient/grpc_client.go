package scannerclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/mtls"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Client is a Scanner gRPC client.
type Client struct {
	client scannerV1.ImageScanServiceClient
	conn   *grpc.ClientConn
}

// NewGRPCClient creates a new Scanner client.
func NewGRPCClient(endpoint string) (*Client, error) {
	if endpoint == "" {
		// No Scanner connection desired.
		return nil, nil
	}

	parts := strings.SplitN(endpoint, "://", 2)
	if parts[0] != "https" {
		if len(parts) != 1 {
			return nil, errors.Errorf("creating client unsupported scheme %s", parts[0])
		}

		endpoint = fmt.Sprintf("https://%s", endpoint)
	}

	// TODO: is this right?
	tlsConfig, err := clientconn.TLSConfig(mtls.ScannerSubject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize Scanner TLS config")
	}

	conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to Scanner")
	}

	return &Client{
		client: scannerV1.NewImageScanServiceClient(conn),
		conn: conn,
	}, nil
}

// GetImageAnalysis retrieves the image analysis results for the given image.
// The steps are as follows:
// 1. Retrieve image metadata.
// 2. Request image analysis from Scanner, directly.
// 3. Return image analysis results.
func (c *Client) GetImageAnalysis(ctx context.Context, image *storage.ContainerImage) (*scannerV1.GetImageComponentsResponse, error) {
	// TODO: get image metadata

	resp, err := c.client.GetImageComponents(ctx, &scannerV1.GetImageComponentsRequest{
		Image: image.GetId(),
		// TODO
		Registry: &scannerV1.RegistryData{
			Url:      image.GetName().GetRegistry(),
			Username: "",
			Password: "",
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "getting image components from scanner")
	}

	return resp, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
