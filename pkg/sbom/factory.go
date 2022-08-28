package sbom

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
)

// Fetcher is responsible for fetching SBOMs for an image from a registry.
type Fetcher interface {
	FetchSBOM(ctx context.Context, image *storage.Image, registry registryTypes.Registry) (*storage.ImageSBOM, error)
}

// Verifier is responsible for verifying whether SBOMs attached to an image cover all its contents.
type Verifier interface {
	VerifySBOM(ctx context.Context, image *storage.Image) error
}

// NewFetcher creates a new fetcher for SBOMs.
func NewFetcher() Fetcher {
	return newSigstoreSBOMFetcher()
}

// NewVerifier creates a new verifier for SBOMs.
func NewVerifier() Verifier {
	return newSBOMVerifier()
}
