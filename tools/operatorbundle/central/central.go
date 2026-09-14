// Package central provides StackRox Central-backed implementations of the
// operatorbundle.InstalledImageSource and operatorbundle.ImageScanner interfaces, using the
// v1 ImageService (GetImage for already-scanned installed images, ScanImage to scan
// update-candidate images).
package central

import (
	"context"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/operatorbundle"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client adapts the Central ImageService to the operatorbundle data-source interfaces.
type Client struct {
	svc            v1.ImageServiceClient
	force          bool
	includeSnoozed bool
}

// NewClient builds a Client from an established gRPC connection to Central.
func NewClient(conn *grpc.ClientConn, force bool) *Client {
	return &Client{
		svc:            v1.NewImageServiceClient(conn),
		force:          force,
		includeSnoozed: true,
	}
}

// GetCVEsByDigest returns the CVEs for an already-scanned installed image, identified by its
// digest. It returns (nil, nil) when the image is not present in Central.
func (c *Client) GetCVEsByDigest(ctx context.Context, digest string) (*operatorbundle.ImageCVEs, error) {
	image, err := c.svc.GetImage(ctx, &v1.GetImageRequest{Id: digest, IncludeSnoozed: c.includeSnoozed})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "getting image %s from Central", digest)
	}
	return operatorbundle.ImageCVEsFromStorage(image), nil
}

// ScanImage triggers a scan of the candidate image reference in Central and returns its CVEs.
func (c *Client) ScanImage(ctx context.Context, reference string) (*operatorbundle.ImageCVEs, error) {
	image, err := c.svc.ScanImage(ctx, &v1.ScanImageRequest{
		ImageName:      reference,
		Force:          c.force,
		IncludeSnoozed: c.includeSnoozed,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "scanning image %s", reference)
	}
	return operatorbundle.ImageCVEsFromStorage(image), nil
}
