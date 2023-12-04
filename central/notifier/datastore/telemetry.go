package datastore

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	pkgNotifiers "github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Gather notifier types.
// Current properties we gather:
// "Total Notifiers"
// "Total <notifier type> Notifiers"
var Gather phonehome.GatherFunc = func(ctx context.Context) (map[string]any, error) {
	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))
	props := make(map[string]any)

	notifiers, err := Singleton().GetNotifiers(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get notifiers")
	}

	// Can safely ignore the error here since we already fetched notifiers.
	_ = phonehome.AddTotal(ctx, props, "Notifiers", func(_ context.Context) (int, error) {
		return len(notifiers), nil
	})

	notifierTypesCount := map[string]int{
		pkgNotifiers.AWSSecurityHubType: 0,
		pkgNotifiers.CSCCType:           0,
		pkgNotifiers.EmailType:          0,
		pkgNotifiers.GenericType:        0,
		pkgNotifiers.JiraType:           0,
		pkgNotifiers.PagerDutyType:      0,
		pkgNotifiers.SlackType:          0,
		pkgNotifiers.SplunkType:         0,
		pkgNotifiers.SumoLogicType:      0,
		pkgNotifiers.SyslogType:         0,
		pkgNotifiers.TeamsType:          0,
	}

	for _, notifier := range notifiers {
		notifierTypesCount[notifier.GetType()]++
	}

	for notifierType, count := range notifierTypesCount {
		props[fmt.Sprintf("Total %s Notifiers",
			cases.Title(language.English, cases.Compact).String(notifierType))] = count
	}

	return props, nil
}
