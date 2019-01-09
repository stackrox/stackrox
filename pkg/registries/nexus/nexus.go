package nexus

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
)

var (
	log = logging.LoggerForModule()
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.ImageRegistry, error)) {
	return "nexus", func(integration *storage.ImageIntegration) (types.ImageRegistry, error) {
		reg, err := newRegistry(integration)
		return reg, err
	}
}

// Registry is the basic docker registry implementation
type Registry struct {
	*docker.Registry

	manifestFuncs []func(remote, ref string) (*storage.ImageMetadata, error)
}

func newRegistry(integration *storage.ImageIntegration) (*Registry, error) {
	dockerRegistry, err := docker.NewDockerRegistry(integration)
	if err != nil {
		return nil, err
	}

	r := &Registry{
		Registry: dockerRegistry,
	}

	r.manifestFuncs = []func(remote, ref string) (*storage.ImageMetadata, error){
		r.HandleV2ManifestList,
		r.HandleV2Manifest,
		r.HandleV1SignedManifest,
		r.HandleV1Manifest,
	}
	return r, nil
}

// Metadata returns the metadata via this registries implementation
func (r *Registry) Metadata(image *storage.Image) (*storage.ImageMetadata, error) {
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
	for _, f := range r.manifestFuncs {
		metadata, err := f(remote, ref)
		if err != nil {
			errorList.AddError(err)
			continue
		}
		return metadata, nil
	}
	return nil, errorList.ToError()
}
