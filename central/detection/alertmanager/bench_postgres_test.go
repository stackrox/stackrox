//go:build sql_integration

package alertmanager

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	runtimeDetectorMocks "github.com/stackrox/rox/central/detection/runtime/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	notifierMocks "github.com/stackrox/rox/pkg/notifier/mocks"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var defaultPolicyIDs = []struct {
	id       string
	name     string
	severity storage.Severity
}{
	{"2e90874a-3521-44de-85c6-5720f519a701", "Latest tag", storage.Severity_LOW_SEVERITY},
	{"fe9de18b-86db-44d5-a7c4-74173ccffe2e", "Privileged Container", storage.Severity_MEDIUM_SEVERITY},
	{"886c3c94-3a6a-4f2b-82fc-d6bf5a310840", "No CPU request or memory limit specified", storage.Severity_MEDIUM_SEVERITY},
	{"f09f8da1-6111-4ca0-8f49-294a76c65115", "Fixable CVSS >= 7", storage.Severity_HIGH_SEVERITY},
	{"cf80fb33-c7d0-4490-b6f4-e56e1f27b4e4", "Log4Shell: log4j Remote Code Execution vulnerability", storage.Severity_CRITICAL_SEVERITY},
}

var deploymentIDs = []string{
	fixtureconsts.Deployment1,
	fixtureconsts.Deployment2,
	fixtureconsts.Deployment3,
}

type benchFixture struct {
	ctx          context.Context
	alertManager AlertManager
	datastore    alertDataStore.DataStore
	mockCtrl     *gomock.Controller
}

func setupBench(b *testing.B, totalAlerts int) *benchFixture {
	b.Helper()

	testDB := pgtest.ForT(b)
	ctx := sac.WithAllAccess(context.Background())
	ds := alertDataStore.GetTestPostgresDataStore(b, testDB.DB)

	mockCtrl := gomock.NewController(b)
	notifier := notifierMocks.NewMockProcessor(mockCtrl)
	runtimeDetector := runtimeDetectorMocks.NewMockDetector(mockCtrl)

	notifier.EXPECT().ProcessAlert(gomock.Any(), gomock.Any()).AnyTimes()
	runtimeDetector.EXPECT().DeploymentInactive(gomock.Any()).Return(false).AnyTimes()

	am := New(notifier, ds, runtimeDetector)

	for i := 0; i < totalAlerts; i++ {
		a := fixtures.GetAlertWithID(uuid.NewV4().String())

		// Distribute across 3 deployments with 60/25/15 ratio
		randVal := rand.Intn(100)
		switch {
		case randVal < 60:
			a.GetDeployment().Id = fixtureconsts.Deployment1
		case randVal < 85:
			a.GetDeployment().Id = fixtureconsts.Deployment2
		default:
			a.GetDeployment().Id = fixtureconsts.Deployment3
		}

		policyInfo := defaultPolicyIDs[rand.Intn(len(defaultPolicyIDs))]
		a.Policy = fixtures.GetPolicy()
		a.Policy.Id = policyInfo.id
		a.Policy.Name = policyInfo.name
		a.Policy.Severity = policyInfo.severity

		lifecycleStages := []storage.LifecycleStage{
			storage.LifecycleStage_DEPLOY,
			storage.LifecycleStage_RUNTIME,
		}
		a.LifecycleStage = lifecycleStages[rand.Intn(len(lifecycleStages))]

		states := []storage.ViolationState{
			storage.ViolationState_ACTIVE,
			storage.ViolationState_ATTEMPTED,
		}
		a.State = states[rand.Intn(len(states))]

		require.NoError(b, ds.UpsertAlert(ctx, a))
	}

	return &benchFixture{
		ctx:          ctx,
		alertManager: am,
		datastore:    ds,
		mockCtrl:     mockCtrl,
	}
}

// makeIncomingAlerts creates incoming alerts for a specific deployment with
// the given lifecycle stage. These simulate what the detection pipeline
// produces before calling AlertAndNotify.
func makeIncomingAlerts(count int, deploymentID string, stage storage.LifecycleStage) []*storage.Alert {
	alerts := make([]*storage.Alert, count)
	for i := range alerts {
		a := fixtures.GetAlertWithID(uuid.NewV4().String())
		a.GetDeployment().Id = deploymentID

		policyInfo := defaultPolicyIDs[rand.Intn(len(defaultPolicyIDs))]
		a.Policy = fixtures.GetPolicy()
		a.Policy.Id = policyInfo.id
		a.Policy.Name = policyInfo.name
		a.Policy.Severity = policyInfo.severity
		a.LifecycleStage = stage
		a.State = storage.ViolationState_ACTIVE

		alerts[i] = a
	}
	return alerts
}

func BenchmarkMergeManyAlerts(b *testing.B) {
	alertCounts := []int{100, 500, 1000}

	for _, totalAlerts := range alertCounts {
		b.Run(fmt.Sprintf("previousAlerts=%d", totalAlerts), func(b *testing.B) {
			fix := setupBench(b, totalAlerts)
			impl := fix.alertManager.(*alertManagerImpl)

			// Simulate HandleDeploymentAlerts: incoming alerts for one deployment
			incoming := makeIncomingAlerts(5, fixtureconsts.Deployment1, storage.LifecycleStage_DEPLOY)
			filters := []AlertFilterOption{
				WithLifecycleStage(storage.LifecycleStage_DEPLOY),
				WithDeploymentID(fixtureconsts.Deployment1, false),
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, _, err := impl.mergeManyAlerts(fix.ctx, incoming, filters...)
				require.NoError(b, err)
			}
		})
	}
}

func BenchmarkAlertAndNotify(b *testing.B) {
	alertCounts := []int{100, 500, 1000}

	for _, totalAlerts := range alertCounts {
		b.Run(fmt.Sprintf("previousAlerts=%d", totalAlerts), func(b *testing.B) {
			fix := setupBench(b, totalAlerts)

			incoming := makeIncomingAlerts(5, fixtureconsts.Deployment1, storage.LifecycleStage_DEPLOY)
			filters := []AlertFilterOption{
				WithLifecycleStage(storage.LifecycleStage_DEPLOY),
				WithDeploymentID(fixtureconsts.Deployment1, false),
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := fix.alertManager.AlertAndNotify(fix.ctx, incoming, filters...)
				require.NoError(b, err)
			}
		})
	}
}
