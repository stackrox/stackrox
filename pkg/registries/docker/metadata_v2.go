package docker

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

// HandleV2ManifestList takes in a v2 manifest list ref and returns the image metadata
func HandleV2ManifestList(r *Registry, remote, ref string) (*storage.ImageMetadata, error) {
	manifestList, err := r.Client.ManifestList(remote, ref)
	if err != nil {
		return nil, err
	}
	if len(manifestList.Manifests) == 1 {
		return HandleV2Manifest(r, remote, manifestList.Manifests[0].Digest.String())
	}
	for _, manifest := range manifestList.Manifests {
		// Default to linux arch
		// TODO(ROX-13284): Support multi-arch images.
		if manifest.Platform.OS == "linux" && manifest.Platform.Architecture == "amd64" {
			return HandleV2Manifest(r, remote, manifest.Digest.String())
		}
	}
	return nil, fmt.Errorf("could not find manifest in list for architecture linux:amd64: '%s'", ref)
}

// HandleV2Manifest takes in a v2 ref and returns the image metadata
func HandleV2Manifest(r *Registry, remote, ref string) (*storage.ImageMetadata, error) {
	metadata, err := r.Client.ManifestV2(remote, ref)
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
			Digest: ref,
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
	if len(index.Manifests) == 1 {
		return HandleOCIManifest(r, remote, index.Manifests[0].Digest.String())
	}
	for _, manifest := range index.Manifests {
		// Default to linux arch
		// TODO(ROX-13284): Support multi-arch images.
		if manifest.Platform.OS == "linux" && manifest.Platform.Architecture == "amd64" {
			return HandleOCIManifest(r, remote, manifest.Digest.String())
		}
	}
	return nil, fmt.Errorf("could not find manifest in index for architecture linux:amd64: %q", ref)
}

// HandleOCIManifest handles fetching data if the media type is OCI
func HandleOCIManifest(r *Registry, remote, ref string) (*storage.ImageMetadata, error) {
	metadata, err := r.Client.ManifestOCI(remote, ref)
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
			Digest: ref,
		},
		LayerShas: layers,
	}, nil
}
