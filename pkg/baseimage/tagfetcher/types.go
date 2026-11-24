package tagfetcher

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
)

// CurrentMetadataVersion identifies the set of derived fields this package
// writes into the tag cache and how they are computed.
//
// Bump this whenever a cached field is added, removed, or its derivation
// changes. Cached rows stamped with a different version are treated as misses,
// so their next normal scan writes the fetched data with the current version.
// No data migration is needed.
//
// Version history:
//   - 1: added BaseImageTag.history.
const CurrentMetadataVersion uint32 = 1

// KnownTag is the cached state for a single tag, used to decide whether a
// fetch result is redundant.
type KnownTag struct {
	// ManifestDigest is the digest stored in the cache.
	ManifestDigest string
	// MetadataVersion is the derivation version that produced the cached row.
	// Zero means the row predates versioning.
	MetadataVersion uint32
}

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

	// History is config history in order, including empty lines.
	// Layer digests omit those lines. Enrichment uses this when setting max_history_index.
	History []*storage.BaseImageHistoryEntry

	// MetadataVersion is the tagfetcher version that produced this metadata.
	// The watcher copies it onto the cached tag so stale rows can be detected.
	MetadataVersion uint32

	// Error contains any error that occurred during metadata fetching.
	// If non-nil, only Tag is guaranteed to be populated.
	Error error
}
