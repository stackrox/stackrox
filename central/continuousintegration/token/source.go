package token

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	_ tokens.Source = (*continuousIntegrationSource)(nil)

	onceSources sync.Once
	sources     map[storage.ContinuousIntegrationType]tokens.Source
)

type continuousIntegrationSource struct {
	integrationType storage.ContinuousIntegrationType
}

// SingletonSourceForContinuousIntegration returns a singleton tokens.Source for the specific
// storage.ContinuousIntegrationType.
func SingletonSourceForContinuousIntegration(integrationType storage.ContinuousIntegrationType) tokens.Source {
	onceSources.Do(func() {
		gitHubSource := &continuousIntegrationSource{integrationType: storage.ContinuousIntegrationType_GITHUB_ACTIONS}
		sources = map[storage.ContinuousIntegrationType]tokens.Source{
			storage.ContinuousIntegrationType_GITHUB_ACTIONS: gitHubSource,
		}
	})
	return sources[integrationType]
}

func (c continuousIntegrationSource) Validate(_ context.Context, _ *tokens.Claims) error {
	return nil
}

func (c continuousIntegrationSource) ID() string {
	return c.integrationType.String()
}
