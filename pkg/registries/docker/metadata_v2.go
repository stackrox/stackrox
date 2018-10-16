package docker

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
)

func (r *Registry) handleV2ManifestList(remote, ref string) (*v1.ImageMetadata, error) {
	manifestList, err := r.client.ManifestList(remote, ref)
	if err != nil {
		return nil, err
	}
	for _, manifest := range manifestList.Manifests {
		// Default to linux arch
		if manifest.Platform.OS == "linux" && manifest.Platform.Architecture == "amd64" {
			return r.handleV2Manifest(remote, manifest.Digest)
		}
	}
	return nil, fmt.Errorf("could not find manifest in list for architecture linux:amd64: '%s'", ref)
}

func (r *Registry) handleV2Manifest(remote, ref string) (*v1.ImageMetadata, error) {
	metadata, err := r.client.ManifestV2(remote, ref)
	if err != nil {
		return nil, err
	}
	layers := make([]string, 0, len(metadata.Layers))
	for _, layer := range metadata.Layers {
		layers = append(layers, layer.Digest.String())
	}

	var v1Metadata *v1.V1Metadata
	if metadata.Config.Digest != "" {
		v1Metadata, err = r.handleV1ManifestLayer(remote, metadata.Config.Digest)
		if err != nil {
			return nil, err
		}
	}
	return &v1.ImageMetadata{
		V1: v1Metadata,
		V2: &v1.V2Metadata{
			Digest: ref,
			Layers: layers,
		},
	}, nil
}
