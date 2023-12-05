package datastore

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/externalbackups"
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

	backups, err := Singleton().ListBackups(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get external backups")
	}

	// Can safely ignore the error here since we already fetched backups.
	_ = phonehome.AddTotal(ctx, props, "External Backups", func(_ context.Context) (int, error) {
		return len(backups), nil
	})

	backupTypesCount := map[string]int{
		externalbackups.S3Type:  0,
		externalbackups.GCSType: 0,
	}

	cloudCredentialsEnabledBackupsCount := map[string]int{
		externalbackups.S3Type:  0,
		externalbackups.GCSType: 0,
	}

	for _, backup := range backups {
		backupTypesCount[backup.GetType()]++

		if backup.GetType() == externalbackups.S3Type && backup.GetS3().GetUseIam() {
			cloudCredentialsEnabledBackupsCount[externalbackups.S3Type]++
		}

		if backup.GetType() == externalbackups.GCSType && backup.GetGcs().GetUseWorkloadId() {
			cloudCredentialsEnabledBackupsCount[externalbackups.GCSType]++
		}
	}

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
