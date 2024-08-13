package docker

import (
	"runtime"

	"github.com/docker/distribution/manifest/manifestlist"
	godigest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

func handleManifestLists(r *Registry, remote, ref string, manifests []manifestlist.ManifestDescriptor) (*storage.ImageMetadata, error) {
	if len(manifests) == 0 {
		return nil, errors.Errorf("no valid manifests found for %s:%s", remote, ref)
	}
	if len(manifests) == 1 {
		return handleManifests(r, manifests[0].MediaType, remote, manifests[0].Digest)
	}
	var amdManifest manifestlist.ManifestDescriptor
	var foundAMD bool
	for _, m := range manifests {
		if m.Platform.OS != "linux" {
			continue
		}
		// Matching platform for GOARCH takes priority so return immediately
		if m.Platform.Architecture == runtime.GOARCH {
			return handleManifests(r, m.MediaType, remote, m.Digest)
		}
		if m.Platform.Architecture == "amd64" {
			foundAMD = true
			amdManifest = m
		}
	}
	if foundAMD {
		return handleManifests(r, amdManifest.MediaType, remote, amdManifest.Digest)
	}
	return nil, errors.Errorf("no manifest in list matched linux and amd64 or %s architectures: %q", runtime.GOARCH, ref)
}

// HandleV2ManifestList takes in a v2 manifest list ref and returns the image metadata
func HandleV2ManifestList(r *Registry, remote, ref string) (*storage.ImageMetadata, error) {
	manifestList, err := r.Client.ManifestList(remote, ref)
	if err != nil {
		return nil, err
	}
	return handleManifestLists(r, remote, ref, manifestList.Manifests)
}

// HandleV2Manifest takes in a v2 ref and returns the image metadata
func HandleV2Manifest(r *Registry, remote, ref string) (*storage.ImageMetadata, error) {
	metadata, dig, err := r.Client.ManifestV2WithDigest(remote, ref)
	if err != nil {
		return nil, err
	}
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

// HandleOCIImageIndex handles fetching data if the media type is OCI image index.
func HandleOCIImageIndex(r *Registry, remote, ref string) (*storage.ImageMetadata, error) {
	index, err := r.Client.ImageIndex(remote, ref)
	if err != nil {
		return nil, err
	}
	return handleManifestLists(r, remote, ref, index.Manifests)
}

// HandleOCIManifest handles fetching data if the media type is OCI
func HandleOCIManifest(r *Registry, remote, ref string) (*storage.ImageMetadata, error) {
	metadata, dig, err := r.Client.ManifestOCIWithDigest(remote, ref)
	if err != nil {
		return nil, err
	}
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
