package scannerclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/mtls"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
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

// newGRPCClient creates a new Scanner client.
func newGRPCClient(endpoint string) (*client, error) {
	if endpoint == "" {
		log.Info("No Scanner connection desired")
		return nil, nil
	}

	parts := strings.SplitN(endpoint, "://", 2)
	if parts[0] != "https" {
		if len(parts) != 1 {
			return nil, errors.Errorf("creating client unsupported scheme %s", parts[0])
		}

		endpoint = fmt.Sprintf("https://%s", endpoint)
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

	log.Infof("Connected to Scanner at %s", endpoint)

	return &client{
		client: scannerV1.NewImageScanServiceClient(conn),
		conn:   conn,
	}, nil
}

// GetImageAnalysis retrieves the image analysis results for the given image.
// The steps are as follows:
// 1. Retrieve image metadata.
// 2. Request image analysis from Scanner, directly.
// 3. Return image analysis results.
func (c *client) GetImageAnalysis(ctx context.Context, image *storage.ContainerImage) (*imageData, error) {
	reg, err := getRegistry(image)
	if err != nil {
		return nil, errors.Wrapf(err, "determining image registry for %s in namespace %q", image.GetName().GetFullName(), image.GetNamespace())
	}

	metadata, err := reg.Metadata(types.ToImage(image))
	if err != nil {
		return nil, errors.Wrapf(err, "getting image metadata for %s in namespace %q", image.GetName().GetFullName(), image.GetNamespace())
	}

	// TODO: Switch to debug, but for now process signals are really bothering me
	log.Infof("Retrieved metadata for image %s in namespace %s", image.GetName().GetFullName(), image.GetNamespace())

	cfg := reg.Config()
	resp, err := c.client.GetImageComponents(ctx, &scannerV1.GetImageComponentsRequest{
		Image: image.GetId(),
		Registry: &scannerV1.RegistryData{
			Url:      cfg.URL,
			Username: cfg.Username,
			Password: cfg.Password,
			Insecure: cfg.Insecure,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "getting image components from scanner")
	}

	return &imageData{
		Metadata:                   metadata,
		GetImageComponentsResponse: resp,
	}, nil
}

func getRegistry(img *storage.ContainerImage) (registryTypes.Registry, error) {
	reg := img.GetName().GetRegistry()
	regs := registry.Singleton().GetAllInNamespace(img.GetNamespace())
	for _, r := range regs.GetAll() {
		if r.Name() == reg {
			return r, nil
		}
	}

	return nil, errors.Errorf("Unknown image registry: %q", reg)
}

func (c *client) Close() error {
	return c.conn.Close()
}
