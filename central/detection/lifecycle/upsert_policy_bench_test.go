//go:build sql_integration

package lifecycle

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	centralDetection "github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/central/detection/alertmanager"
	runtimeMocks "github.com/stackrox/rox/central/detection/runtime/mocks"
	reprocessorMocks "github.com/stackrox/rox/central/reprocessor/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/detection"
	deploytimeDetect "github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/fixtures"
	notifierMocks "github.com/stackrox/rox/pkg/notifier/mocks"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// noopPolicySet wraps a real PolicySet and adds the RemoveNotifier method
// required by central/detection.PolicySet. UpsertPolicy always succeeds
// so we measure DB cost, not policy compilation.
type noopPolicySet struct {
	detection.PolicySet
}

func (noopPolicySet) UpsertPolicy(_ *storage.Policy) error { return nil }
func (noopPolicySet) RemoveNotifier(_ string) error        { return nil }

var _ centralDetection.PolicySet = noopPolicySet{}

// stubBuildtimeDetector satisfies buildtime.Detector using a noopPolicySet.
type stubBuildtimeDetector struct{ ps detection.PolicySet }

func (s *stubBuildtimeDetector) PolicySet() detection.PolicySet { return s.ps }
func (s *stubBuildtimeDetector) Detect(_ context.Context, _ *storage.Image, _ ...detection.FilterOption) ([]*storage.Alert, error) {
	return nil, nil
}

// stubDeploytimeDetector satisfies deploytime.Detector using a noopPolicySet.
type stubDeploytimeDetector struct{ ps detection.PolicySet }

func (s *stubDeploytimeDetector) PolicySet() detection.PolicySet { return s.ps }
func (s *stubDeploytimeDetector) Detect(_ context.Context, _ booleanpolicy.EnhancedDeployment, _ ...deploytimeDetect.DetectOption) ([]*storage.Alert, error) {
	return nil, nil
}

// Policies that apply at both DEPLOY and RUNTIME, matching the pattern seen
// in long-running clusters where deploy-time alerts accumulate for policies
// that also have a runtime lifecycle stage.
var benchPolicies = []struct {
	id   string
	name string
}{
	{"16c95922-08c4-41b6-a721-dc4b2a806632", "bench-policy-1"},
	{"2db9a279-2aec-4618-a85d-7f1bdf4911b1", "bench-policy-2"},
	{"dce17697-1b72-49d2-b18a-05d893cd9368", "bench-policy-3"},
	{"89cae2e6-0cb7-4329-8692-c2c3717c1237", "bench-policy-4"},
	{"a919ccaf-6b43-4160-ac5d-a405e1440a41", "bench-policy-5"},
}

type benchEnv struct {
	ctx     context.Context
	manager *managerImpl
}

func setupBenchEnv(b *testing.B, alertsPerPolicy int) *benchEnv {
	b.Helper()

	testDB := pgtest.ForT(b)
	ctx := sac.WithAllAccess(context.Background())
	ds := alertDataStore.GetTestPostgresDataStore(b, testDB.DB)

	mockCtrl := gomock.NewController(b)
	notifier := notifierMocks.NewMockProcessor(mockCtrl)
	notifier.EXPECT().ProcessAlert(gomock.Any(), gomock.Any()).AnyTimes()

	rp := reprocessorMocks.NewMockLoop(mockCtrl)
	rp.EXPECT().ReprocessRiskForDeployments(gomock.Any()).AnyTimes()

	buildPS := detection.NewPolicySet(nil, nil)
	deployPS := detection.NewPolicySet(nil, nil)

	// Use a noopPolicySet so UpsertPolicy always succeeds without requiring
	// compilable runtime policy sections. The benchmark measures the DB
	// query cost of AlertAndNotify, not policy compilation.
	runtimePS := noopPolicySet{PolicySet: detection.NewPolicySet(nil, nil)}

	runtimeDet := runtimeMocks.NewMockDetector(mockCtrl)
	runtimeDet.EXPECT().PolicySet().Return(runtimePS).AnyTimes()
	runtimeDet.EXPECT().DeploymentInactive(gomock.Any()).Return(false).AnyTimes()
	runtimeDet.EXPECT().DeploymentWhitelistedForPolicy(gomock.Any(), gomock.Any()).Return(false).AnyTimes()

	am := alertmanager.New(notifier, ds, runtimeDet)

	// Seed alerts: deploy-time ACTIVE and ATTEMPTED alerts for each policy,
	// matching the pattern observed in the long-running cluster.
	for _, pol := range benchPolicies {
		for i := 0; i < alertsPerPolicy; i++ {
			a := fixtures.GetAlertWithID(uuid.NewV4().String())
			a.Policy = fixtures.GetPolicy()
			a.Policy.Id = pol.id
			a.Policy.Name = pol.name
			a.Policy.LifecycleStages = []storage.LifecycleStage{
				storage.LifecycleStage_DEPLOY,
				storage.LifecycleStage_RUNTIME,
			}
			a.LifecycleStage = storage.LifecycleStage_DEPLOY

			if i%4 == 0 {
				a.State = storage.ViolationState_ATTEMPTED
			} else {
				a.State = storage.ViolationState_ACTIVE
			}

			require.NoError(b, ds.UpsertAlert(ctx, a))
		}
	}

	m := &managerImpl{
		buildTimeDetector:         &stubBuildtimeDetector{ps: noopPolicySet{PolicySet: buildPS}},
		deployTimeDetector:        &stubDeploytimeDetector{ps: noopPolicySet{PolicySet: deployPS}},
		runtimeDetector:           runtimeDet,
		alertManager:              am,
		reprocessor:               rp,
		removedOrDisabledPolicies: set.NewStringSet(),
	}

	return &benchEnv{ctx: ctx, manager: m}
}

