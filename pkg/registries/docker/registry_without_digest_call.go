package docker

import (
	"fmt"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/errorhelpers"
)

var manifestFuncs = []func(registry *Registry, remote, ref string) (*storage.ImageMetadata, error){
	HandleV2ManifestList,
	HandleV2Manifest,
	HandleOCIManifest,
	HandleV1SignedManifest,
	HandleV1Manifest,
}

// RegistryWithoutManifestCall is the basic docker registry implementation without the manifest digest call
type RegistryWithoutManifestCall struct {
	*Registry
}

// NewRegistryWithoutManifestCall creates a new basic docker registry without a manifest digest call
func NewRegistryWithoutManifestCall(integration *storage.ImageIntegration) (*RegistryWithoutManifestCall, error) {
	dockerRegistry, err := NewDockerRegistry(integration)
	if err != nil {
		return nil, err
	}

	r := &RegistryWithoutManifestCall{
		Registry: dockerRegistry,
	}
	return r, nil
}

// Metadata returns the metadata via this registries implementation
func (r *RegistryWithoutManifestCall) Metadata(image *storage.Image) (*storage.ImageMetadata, error) {
	if image == nil {
		return nil, nil
	}
	log.Debugf("Getting metadata for image %s", image.GetName().GetFullName())

	remote := image.GetName().GetRemote()

	// If the image ID is empty, then populate with the digest from the manifest
	// This only applies in a situation with CI client
	ref := image.Id
	if ref == "" {
		ref = image.GetName().GetTag()
	}

	errorList := errorhelpers.NewErrorList(fmt.Sprintf("Error accessing %q", image.GetName().GetFullName()))
	for _, f := range manifestFuncs {
		metadata, err := f(r.Registry, remote, ref)
		if err != nil {
			errorList.AddError(err)
			continue
		}
		return metadata, nil
	}
	return nil, errorList.ToError()
}
