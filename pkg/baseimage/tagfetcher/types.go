package tagfetcher

import (
	"time"
)

// TagMetadata holds metadata fetched for a single image tag.
// This struct aligns with storage.BaseImageTag for cache persistence and
// provides layer digests for base image matching.
//
// TODO(ROX-32382): Until multi-arch support is added, IsManifestList is always
// false, ManifestDigest/LayerDigests are for the caller's architecture only,
// and ListDigests is empty.
type TagMetadata struct {
	// Tag is the image tag name (e.g., "8.10-1234").
	Tag string

	// ManifestDigest is the SHA256 digest of the manifest (or manifest list
	// for multi-arch images).
	ManifestDigest string

	// Created is the image creation timestamp from the config blob.
	Created *time.Time

	// LayerDigests contains the SHA256 digests of all layers in order.
	// Used for base image matching via layer commonality.
	LayerDigests []string

	// Error contains any error that occurred during metadata fetching.
	// If non-nil, only Tag is guaranteed to be populated.
	Error error
}
