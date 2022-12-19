package datastore

import (
	"context"

	"github.com/pkg/errors"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// Gather auth provider names and number of groups per auth provider.
var Gather phonehome.GatherFunc = func(ctx context.Context) (map[string]any, error) {
	// WithAllAccess is required only to fetch and calculate the number of
	// auth providers and groups. It is not propagated anywhere else.
	ctx = sac.WithAllAccess(ctx)
	props := make(map[string]any)

	providers, err := Singleton().GetAllAuthProviders(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get AuthProviders")
	}

	providerIDTypes := make(map[string]string, len(providers))
	providerTypes := set.NewSet[string]()
	for _, provider := range providers {
		providerIDTypes[provider.GetId()] = provider.GetType()
		providerTypes.Add(provider.GetType())
	}
	props["Auth Providers"] = providerTypes.AsSlice()

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
		props["Total Groups of "+providerIDTypes[id]] = n
	}
	return props, nil
}
