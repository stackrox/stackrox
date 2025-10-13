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

	notifierTypesCount := map[string]int{
		pkgNotifiers.AWSSecurityHubType:    0,
		pkgNotifiers.CSCCType:              0,
		pkgNotifiers.EmailType:             0,
		pkgNotifiers.GenericType:           0,
		pkgNotifiers.JiraType:              0,
		pkgNotifiers.MicrosoftSentinelType: 0,
		pkgNotifiers.PagerDutyType:         0,
		pkgNotifiers.SlackType:             0,
		pkgNotifiers.SplunkType:            0,
		pkgNotifiers.SumoLogicType:         0,
		pkgNotifiers.SyslogType:            0,
		pkgNotifiers.TeamsType:             0,
	}

	cloudCredentialsEnabledNotifiersCount := map[string]int{
		pkgNotifiers.AWSSecurityHubType:    0,
		pkgNotifiers.CSCCType:              0,
		pkgNotifiers.MicrosoftSentinelType: 0,
	}

	count := 0
	err := Singleton().ForEachNotifier(ctx, func(notifier *storage.Notifier) error {
		count++
		notifierTypesCount[notifier.GetType()]++

		if notifier.GetType() == pkgNotifiers.AWSSecurityHubType && notifier.GetAwsSecurityHub().GetCredentials().GetStsEnabled() {
			cloudCredentialsEnabledNotifiersCount[pkgNotifiers.AWSSecurityHubType]++
		}

		if notifier.GetType() == pkgNotifiers.CSCCType && notifier.GetCscc().GetWifEnabled() {
			cloudCredentialsEnabledNotifiersCount[pkgNotifiers.CSCCType]++
		}

		if notifier.GetType() == pkgNotifiers.MicrosoftSentinelType && notifier.GetMicrosoftSentinel().GetWifEnabled() {
			cloudCredentialsEnabledNotifiersCount[pkgNotifiers.MicrosoftSentinelType]++
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get notifiers")
	}

	// Can safely ignore the error here since we already fetched notifiers.
	_ = phonehome.AddTotal(ctx, props, "Notifiers", phonehome.Constant(count))

	titleCase := cases.Title(language.English, cases.Compact).String

	for notifierType, count := range notifierTypesCount {
		props[fmt.Sprintf("Total %s Notifiers", titleCase(notifierType))] = count
	}

	for cloudCredentialsType, count := range cloudCredentialsEnabledNotifiersCount {
		props[fmt.Sprintf("Total STS enabled %s Notifiers", titleCase(cloudCredentialsType))] = count
	}

	return props, nil
}
