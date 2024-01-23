package datastore

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Gather image integration types.
// Current properties we gather:
// "Total Image Integrations"
// "Total <integration type> Image Integrations"
var Gather phonehome.GatherFunc = func(ctx context.Context) (map[string]any, error) {
	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
	props := make(map[string]any)

	integrations, err := Singleton().GetImageIntegrations(ctx, &v1.GetImageIntegrationsRequest{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get image integrations")
	}

	// Can safely ignore the error here since we already fetched integrations.
	_ = phonehome.AddTotal(ctx, props, "Image Integrations", func(_ context.Context) (int, error) {
		return len(integrations), nil
	})

	totalCount := map[string]int{}
	stsCount := map[string]int{}

	for _, ii := range integrations {
		iiType := ii.GetType()
		totalCount[iiType]++

		if (iiType == types.ECRType && ii.GetEcr().GetUseIam()) ||
			(iiType == types.ArtifactRegistryType && ii.GetGoogle().GetWifEnabled()) ||
			(iiType == types.GoogleType && ii.GetGoogle().GetWifEnabled()) ||
			(iiType == types.AzureType && ii.GetAzure().GetWifEnabled()) {
			stsCount[iiType]++
		}
	}

	for iiType, count := range totalCount {
		props[fmt.Sprintf("Total %s Image Integrations",
			cases.Title(language.English, cases.Compact).String(iiType))] = count
	}

	for iiType, count := range stsCount {
		props[fmt.Sprintf("Total STS enabled %s Image Integrations",
			cases.Title(language.English, cases.Compact).String(iiType))] = count
	}

	return props, nil
}
