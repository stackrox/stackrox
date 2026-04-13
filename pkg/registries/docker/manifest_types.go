package docker

// Inline manifest types replacing github.com/docker/distribution.
// These are the JSON wire format for Docker Registry V2 manifest structs —
// only used for JSON unmarshal, no methods needed.

import (
	godigest "github.com/opencontainers/go-digest"
)

// Docker Registry V2 media type constants.
const (
	MediaTypeV1Manifest       = "application/vnd.docker.distribution.manifest.v1+json"
	MediaTypeV1SignedManifest = "application/vnd.docker.distribution.manifest.v1+prettyjws"
	MediaTypeV2ManifestList   = "application/vnd.docker.distribution.manifest.list.v2+json"
	MediaTypeV2Manifest       = "application/vnd.docker.distribution.manifest.v2+json"
)

// manifestDescriptor is a content-addressable object reference within a manifest.
type manifestDescriptor struct {
	MediaType string          `json:"mediaType,omitempty"`
	Size      int64           `json:"size,omitempty"`
	Digest    godigest.Digest `json:"digest,omitempty"`
}

// platformSpec identifies the platform a manifest targets.
type platformSpec struct {
	Architecture string   `json:"architecture"`
	OS           string   `json:"os"`
	OSVersion    string   `json:"os.version,omitempty"`
	OSFeatures   []string `json:"os.features,omitempty"`
	Variant      string   `json:"variant,omitempty"`
}

// manifestListEntry is a single entry in a V2 manifest list.
type manifestListEntry struct {
	manifestDescriptor
	Platform platformSpec `json:"platform"`
}

// v2ManifestList is the wire format for Docker V2 manifest lists.
type v2ManifestList struct {
	Manifests []manifestListEntry `json:"manifests"`
}

// v2Manifest is the wire format for Docker V2 image manifests (schema 2).
type v2Manifest struct {
	Config manifestDescriptor   `json:"config"`
	Layers []manifestDescriptor `json:"layers"`
}

// v1FSLayer is a filesystem layer reference in V1 manifests.
type v1FSLayer struct {
	BlobSum godigest.Digest `json:"blobSum"`
}

// v1History is a history entry in V1 manifests.
type v1History struct {
	V1Compatibility string `json:"v1Compatibility"`
}

// v1SignedManifest is the wire format for Docker V1 (signed) manifests.
type v1SignedManifest struct {
	FSLayers []v1FSLayer `json:"fsLayers"`
	History  []v1History `json:"history"`
}
