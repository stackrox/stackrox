// Package datasource provides in-process implementations of the pkg/operatorbundle data-source
// interfaces (InstalledImageSource, ImageScanner, DeployedImageSource) backed by Central's
// ImageService. It runs the same advisor logic the operatorbundle CLI uses, but without any
// gRPC-to-self: it calls the in-process ImageService directly.
package datasource

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	imageService "github.com/stackrox/rox/central/image/service"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/operatorbundle"
	"github.com/stackrox/rox/pkg/search"
)

// imageAPI is the subset of Central's ImageService this package uses. It is satisfied by
// central/image/service.Service (which embeds v1.ImageServiceServer).
type imageAPI interface {
	GetImage(context.Context, *v1.GetImageRequest) (*storage.Image, error)
	ScanImage(context.Context, *v1.ScanImageRequest) (*storage.Image, error)
	ListImages(context.Context, *v1.RawQuery) (*v1.ListImagesResponse, error)
}

// CentralImages adapts the Central ImageService to the operatorbundle data-source interfaces.
type CentralImages struct {
	svc imageAPI
}

// New builds a CentralImages from an image API (dependency-injected for testing).
func New(svc imageAPI) *CentralImages {
	return &CentralImages{svc: svc}
}

// Singleton builds a CentralImages backed by the in-process ImageService singleton.
func Singleton() *CentralImages {
	return New(imageService.Singleton())
}

// ResolveDigest maps a Central image ID (which may be a sha256 digest or an ImageV2 UUID) to the
// sha256 digest used by operator bundle relatedImages. Returns ("", nil) when the image is unknown.
func (c *CentralImages) ResolveDigest(ctx context.Context, imageID string) (string, error) {
	if strings.HasPrefix(imageID, "sha256:") {
		return imageID, nil
	}
	image, err := c.svc.GetImage(ctx, &v1.GetImageRequest{Id: imageID, IncludeSnoozed: true})
	if err != nil {
		if errors.Is(err, errox.NotFound) {
			return "", nil
		}
		return "", errors.Wrapf(err, "resolving image %s", imageID)
	}
	if sha := image.GetId(); strings.HasPrefix(sha, "sha256:") {
		return sha, nil
	}
	if full := image.GetName().GetFullName(); strings.Contains(full, "@") {
		return full[strings.Index(full, "@")+1:], nil
	}
	return image.GetId(), nil
}

// GetCVEsByDigest implements operatorbundle.InstalledImageSource using ImageService.GetImage.
// Returns (nil, nil) when the image is not present in Central.
func (c *CentralImages) GetCVEsByDigest(ctx context.Context, digest string) (*operatorbundle.ImageCVEs, error) {
	image, err := c.svc.GetImage(ctx, &v1.GetImageRequest{Id: digest, IncludeSnoozed: true})
	if err != nil {
		if errors.Is(err, errox.NotFound) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "getting image %s from Central", digest)
	}
	return operatorbundle.ImageCVEsFromStorage(image), nil
}

// ScanImage implements operatorbundle.ImageScanner using ImageService.ScanImage.
func (c *CentralImages) ScanImage(ctx context.Context, reference string) (*operatorbundle.ImageCVEs, error) {
	image, err := c.svc.ScanImage(ctx, &v1.ScanImageRequest{ImageName: reference, IncludeSnoozed: true})
	if err != nil {
		return nil, errors.Wrapf(err, "scanning image %s", reference)
	}
	return operatorbundle.ImageCVEsFromStorage(image), nil
}

// ListDeployedAmong implements operatorbundle.DeployedImageSource. It issues a single ListImages
// query: the exact digests OR-ed, AND-ed with a match-any deployment predicate that forces an inner
// join to deployments, so only images referenced by a running deployment are returned.
func (c *CentralImages) ListDeployedAmong(ctx context.Context, digests []string) (map[string]struct{}, error) {
	if len(digests) == 0 {
		return map[string]struct{}{}, nil
	}
	query := search.NewQueryBuilder().
		AddExactMatches(search.ImageSHA, digests...).
		AddRegexes(search.DeploymentName, ".*").
		Query()
	resp, err := c.svc.ListImages(ctx, &v1.RawQuery{Query: query})
	if err != nil {
		return nil, errors.Wrap(err, "listing deployed images from Central")
	}
	deployed := make(map[string]struct{}, len(resp.GetImages()))
	for _, img := range resp.GetImages() {
		deployed[img.GetId()] = struct{}{}
	}
	return deployed, nil
}
