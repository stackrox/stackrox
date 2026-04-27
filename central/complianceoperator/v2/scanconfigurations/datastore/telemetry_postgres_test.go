//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatherProfilesNilDatastore(t *testing.T) {
	props, err := GatherProfiles(nil)(context.Background())
	require.NoError(t, err)
	assert.Empty(t, props)
}

func TestGatherTailoredProfilesNilDatastore(t *testing.T) {
	props, err := GatherTailoredProfiles(nil)(context.Background())
	require.NoError(t, err)
	assert.Empty(t, props)
}

func TestGatherProfiles(t *testing.T) {
	t.Setenv(features.ComplianceEnhancements.EnvVar(), "true")

	pool := pgtest.ForT(t)
	ds := GetTestPostgresDataStore(t, pool.DB)

	writeCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance, resources.Cluster),
		),
	)

	testCases := map[string]struct {
		scanConfigs         []*storage.ComplianceOperatorScanConfigurationV2
		expectedProfileKeys []string
		excludedKeys        []string
	}{
		"no scan configs": {
			scanConfigs:         nil,
			expectedProfileKeys: nil,
		},
		"reports each regular profile name": {
			scanConfigs: []*storage.ComplianceOperatorScanConfigurationV2{
				makeScanConfig("scan-1", []profileDef{
					{"ocp4-cis", storage.ComplianceOperatorProfileV2_PROFILE},
					{"ocp4-nist", storage.ComplianceOperatorProfileV2_PROFILE},
				}),
			},
			expectedProfileKeys: []string{"Compliance Operator Profile ocp4-cis", "Compliance Operator Profile ocp4-nist"},
		},
		"excludes tailored profile names": {
			scanConfigs: []*storage.ComplianceOperatorScanConfigurationV2{
				makeScanConfig("scan-2", []profileDef{
					{"ocp4-cis", storage.ComplianceOperatorProfileV2_PROFILE},
					{"my-tp", storage.ComplianceOperatorProfileV2_TAILORED_PROFILE},
				}),
			},
			expectedProfileKeys: []string{"Compliance Operator Profile ocp4-cis"},
			excludedKeys:        []string{"Compliance Operator Profile my-tp"},
		},
		"legacy scan config without profile_refs": {
			scanConfigs: []*storage.ComplianceOperatorScanConfigurationV2{
				makeScanConfigNoRefs("scan-legacy", []string{"ocp4-cis"}),
			},
			expectedProfileKeys: []string{"Compliance Operator Profile ocp4-cis"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			for _, sc := range tc.scanConfigs {
				require.NoError(t, ds.UpsertScanConfiguration(writeCtx, sc))
			}
			t.Cleanup(func() {
				for _, sc := range tc.scanConfigs {
					_, _ = ds.DeleteScanConfiguration(writeCtx, sc.GetId())
				}
			})

			props, err := GatherProfiles(ds)(context.Background())
			require.NoError(t, err)

			for _, key := range tc.expectedProfileKeys {
				assert.Contains(t, props, key)
			}
			for _, key := range tc.excludedKeys {
				assert.NotContains(t, props, key)
			}
		})
	}
}

