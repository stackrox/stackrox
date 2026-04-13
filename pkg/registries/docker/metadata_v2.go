package docker

import (
	"context"
	"encoding/json"
	"runtime"

	godigest "github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

func handleManifestLists(r *Registry, remote, ref string, manifests []manifestListEntry) (*storage.ImageMetadata, error) {
	if len(manifests) == 0 {
		return nil, errors.Errorf("no valid manifests found for %s:%s", remote, ref)
	}
	if len(manifests) == 1 {
		return handleManifests(r, manifests[0].MediaType, remote, manifests[0].Digest.String())
	}
	var amdManifest manifestListEntry
	var foundAMD bool
	for _, m := range manifests {
		if m.Platform.OS != "linux" {
			continue
		}
		// Matching platform for GOARCH takes priority so return immediately
		if m.Platform.Architecture == runtime.GOARCH {
			return handleManifests(r, m.MediaType, remote, m.Digest.String())
		}
		if m.Platform.Architecture == "amd64" {
			foundAMD = true
			amdManifest = m
		}
	}
	if foundAMD {
		return handleManifests(r, amdManifest.MediaType, remote, amdManifest.Digest.String())
	}
	return nil, errors.Errorf("no manifest in list matched linux and amd64 or %s architectures: %q", runtime.GOARCH, ref)
}

// HandleV2ManifestList takes in a v2 manifest list ref and returns the image metadata
func HandleV2ManifestList(r *Registry, remote, ref string) (*storage.ImageMetadata, error) {
	body, _, err := r.client.manifest(context.Background(), remote, ref)
	if err != nil {
		return nil, err
	}
	var ml v2ManifestList
	if err := json.Unmarshal(body, &ml); err != nil {
		return nil, errors.Wrap(err, "unmarshaling manifest list")
	}
	return handleManifestLists(r, remote, ref, ml.Manifests)
}

// HandleV2Manifest takes in a v2 ref and returns the image metadata
func HandleV2Manifest(r *Registry, remote, ref string) (*storage.ImageMetadata, error) {
	body, _, err := r.client.manifest(context.Background(), remote, ref)
	if err != nil {
		return nil, err
	}
	var manifest v2Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, errors.Wrap(err, "unmarshaling v2 manifest")
	}

	// Compute digest from the raw manifest bytes (the canonical representation).
	dig := godigest.FromBytes(body)

	layers := make([]string, 0, len(manifest.Layers))
	for _, layer := range manifest.Layers {
		layers = append(layers, layer.Digest.String())
	}

	var v1Metadata *storage.V1Metadata
	if manifest.Config.Digest != "" {
		v1Metadata, err = r.handleV1ManifestLayer(remote, manifest.Config.Digest)
		if err != nil {
			return nil, err
		}
	}
	return &storage.ImageMetadata{
		V1: v1Metadata,
		V2: &storage.V2Metadata{
			Digest: digestOrRef(ref, dig),
		},
		LayerShas: layers,
	}, nil
}

// HandleOCIImageIndex handles fetching data if the media type is OCI image index.
func HandleOCIImageIndex(r *Registry, remote, ref string) (*storage.ImageMetadata, error) {
	body, _, err := r.client.manifest(context.Background(), remote, ref)
	if err != nil {
		return nil, err
	}
	var index v1.Index
	if err := json.Unmarshal(body, &index); err != nil {
		return nil, errors.Wrap(err, "unmarshaling OCI image index")
	}

	// Convert OCI manifests to manifest list entries
	manifests := make([]manifestListEntry, 0, len(index.Manifests))
	for _, m := range index.Manifests {
		manifests = append(manifests, manifestListEntry{
			manifestDescriptor: manifestDescriptor{
				MediaType: m.MediaType,
				Size:      m.Size,
				Digest:    m.Digest,
			},
			Platform: platformSpec{
				Architecture: m.Platform.Architecture,
				OS:           m.Platform.OS,
				OSVersion:    m.Platform.OSVersion,
				OSFeatures:   m.Platform.OSFeatures,
				Variant:      m.Platform.Variant,
			},
		})
	}
	return handleManifestLists(r, remote, ref, manifests)
}

// HandleOCIManifest handles fetching data if the media type is OCI
func HandleOCIManifest(r *Registry, remote, ref string) (*storage.ImageMetadata, error) {
	body, _, err := r.client.manifest(context.Background(), remote, ref)
	if err != nil {
		return nil, err
	}
	var metadata v1.Manifest
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, errors.Wrap(err, "unmarshaling OCI manifest")
	}

	// Compute digest from the canonical bytes
	dig := godigest.FromBytes(body)

	layers := make([]string, 0, len(metadata.Layers))
	for _, layer := range metadata.Layers {
		layers = append(layers, layer.Digest.String())
	}

	var v1Metadata *storage.V1Metadata
	if metadata.Config.Digest != "" {
		v1Metadata, err = r.handleV1ManifestLayer(remote, metadata.Config.Digest)
		if err != nil {
			return nil, err
		}
	}
	return &storage.ImageMetadata{
		V1: v1Metadata,
		V2: &storage.V2Metadata{
			Digest: digestOrRef(ref, dig),
		},
		LayerShas: layers,
	}, nil
}

// digestOrRef returns digest if populated and ref is NOT a valid digest, otherwise returns ref.
func digestOrRef(ref string, digest godigest.Digest) string {
	if digest == "" {
		// If no digest, return the ref as is.
		return ref
	}

	if _, err := godigest.Parse(ref); err != nil {
		// If ref itself is not a digest, then return digest instead.
		return string(digest)
	}

	return ref
}
