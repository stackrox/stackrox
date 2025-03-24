package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var Gather phonehome.GatherFunc = func(ctx context.Context) (map[string]any, error) {
	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))

	props := make(map[string]any)

	configs, err := Singleton().ListAuthM2MConfigs(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get machine-to-machine configurations")
	}

	_ = phonehome.AddTotal(ctx, props, "Machine-To-Machine Configurations", phonehome.Len(configs))

	countByType := map[string]int{
		storage.AuthMachineToMachineConfig_GITHUB_ACTIONS.String(): 0,
		storage.AuthMachineToMachineConfig_GENERIC.String():        0,
	}

	for _, config := range configs {
		countByType[config.GetType().String()]++
	}

	titleCase := cases.Title(language.English, cases.Compact).String
	for configType, count := range countByType {
		_ = phonehome.AddTotal(ctx, props,
			titleCase(configType)+" Machine-To-Machine Configurations",
			phonehome.Constant(count))
	}

	return props, nil
}