func makeRuntimePolicy(id, name string) *storage.Policy {
	p := fixtures.GetPolicy()
	p.Id = id
	p.Name = name
	p.LifecycleStages = []storage.LifecycleStage{
		storage.LifecycleStage_DEPLOY,
		storage.LifecycleStage_RUNTIME,
	}
	return p
}

// BenchmarkUpsertPolicyLockContention measures reader lock wait time during
// UpsertPolicy with real postgres-backed alert queries.
//
// "Before": AlertAndNotify runs inside the write lock (old behavior).
// "After": AlertAndNotify runs outside the write lock (new behavior).
func BenchmarkUpsertPolicyLockContention(b *testing.B) {
	alertCounts := []int{100, 1000, 5000}

	for _, perPolicy := range alertCounts {
		b.Run(fmt.Sprintf("alertsPerPolicy=%d", perPolicy), func(b *testing.B) {
			env := setupBenchEnv(b, perPolicy)
			policy := makeRuntimePolicy(benchPolicies[0].id, benchPolicies[0].name)

			// upsertPolicyOld simulates the original code: AlertAndNotify
			// executes while holding the write lock.
			upsertPolicyOld := func() error {
				env.manager.policyAlertsLock.Lock()
				defer env.manager.policyAlertsLock.Unlock()

				_ = env.manager.runtimeDetector.PolicySet().UpsertPolicy(policy)

				modifiedDeployments, err := env.manager.alertManager.AlertAndNotify(
					lifecycleMgrCtx, nil,
					alertmanager.WithPolicyID(policy.GetId()))
				if err != nil {
					return err
				}
				if modifiedDeployments.Cardinality() > 0 {
					env.manager.reprocessor.ReprocessRiskForDeployments(modifiedDeployments.AsSlice()...)
				}
				return nil
			}

			b.Run("Before_LockIncludesAlertAndNotify", func(b *testing.B) {
				var totalReaderWait atomic.Int64

				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var wg sync.WaitGroup
					wg.Add(1)

					go func() {
						defer wg.Done()
						time.Sleep(100 * time.Microsecond)
						start := time.Now()
						env.manager.policyAlertsLock.RLock()
						totalReaderWait.Add(time.Since(start).Microseconds())
						env.manager.policyAlertsLock.RUnlock()
					}()

					require.NoError(b, upsertPolicyOld())
					wg.Wait()
				}
				b.ReportMetric(float64(totalReaderWait.Load())/float64(b.N), "avg-reader-wait-µs")
			})

			b.Run("After_LockExcludesAlertAndNotify", func(b *testing.B) {
				var totalReaderWait atomic.Int64

				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var wg sync.WaitGroup
					wg.Add(1)

					go func() {
						defer wg.Done()
						time.Sleep(100 * time.Microsecond)
						start := time.Now()
						env.manager.policyAlertsLock.RLock()
						totalReaderWait.Add(time.Since(start).Microseconds())
						env.manager.policyAlertsLock.RUnlock()
					}()

					require.NoError(b, env.manager.UpsertPolicy(policy))
					wg.Wait()
				}
				b.ReportMetric(float64(totalReaderWait.Load())/float64(b.N), "avg-reader-wait-µs")
			})
		})
	}
}

// BenchmarkPolicyInjectionLoop measures the full end-to-end cost of injecting
// all policies at startup — the initialize() hot path. Each iteration injects
// all benchPolicies, simulating a Central restart.
func BenchmarkPolicyInjectionLoop(b *testing.B) {
	alertCounts := []int{100, 1000, 5000}

	for _, perPolicy := range alertCounts {
		b.Run(fmt.Sprintf("alertsPerPolicy=%d", perPolicy), func(b *testing.B) {
			env := setupBenchEnv(b, perPolicy)

			allPolicies := make([]*storage.Policy, len(benchPolicies))
			for i, bp := range benchPolicies {
				allPolicies[i] = makeRuntimePolicy(bp.id, bp.name)
			}

			// upsertPolicyOld simulates the original code: AlertAndNotify
			// executes while holding the write lock.
			upsertPolicyOld := func(policy *storage.Policy) error {
				env.manager.policyAlertsLock.Lock()
				defer env.manager.policyAlertsLock.Unlock()

				_ = env.manager.runtimeDetector.PolicySet().UpsertPolicy(policy)

				modifiedDeployments, err := env.manager.alertManager.AlertAndNotify(
					lifecycleMgrCtx, nil,
					alertmanager.WithPolicyID(policy.GetId()))
				if err != nil {
					return err
				}
				if modifiedDeployments.Cardinality() > 0 {
					env.manager.reprocessor.ReprocessRiskForDeployments(modifiedDeployments.AsSlice()...)
				}
				return nil
			}

			b.Run("Before", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					for _, policy := range allPolicies {
						require.NoError(b, upsertPolicyOld(policy))
					}
				}
			})

			b.Run("After", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					for _, policy := range allPolicies {
						require.NoError(b, env.manager.UpsertPolicy(policy))
					}
				}
			})
		})
	}
}
