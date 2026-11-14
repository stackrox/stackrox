package operatorbundle

import (
	"context"
	"strings"
)

// CatalogClient resolves operator bundle information from an operator catalog (e.g. the Red
// Hat Ecosystem Catalog GraphQL API).
type CatalogClient interface {
	// FindInstalledBundle returns the bundle that ships the image identified by digest.
	// Implementations should return (nil, nil) when no bundle matches the digest.
	FindInstalledBundle(ctx context.Context, digest string) (*Bundle, error)
	// FindCandidateBundles returns bundles of the given package and channel created on or
	// after sinceCreationDate, ordered newest first.
	FindCandidateBundles(ctx context.Context, pkg, channel, sinceCreationDate string) ([]Bundle, error)
}

// ImageScanner obtains the CVEs for an image that may not yet exist in the system, by
// triggering a scan of the given reference (repository@digest).
type ImageScanner interface {
	ScanImage(ctx context.Context, reference string) (*ImageCVEs, error)
}

// InstalledImageSource returns CVEs for an image that has already been scanned and stored,
// identified by its digest.
type InstalledImageSource interface {
	// GetCVEsByDigest returns the CVEs for the stored image with the given digest.
	// Implementations should return (nil, nil) when the image is not found.
	GetCVEsByDigest(ctx context.Context, digest string) (*ImageCVEs, error)
}

// DeployedImageSource returns, from a set of image digests, the subset referenced by a
// currently-running workload (deployment) in the cluster. Used to optionally restrict analysis
// to actively-deployed images.
type DeployedImageSource interface {
	ListDeployedAmong(ctx context.Context, digests []string) (map[string]struct{}, error)
}

// RepositoryFromReference strips any tag and digest from an image reference, returning the
// registry+repository portion used to pair images across bundles. For example
// "registry.redhat.io/albo/controller-rhel9@sha256:abc" and
// "registry.redhat.io/albo/controller-rhel9:1.2" both yield
// "registry.redhat.io/albo/controller-rhel9".
func RepositoryFromReference(reference string) string {
	ref := reference
	// Strip digest first (everything from '@').
	if idx := strings.Index(ref, "@"); idx >= 0 {
		ref = ref[:idx]
	}
	// Strip a tag: the ':' after the last '/'. A ':' before the last '/' is part of a
	// registry:port and must be preserved.
	lastSlash := strings.LastIndex(ref, "/")
	if idx := strings.LastIndex(ref, ":"); idx > lastSlash {
		ref = ref[:idx]
	}
	return ref
}
