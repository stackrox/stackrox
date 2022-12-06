package datastore

import (
	"context"

	"github.com/pkg/errors"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// Gather auth provider names and number of groups per auth provider.
func Gather(ctx context.Context) (phonehome.Properties, error) {
	// WithAllAccess is required only to fetch and calculate the number of auth
	// providers and groups. It is not propagated anywhere else.
	ctx = sac.WithAllAccess(ctx)
	props := make(phonehome.Properties)

	providers, err := Singleton().GetAllAuthProviders(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get AuthProviders")
	}

	providerIDNames := make(map[string]string, len(providers))
	providerNames := make([]string, 0, len(providers))
	for _, provider := range providers {
		providerIDNames[provider.GetId()] = provider.GetName()
		providerNames = append(providerNames, provider.GetName())
	}
	props["Auth Providers"] = providerNames

	groups, err := groupDataStore.Singleton().GetAll(ctx)
	if err != nil {
		return props, errors.Wrap(err, "failed to get Groups")
	}

	providerGroups := make(map[string]int)
	for _, group := range groups {
		id := group.GetProps().GetAuthProviderId()
		providerGroups[id] = providerGroups[id] + 1
	}

	for id, n := range providerGroups {
		props["Total Groups of "+providerIDNames[id]] = n
	}
	return props, nil
}
