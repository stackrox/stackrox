package telemetry

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/authprovider/datastore"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Gather auth provider names and number of groups per auth provider.
var Gather phonehome.GatherFunc = func(ctx context.Context) (map[string]any, error) {
	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))
	props := make(map[string]any)

	providers, err := datastore.Singleton().GetAllAuthProviders(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get AuthProviders")
	}

	providerIDTypes := make(map[string]string, len(providers))
	providerTypes := set.NewSet[string]()
	providerOriginCount := map[storage.Traits_Origin]int{
		storage.Traits_DEFAULT:              0,
		storage.Traits_IMPERATIVE:           0,
		storage.Traits_DECLARATIVE:          0,
		storage.Traits_DECLARATIVE_ORPHANED: 0,
	}
	for _, provider := range providers {
		providerIDTypes[provider.GetId()] = provider.GetType()
		providerTypes.Add(provider.GetType())
		providerOriginCount[provider.GetTraits().GetOrigin()]++
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

	for origin, count := range providerOriginCount {
		props[fmt.Sprintf("Total %s Auth Providers",
			cases.Title(language.English, cases.Compact).String(origin.String()))] = count
	}
	return props, nil
}
