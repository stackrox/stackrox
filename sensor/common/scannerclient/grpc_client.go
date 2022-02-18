package scannerclient

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/sensor/common/registry"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// client is a Scanner gRPC client.
type client struct {
	client scannerV1.ImageScanServiceClient
	conn   *grpc.ClientConn
}

// dial Scanner and return a new client.
func dial(endpoint string) (*client, error) {
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

	conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to Scanner")
	}

	log.Infof("Dialing Scanner at %s", endpoint)

	return &client{
		client: scannerV1.NewImageScanServiceClient(conn),
		conn:   conn,
	}, nil
}

func ()

// GetImageAnalysis retrieves the image analysis results for the given image.
// The steps are as follows:
// 1. Retrieve image metadata.
// 2. Request image analysis from Scanner, directly.
// 3. Return image analysis results.
func (c *client) GetImageAnalysis(ctx context.Context, ci *storage.ContainerImage) (*imageData, error) {
	reg, err := registry.Singleton().GetRegistryForImage(ci.GetName())
	if err != nil {
		return nil, errors.Wrap(err, "determining image registry")
	}

	name := ci.GetName().GetFullName()

	image := types.ToImage(ci)
	metadata, err := reg.Metadata(image)
	if err != nil {
		log.Debugf("Failed to get metadata for image %s: %v", name, err)
		return nil, errors.Wrap(err, "getting image metadata")
	}

	log.Debugf("Retrieved metadata for image %s: %v", name, metadata)

	cfg := reg.Config()
	resp, err := c.client.GetImageComponents(ctx, &scannerV1.GetImageComponentsRequest{
		Image: utils.GetFullyQualifiedFullName(image),
		Registry: &scannerV1.RegistryData{
			Url:      cfg.URL,
			Username: cfg.Username,
			Password: cfg.Password,
			Insecure: cfg.Insecure,
		},
	})
	if err != nil {
		log.Debugf("Unable to get image components from local Scanner for image %s: %v", name, err)
		return nil, errors.Wrap(err, "getting image components from scanner")
	}

	log.Debugf("Got image components from local Scanner for image %s", name)

	return &imageData{
		Metadata:                   metadata,
		GetImageComponentsResponse: resp,
	}, nil
}

func (c *client) Close() error {
	return c.conn.Close()
}
