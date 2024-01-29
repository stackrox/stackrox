package google

import (
	"context"
	"strings"

	artifactv1 "cloud.google.com/go/artifactregistry/apiv1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudproviders/gcp/auth"
	"github.com/stackrox/rox/pkg/cloudproviders/gcp/utils"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/stringutils"
	"golang.org/x/oauth2"
)

var _ types.Registry = (*googleRegistry)(nil)

type googleRegistry struct {
	types.Registry
	project string
}

// Match overrides the underlying Match function in types.Registry because our google registries are scoped by
// GCP projects.
func (g *googleRegistry) Match(image *storage.ImageName) bool {
	if stringutils.GetUpTo(image.GetRemote(), "/") != g.project {
		return false
	}
	return g.Registry.Match(image)
}

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, types.Creator) {
	return types.GoogleType,
		func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
			cfg := types.ApplyCreatorOptions(options...)
			return NewRegistry(integration, false, cfg.GetGCPTokenManager())
		}
}

// CreatorWithoutRepoList provides the type and registries.Creator to add to the registries Registry.
// Populating the internal repo list will be disabled.
func CreatorWithoutRepoList() (string, types.Creator) {
	return types.GoogleType,
		func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
			cfg := types.ApplyCreatorOptions(options...)
			return NewRegistry(integration, true, cfg.GetGCPTokenManager())
		}
}

func validate(google *storage.GoogleConfig) error {
	errorList := errorhelpers.NewErrorList("Google Validation")
	if google.GetEndpoint() == "" {
		errorList.AddString("Endpoint must be specified for Google registry (e.g. gcr.io, us.gcr.io, eu.gcr.io)")
	}
	if !google.GetWifEnabled() && google.GetServiceAccount() == "" {
		errorList.AddString("Service account must be specified for Google registry")
	}
	if google.GetWifEnabled() && !features.CloudCredentials.Enabled() {
		return errors.Errorf("cannot use workload identities without the feature flag %s being enabled", features.CloudCredentials.Name())
	}
	return errorList.ToError()
}

// NewRegistry creates an image integration based on the Google config. It also checks against
// the specified Google project as a part of the registry match.
func NewRegistry(integration *storage.ImageIntegration, disableRepoList bool, manager auth.STSTokenManager) (types.Registry, error) {
	config := integration.GetGoogle()
	if config == nil {
		return nil, errors.New("Google configuration required")
	}
	if err := validate(config); err != nil {
		return nil, err
	}

	dockerConfig := &docker.Config{
		Endpoint:        config.GetEndpoint(),
		DisableRepoList: disableRepoList,
	}
	var (
		tokenSource oauth2.TokenSource
		err         error
	)
	if features.CloudCredentials.Enabled() {
		tokenSource, err = utils.CreateTokenSourceFromConfigWithManager(
			context.Background(),
			manager,
			[]byte(config.GetServiceAccount()),
			config.GetWifEnabled(),
			artifactv1.DefaultAuthScopes()...,
		)
	} else {
		tokenSource, err = utils.CreateTokenSourceFromConfig(
			context.Background(),
			[]byte(config.GetServiceAccount()),
			config.GetWifEnabled(),
			artifactv1.DefaultAuthScopes()...,
		)
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to create token source")
	}
	dockerConfig.Transport = newGoogleTransport(dockerConfig, tokenSource)
	reg, err := docker.NewDockerRegistryWithConfig(dockerConfig, integration)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create docker registry")
	}
	return &googleRegistry{
		Registry: reg,
		project:  strings.ToLower(config.GetProject()),
	}, nil
}