func TestGatherTailoredProfiles(t *testing.T) {
	t.Setenv(features.ComplianceEnhancements.EnvVar(), "true")

	pool := pgtest.ForT(t)
	ds := GetTestPostgresDataStore(t, pool.DB)

	writeCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance, resources.Cluster),
		),
	)

	testCases := map[string]struct {
		scanConfigs      []*storage.ComplianceOperatorScanConfigurationV2
		expectedTailored int
	}{
		"no scan configs": {
			scanConfigs:      nil,
			expectedTailored: 0,
		},
		"scan config with only regular profiles": {
			scanConfigs: []*storage.ComplianceOperatorScanConfigurationV2{
				makeScanConfig("scan-regular", []profileDef{
					{"ocp4-cis", storage.ComplianceOperatorProfileV2_PROFILE},
					{"ocp4-nist", storage.ComplianceOperatorProfileV2_PROFILE},
				}),
			},
			expectedTailored: 0,
		},
		"scan config with tailored profiles": {
			scanConfigs: []*storage.ComplianceOperatorScanConfigurationV2{
				makeScanConfig("scan-tailored", []profileDef{
					{"ocp4-cis", storage.ComplianceOperatorProfileV2_PROFILE},
					{"my-tailored-cis", storage.ComplianceOperatorProfileV2_TAILORED_PROFILE},
				}),
			},
			expectedTailored: 1,
		},
		"multiple scan configs with multiple tailored profiles": {
			scanConfigs: []*storage.ComplianceOperatorScanConfigurationV2{
				makeScanConfig("scan-1", []profileDef{
					{"ocp4-cis", storage.ComplianceOperatorProfileV2_PROFILE},
					{"tp-cis", storage.ComplianceOperatorProfileV2_TAILORED_PROFILE},
				}),
				makeScanConfig("scan-2", []profileDef{
					{"tp-nist", storage.ComplianceOperatorProfileV2_TAILORED_PROFILE},
					{"tp-pci", storage.ComplianceOperatorProfileV2_TAILORED_PROFILE},
				}),
			},
			expectedTailored: 3,
		},
		"scan config without profile_refs populated": {
			scanConfigs: []*storage.ComplianceOperatorScanConfigurationV2{
				makeScanConfigNoRefs("scan-no-refs", []string{"ocp4-cis"}),
			},
			expectedTailored: 0,
		},
		"scan config with unspecified kind does not count as tailored": {
			scanConfigs: []*storage.ComplianceOperatorScanConfigurationV2{
				makeScanConfig("scan-unspecified", []profileDef{
					{"ocp4-cis", storage.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED},
					{"ocp4-nist", storage.ComplianceOperatorProfileV2_PROFILE},
					{"tp-custom", storage.ComplianceOperatorProfileV2_TAILORED_PROFILE},
				}),
			},
			expectedTailored: 1,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			for _, sc := range tc.scanConfigs {
				require.NoError(t, ds.UpsertScanConfiguration(writeCtx, sc))
			}
			t.Cleanup(func() {
				for _, sc := range tc.scanConfigs {
					_, _ = ds.DeleteScanConfiguration(writeCtx, sc.GetId())
				}
			})

			props, err := GatherTailoredProfiles(ds)(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tc.expectedTailored, props["Compliance Operator Tailored Profile"])
		})
	}
}

type profileDef struct {
	name string
	kind storage.ComplianceOperatorProfileV2_OperatorKind
}

func makeScanConfig(scanName string, profiles []profileDef) *storage.ComplianceOperatorScanConfigurationV2 {
	profileNames := make([]*storage.ComplianceOperatorScanConfigurationV2_ProfileName, 0, len(profiles))
	profileRefs := make([]*storage.ComplianceOperatorScanConfigurationV2_ProfileReference, 0, len(profiles))
	for _, p := range profiles {
		profileNames = append(profileNames, &storage.ComplianceOperatorScanConfigurationV2_ProfileName{
			ProfileName: p.name,
		})
		profileRefs = append(profileRefs, &storage.ComplianceOperatorScanConfigurationV2_ProfileReference{
			Name: p.name,
			Kind: p.kind,
		})
	}
	return &storage.ComplianceOperatorScanConfigurationV2{
		Id:             uuid.NewV4().String(),
		ScanConfigName: scanName,
		Profiles:       profileNames,
		ProfileRefs:    profileRefs,
		Clusters: []*storage.ComplianceOperatorScanConfigurationV2_Cluster{
			{ClusterId: testconsts.Cluster1},
		},
	}
}

func makeScanConfigNoRefs(scanName string, profiles []string) *storage.ComplianceOperatorScanConfigurationV2 {
	profileNames := make([]*storage.ComplianceOperatorScanConfigurationV2_ProfileName, 0, len(profiles))
	for _, p := range profiles {
		profileNames = append(profileNames, &storage.ComplianceOperatorScanConfigurationV2_ProfileName{
			ProfileName: p,
		})
	}
	return &storage.ComplianceOperatorScanConfigurationV2{
		Id:             uuid.NewV4().String(),
		ScanConfigName: scanName,
		Profiles:       profileNames,
		Clusters: []*storage.ComplianceOperatorScanConfigurationV2_Cluster{
			{ClusterId: testconsts.Cluster1},
		},
	}
}
