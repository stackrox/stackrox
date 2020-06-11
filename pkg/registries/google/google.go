package google

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	username = "_json_key"
)

type googleRegistry struct {
	types.ImageRegistry
	project string
}

// Match overrides the underlying Match function in types.ImageRegistry because our google registries are scoped by
// GCP projects
func (g *googleRegistry) Match(image *storage.ImageName) bool {
	if stringutils.GetUpTo(image.GetRemote(), "/") != g.project {
		return false
	}
	return g.ImageRegistry.Match(image)
}

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.ImageRegistry, error)) {
	return "google", func(integration *storage.ImageIntegration) (types.ImageRegistry, error) {
		reg, err := newRegistry(integration)
		return reg, err
	}
}

func validate(google *storage.GoogleConfig) error {
	errorList := errorhelpers.NewErrorList("Google Validation")
	if google.GetEndpoint() == "" {
		errorList.AddString("Endpoint must be specified for Google registry (e.g. gcr.io, us.gcr.io, eu.gcr.io)")
	}
	if google.GetServiceAccount() == "" {
		errorList.AddString("Service account must be specified for Google registry")
	}
	return errorList.ToError()
}

func newRegistry(integration *storage.ImageIntegration) (types.ImageRegistry, error) {
	config := integration.GetGoogle()
	if config == nil {
		return nil, errors.New("Google configuration required")
	}
	if err := validate(config); err != nil {
		return nil, err
	}
	cfg := docker.Config{
		Username: username,
		Password: config.GetServiceAccount(),
		Endpoint: config.GetEndpoint(),
	}
	reg, err := docker.NewDockerRegistryWithConfig(cfg, integration)
	if err != nil {
		return nil, err
	}
	return &googleRegistry{
		ImageRegistry: reg,
		project:       strings.ToLower(config.GetProject()),
	}, nil
}
