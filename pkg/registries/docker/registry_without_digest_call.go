package docker

import (
	"fmt"

	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/registries/types"
)

// manifestFuncs explicitly lists the container image manifest handlers.
// Note: Any updates here must be accompanied by updates to docker.go.
var manifestFuncs = []func(registry *Registry, remote, ref string) (*storage.ImageMetadata, error){
	HandleV2ManifestList,
	HandleV2Manifest,
	HandleOCIImageIndex,
	HandleOCIManifest,
	HandleV1SignedManifest,
	HandleV1Manifest,
}

// RegistryWithoutManifestCall is the basic docker registry implementation without the manifest digest call
type RegistryWithoutManifestCall struct {
	*Registry
}

// NewRegistryWithoutManifestCall creates a new basic docker registry without a manifest digest call
func NewRegistryWithoutManifestCall(integration *storage.ImageIntegration,
	disableRepoList bool, metricsHandler *types.MetricsHandler,
) (*RegistryWithoutManifestCall, error) {
	dockerRegistry, err := NewDockerRegistry(integration, disableRepoList, metricsHandler)
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

	ref := image.GetName().GetTag()
	// Prefer the image digest over the tag, if it exists.
	if dig := image.GetId(); dig != "" {
		if _, err := digest.Parse(dig); err != nil {
			return nil, errors.Wrapf(err, "invalid image id: %s", dig)
		}
		ref = dig
	}

	errorList := errorhelpers.NewErrorList(fmt.Sprintf("Error accessing %q", image.GetName().GetFullName()))

	if features.AttemptManifestDigest.Enabled() {
		// Try to pull metadata in a standard way, fallback on failure.
		metadata, err := r.Registry.Metadata(image)
		if err == nil {
			return metadata, nil
		}
		errorList.AddError(err)
		log.Debugf("Falling back to trying each handler individually for %q due to: %v", image.GetName().GetFullName(), err)
	}

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
