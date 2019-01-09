package docker

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

// HandleV2ManifestList takes in a v2 manifest list ref and returns the image metadata
func (r *Registry) HandleV2ManifestList(remote, ref string) (*storage.ImageMetadata, error) {
	manifestList, err := r.Client.ManifestList(remote, ref)
	if err != nil {
		return nil, err
	}
	for _, manifest := range manifestList.Manifests {
		// Default to linux arch
		if manifest.Platform.OS == "linux" && manifest.Platform.Architecture == "amd64" {
			return r.HandleV2Manifest(remote, manifest.Digest)
		}
	}
	return nil, fmt.Errorf("could not find manifest in list for architecture linux:amd64: '%s'", ref)
}

// HandleV2Manifest takes in a v2 ref and returns the image metadata
func (r *Registry) HandleV2Manifest(remote, ref string) (*storage.ImageMetadata, error) {
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
