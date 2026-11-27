package tagfetcher

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
)

// TagMetadata holds metadata fetched for a single image tag.
// This struct aligns with storage.BaseImageTag for cache persistence and
// provides layer digests for base image matching.
type TagMetadata struct {
	// Tag is the image tag name (e.g., "8.10-1234").
	Tag string

	// ManifestDigest is the top-level digest: the manifest-list digest for a
	// multi-arch image, or the single manifest digest otherwise. It is the
	// change-detection key (compared against the previously cached digest).
	ManifestDigest string

	// IsManifestList is true when the tag points to a manifest list (multi-platform).
	IsManifestList bool

	// ListDigests maps "os/arch" (or "os/arch/variant" when a variant is present) to
	// that platform's manifest digest, allowing per-platform change detection.
	ListDigests map[string]string

	// Platforms holds per-platform metadata, one entry per linux platform. For a
	// single-platform image this has exactly one entry.
	Platforms []PlatformTagMetadata

	// Created, LayerDigests and History describe the primary platform (the caller's
	// architecture when present, otherwise the first platform). They remain for the
	// watcher's current single-image promotion path and are superseded by Platforms
	// once promotion is per-platform.
	Created      *time.Time
	LayerDigests []string
	History      []*storage.BaseImageHistoryEntry

	// Error contains any error that occurred during metadata fetching.
	// If non-nil, only Tag is guaranteed to be populated.
	Error error
}

// PlatformTagMetadata holds metadata for a single platform of an image tag.
type PlatformTagMetadata struct {
	// OS, Architecture and Variant identify the platform. Variant distinguishes
	// platforms that share an os/architecture (e.g. arm v6/v7/v8), so os/architecture
	// alone is not a unique key.
	OS           string
	Architecture string
	Variant      string

	// ManifestDigest is this platform's (non-list) manifest digest.
	ManifestDigest string

	// Created is the image creation timestamp from the config blob.
	Created *time.Time

	// LayerDigests contains the SHA256 digests of all layers in order, for base image
	// matching via layer commonality.
	LayerDigests []string

	// History is config history in order, including empty lines.
	History []*storage.BaseImageHistoryEntry
}
