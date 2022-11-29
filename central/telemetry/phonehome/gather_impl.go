package phonehome

import (
	"context"

	apDataStore "github.com/stackrox/rox/central/authprovider/datastore"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	roles "github.com/stackrox/rox/central/role/datastore"
	si "github.com/stackrox/rox/central/signatureintegration/datastore"
	"github.com/stackrox/rox/pkg/sac"
)

func addTotal[T any](ctx context.Context, props map[string]any, key string, f func(context.Context) ([]*T, error)) {
	if ps, err := f(ctx); err != nil {
		log.Errorf("Failed to get %s: %v", key, err)
	} else {
		props["Total "+key] = len(ps)
	}
}

func gather(ctx context.Context) map[string]any {
	log.Debug("Starting telemetry data collection.")
	defer log.Debug("Done with telemetry data collection.")

	totals := make(map[string]any)
	rs := roles.Singleton()

	ctx = sac.WithAllAccess(ctx)
	addTotal(ctx, totals, "PermissionSets", rs.GetAllPermissionSets)
	addTotal(ctx, totals, "Roles", rs.GetAllRoles)
	addTotal(ctx, totals, "Access Scopes", rs.GetAllAccessScopes)
	addTotal(ctx, totals, "Signature Integrations", si.Singleton().GetAllSignatureIntegrations)

	groups, err := groupDataStore.Singleton().GetAll(ctx)
	if err != nil {
		log.Error("Failed to get Groups: ", err)
		return nil
	}
	providers, err := apDataStore.Singleton().GetAllAuthProviders(ctx)
	if err != nil {
		log.Error("Failed to get AuthProviders: ", err)
		return nil
	}

	providerIDNames := make(map[string]string)
	providerNames := make([]string, len(providers))
	for _, provider := range providers {
		providerIDNames[provider.GetId()] = provider.GetName()
		providerNames = append(providerNames, provider.GetName())
	}
	totals["Auth Providers"] = providerNames

	providerGroups := make(map[string]int)
	for _, group := range groups {
		id := group.GetProps().GetAuthProviderId()
		providerGroups[id] = providerGroups[id] + 1
	}

	for id, n := range providerGroups {
		totals["Total Groups of "+providerIDNames[id]] = n
	}
	return totals
}
