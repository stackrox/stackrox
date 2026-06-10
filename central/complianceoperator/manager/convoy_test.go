//go:build sql_integration

package manager

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/central/compliance/datastore/mocks"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	checkResultsDatastore "github.com/stackrox/rox/central/complianceoperator/checkresults/datastore"
	checkResultsStore "github.com/stackrox/rox/central/complianceoperator/checkresults/store/postgres"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/profiles/datastore"
	profileStore "github.com/stackrox/rox/central/complianceoperator/profiles/store/postgres"
	rulesDatastore "github.com/stackrox/rox/central/complianceoperator/rules/datastore"
	rulesStore "github.com/stackrox/rox/central/complianceoperator/rules/store/postgres"
	scansDatastore "github.com/stackrox/rox/central/complianceoperator/scans/datastore"
	scansStore "github.com/stackrox/rox/central/complianceoperator/scans/store/postgres"
	scanSettingBindingDatastore "github.com/stackrox/rox/central/complianceoperator/scansettingbinding/datastore"
	scanSettingBindingStore "github.com/stackrox/rox/central/complianceoperator/scansettingbinding/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newManagerTB(tb testing.TB) *managerImpl {
	tb.Helper()
	registry, err := standards.NewRegistry(framework.RegistrySingleton(), metadata.AllStandards...)
	require.NoError(tb, err)

	db := pgtest.ForT(tb)
	prof := profileStore.New(db)
	ssb := scanSettingBindingStore.New(db)
	rules := rulesStore.New(db)
	rulesDS, err := rulesDatastore.NewDatastore(rules)
	require.NoError(tb, err)
	scans := scansStore.New(db)
	scansDS := scansDatastore.NewDatastore(scans)
	checks := checkResultsStore.New(db)

	ctrl := gomock.NewController(tb)
	compliance := mocks.NewMockDataStore(ctrl)

	mgr, err := NewManager(registry, profileDatastore.NewDatastore(prof), scansDS, scanSettingBindingDatastore.NewDatastore(ssb), rulesDS, checkResultsDatastore.NewDatastore(checks), compliance)
	require.NoError(tb, err)
	return mgr.(*managerImpl)
}

func makeConvoyRule(clusterID, ruleName string) *storage.ComplianceOperatorRule {
	return &storage.ComplianceOperatorRule{
		Id:   uuid.NewV4().String(),
		Name: fmt.Sprintf("%s-ext", ruleName),
		Annotations: map[string]string{
			v1alpha1.RuleIDAnnotationKey: ruleName,
		},
		ClusterId: clusterID,
		Title:     fmt.Sprintf("Rule %s", ruleName),
	}
}

func makeConvoyProfile(clusterID, profileName string, ruleNames []string) *storage.ComplianceOperatorProfile {
	rules := make([]*storage.ComplianceOperatorProfile_Rule, len(ruleNames))
	for i, name := range ruleNames {
		rules[i] = &storage.ComplianceOperatorProfile_Rule{Name: name}
	}
	return &storage.ComplianceOperatorProfile{
		Id:          uuid.NewV4().String(),
		Name:        profileName,
		ClusterId:   clusterID,
		Description: fmt.Sprintf("Profile %s on cluster %s", profileName, clusterID),
		Annotations: map[string]string{
			v1alpha1.ProductTypeAnnotation: string(v1alpha1.ScanTypePlatform),
		},
		Rules: rules,
	}
}

func seedProfiles(tb testing.TB, mgr *managerImpl, numClusters, profilesPerCluster int, ruleNames []string) {
	tb.Helper()
	for c := 0; c < numClusters; c++ {
		for p := 0; p < profilesPerCluster; p++ {
			profile := makeConvoyProfile(fmt.Sprintf("cluster-%d", c), fmt.Sprintf("profile-%d", p), ruleNames)
			require.NoError(tb, mgr.AddProfile(profile))
		}
	}
}

// TestConcurrentAddProfileCompletesWithoutConvoy verifies that without the old
// startup Walk competing for resources, concurrent AddProfile calls from sensor
// reconnects complete within a reasonable deadline.
//
// Background: The old NewManager did a Walk of all profiles at startup, calling
// addProfileNoLock for each — which itself did another Walk. This O(N²) pattern
// combined with concurrent sensor AddProfile calls caused a lock convoy that
// exhausted context deadlines and panicked Central. See the customer stack trace:
//
//	panic: context deadline exceeded
//	    iterating over rows
//	    RunCursorQueryForSchemaFn → walkByQuery → Walk →
//	    complianceoperator/profiles/datastore.Walk →
//	    complianceoperator/manager.NewManager
//
// The fix removes the startup Walk (sensors repopulate on reconnect) and
// throttles concurrent profile/rule pipeline operations via semaphore.
func TestConcurrentAddProfileCompletesWithoutConvoy(t *testing.T) {
	mgr := newManagerTB(t)

	ruleNames := []string{"rule-0", "rule-1", "rule-2"}
	for _, name := range ruleNames {
		require.NoError(t, mgr.AddRule(makeConvoyRule("seed", name)))
	}

	seedProfiles(t, mgr, 50, 10, ruleNames)
	t.Logf("Seeded 500 profiles")

	deadline := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()

	sensorClusters := 20
	var wg sync.WaitGroup
	errs := make([]error, sensorClusters)
	wg.Add(sensorClusters)

	start := time.Now()
	for c := 0; c < sensorClusters; c++ {
		go func(clusterIdx int) {
			defer wg.Done()
			clusterID := fmt.Sprintf("reconnecting-%d", clusterIdx)
			for p := 0; p < 10; p++ {
				if ctx.Err() != nil {
					errs[clusterIdx] = ctx.Err()
					return
				}
				profile := makeConvoyProfile(clusterID, fmt.Sprintf("profile-%d", p), ruleNames)
				if err := mgr.AddProfile(profile); err != nil {
					errs[clusterIdx] = err
					return
				}
			}
		}(c)
	}
	wg.Wait()
	elapsed := time.Since(start)
	t.Logf("20 clusters × 10 profiles completed in %v", elapsed)

	for i, err := range errs {
		assert.NoError(t, err, "sensor cluster %d should not error", i)
	}
	assert.Less(t, elapsed, deadline, "should complete well within deadline")
}

// BenchmarkAddProfileConvoy measures concurrent AddProfile throughput.
func BenchmarkAddProfileConvoy(b *testing.B) {
	for _, numClusters := range []int{5, 10, 20} {
		b.Run(fmt.Sprintf("clusters_%d", numClusters), func(b *testing.B) {
			mgr := newManagerTB(b)

			ruleNames := []string{"rule-0", "rule-1", "rule-2"}
			for _, name := range ruleNames {
				require.NoError(b, mgr.AddRule(makeConvoyRule("seed", name)))
			}

			seedProfiles(b, mgr, 5, 10, ruleNames)

			profilesPerCluster := 10

			b.ResetTimer()
			for b.Loop() {
				var wg sync.WaitGroup
				wg.Add(numClusters)
				for c := 0; c < numClusters; c++ {
					go func(clusterIdx int) {
						defer wg.Done()
						clusterID := fmt.Sprintf("reconnecting-%d", clusterIdx)
						for p := 0; p < profilesPerCluster; p++ {
							profile := makeConvoyProfile(clusterID, fmt.Sprintf("profile-%d", p), ruleNames)
							_ = mgr.AddProfile(profile)
						}
					}(c)
				}
				wg.Wait()
			}
		})
	}
}
