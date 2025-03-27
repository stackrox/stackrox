package datastore

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/externalbackups/plugins/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Gather external backup types.
// Current properties we gather:
// "Total External Backups"
// "Total <backup type> External Backups"
var Gather phonehome.GatherFunc = func(ctx context.Context) (map[string]any, error) {
	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
	props := make(map[string]any)

	backupTypesCount := map[string]int{
		types.S3Type:           0,
		types.S3CompatibleType: 0,
		types.GCSType:          0,
	}

	cloudCredentialsEnabledBackupsCount := map[string]int{
		types.S3Type:  0,
		types.GCSType: 0,
	}

	count := 0
	err := Singleton().ProcessBackups(ctx, func(backup *storage.ExternalBackup) error {
		count++
		backupTypesCount[backup.GetType()]++

		if backup.GetType() == types.S3Type && backup.GetS3().GetUseIam() {
			cloudCredentialsEnabledBackupsCount[types.S3Type]++
		}

		if backup.GetType() == types.GCSType && backup.GetGcs().GetUseWorkloadId() {
			cloudCredentialsEnabledBackupsCount[types.GCSType]++
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get external backups")
	}

	// Can safely ignore the error here since we already fetched backups.
	_ = phonehome.AddTotal(ctx, props, "External Backups", phonehome.Constant(count))

	for backupType, count := range backupTypesCount {
		props[fmt.Sprintf("Total %s External Backups",
			cases.Title(language.English, cases.Compact).String(backupType))] = count
	}

	for cloudCredentialsType, count := range cloudCredentialsEnabledBackupsCount {
		props[fmt.Sprintf("Total STS enabled %s External Backups",
			cases.Title(language.English, cases.Compact).String(cloudCredentialsType))] = count
	}

	return props, nil
}
