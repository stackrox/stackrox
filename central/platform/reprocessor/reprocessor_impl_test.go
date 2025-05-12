package reprocessor

import (
	"context"
	"testing"

	alertDSMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	configDatastoreMocks "github.com/stackrox/rox/central/config/datastore/mocks"
	deploymentDSMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
)

func TestPlatformReprocessorImpl(t *testing.T) {
	suite.Run(t, new(platformReprocessorImplTestSuite))
}

type platformReprocessorImplTestSuite struct {
	suite.Suite

	reprocessor         *platformReprocessorImpl
	alertDatastore      *alertDSMocks.MockDataStore
	configDatastore     *configDatastoreMocks.MockDataStore
	deploymentDatastore *deploymentDSMocks.MockDataStore

	mockCtrl *gomock.Controller
}

func (s *platformReprocessorImplTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.alertDatastore = alertDSMocks.NewMockDataStore(s.mockCtrl)
	s.configDatastore = configDatastoreMocks.NewMockDataStore(s.mockCtrl)
	s.deploymentDatastore = deploymentDSMocks.NewMockDataStore(s.mockCtrl)

	s.reprocessor = &platformReprocessorImpl{
		alertDatastore:      s.alertDatastore,
		deploymentDatastore: s.deploymentDatastore,
		platformMatcher:     platformmatcher.Singleton(),
		stopSignal:          concurrency.NewSignal(),
		semaphore:           semaphore.NewWeighted(1),
	}
}

func (s *platformReprocessorImplTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
	s.reprocessor.stopSignal.Signal()
}

func (s *platformReprocessorImplTestSuite) TestRunReprocessing() {
	ctx := sac.WithAllAccess(context.Background())

	// Case: Needs reprocessing is false for both alerts and deployments
	s.alertDatastore.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.deploymentDatastore.EXPECT().SearchRawDeployments(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	s.reprocessor.RunReprocessor()

	deployments := testDeployments()

	// Case: Alerts and deployments are updated

	// Mock calls made by alert reprocessing loop
	s.alertDatastore.EXPECT().WalkByQuery(ctx, gomock.Any(), gomock.Any()).DoAndReturn(walk()).AnyTimes()
	s.alertDatastore.EXPECT().WalkByQuery(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.alertDatastore.EXPECT().UpsertAlerts(ctx, expectedAlerts()).Return(nil).AnyTimes()

	// Mock calls made by deployment reprocessing loop
	s.deploymentDatastore.EXPECT().SearchRawDeployments(ctx, gomock.Any()).Return(deployments, nil).AnyTimes()
	s.deploymentDatastore.EXPECT().SearchRawDeployments(ctx, gomock.Any()).Return(nil, nil).AnyTimes()

	expectedDeps := expectedDeployments()
	s.deploymentDatastore.EXPECT().UpsertDeployment(ctx, expectedDeps[0]).Return(nil).AnyTimes()
	s.deploymentDatastore.EXPECT().UpsertDeployment(ctx, expectedDeps[1]).Return(nil).AnyTimes()
	s.deploymentDatastore.EXPECT().UpsertDeployment(ctx, expectedDeps[2]).Return(nil).AnyTimes()
	s.deploymentDatastore.EXPECT().UpsertDeployment(ctx, expectedDeps[3]).Return(nil).AnyTimes()

	s.reprocessor.RunReprocessor()
}

func (s *platformReprocessorImplTestSuite) TestStartAndStop() {
	s.alertDatastore.EXPECT().Count(gomock.Any(), gomock.Any(), true).Return(6, nil).AnyTimes()
	s.deploymentDatastore.EXPECT().Count(gomock.Any(), gomock.Any()).Return(4, nil).AnyTimes()

	// Mock calls made by alert reprocessing loop
	proceedAlertLoop := concurrency.NewSignal()
	inAlertLoop := concurrency.NewSignal()
	s.alertDatastore.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(walk()).Times(1)
	s.alertDatastore.EXPECT().UpsertAlerts(gomock.Any(), expectedAlerts()).Do(func(_, _ any) {
		inAlertLoop.Signal()
		proceedAlertLoop.Wait()
	}).Return(nil).Times(1)

	// No calls should be made by deployment reprocessing loop after Stop
	s.deploymentDatastore.EXPECT().SearchRawDeployments(gomock.Any(), gomock.Any()).Times(0)
	s.deploymentDatastore.EXPECT().UpsertDeployment(gomock.Any(), gomock.Any()).Times(0)

	reprocessor := New(s.alertDatastore, s.configDatastore, s.deploymentDatastore, platformmatcher.Singleton())
	reprocessor.Start()
	// Wait until execution has entered alert reprocessing loop. The loop will pause waiting for proceedAlertLoop signal
	inAlertLoop.Wait()
	// Stop reprocessor
	reprocessor.Stop()
	// Let the loop proceed
	proceedAlertLoop.Signal()

	// Alert reprocessing loop completes successfully. Mock calls made by alert reprocessing loop
	s.alertDatastore.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(walk()).Times(1)
	s.alertDatastore.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.alertDatastore.EXPECT().UpsertAlerts(gomock.Any(), expectedAlerts()).Return(nil).Times(1)

	proceedDeploymentLoop := concurrency.NewSignal()
	inDeploymentLoop := concurrency.NewSignal()
	// Stop is called when we are in the middle of deployment reprocessing loop. Mock calls made by deployment reprocessing loop
	s.deploymentDatastore.EXPECT().SearchRawDeployments(gomock.Any(), gomock.Any()).Return(testDeployments(), nil).Times(1)
	s.deploymentDatastore.EXPECT().UpsertDeployment(gomock.Any(), gomock.Any()).Return(nil).Times(3)
	s.deploymentDatastore.EXPECT().UpsertDeployment(gomock.Any(), gomock.Any()).Do(func(_, _ any) {
		inDeploymentLoop.Signal()
		proceedDeploymentLoop.Wait()
	}).Return(nil).Times(1)

	reprocessor = New(s.alertDatastore, s.configDatastore, s.deploymentDatastore, platformmatcher.Singleton())
	reprocessor.Start()
	// Wait until execution has entered deployment reprocessing loop. The loop will pause waiting for proceedAlertLoop signal
	inDeploymentLoop.Wait()
	// Stop reprocessor
	reprocessor.Stop()
	// Let the loop proceed
	proceedDeploymentLoop.Signal()
}

func testAlerts() []*storage.Alert {
	return []*storage.Alert{
		{
			Id: "1",
			Entity: &storage.Alert_Resource_{
				Resource: &storage.Alert_Resource{},
			},
		},
		{
			Id: "2",
			Entity: &storage.Alert_Image{
				Image: &storage.ContainerImage{},
			},
		},
		{
			Id: "3",
			Entity: &storage.Alert_Deployment_{
				Deployment: &storage.Alert_Deployment{
					Name:      "dep1",
					Namespace: "my-namespace",
				},
			},
		},
		{
			Id: "4",
			Entity: &storage.Alert_Deployment_{
				Deployment: &storage.Alert_Deployment{
					Name:      "dep1",
					Namespace: "hive-suffix",
				},
			},
		},
		{
			Id: "5",
			Entity: &storage.Alert_Deployment_{
				Deployment: &storage.Alert_Deployment{
					Name:      "dep1",
					Namespace: "openshift-123",
				},
			},
		},
		{
			Id: "6",
			Entity: &storage.Alert_Deployment_{
				Deployment: &storage.Alert_Deployment{
					Name:      "dep1",
					Namespace: "stackrox",
				},
			},
		},
	}
}

func expectedAlerts() []*storage.Alert {
	return []*storage.Alert{
		{
			Id: "1",
			Entity: &storage.Alert_Resource_{
				Resource: &storage.Alert_Resource{},
			},
			EntityType:        storage.Alert_RESOURCE,
			PlatformComponent: false,
		},
		{
			Id: "2",
			Entity: &storage.Alert_Image{
				Image: &storage.ContainerImage{},
			},
			EntityType:        storage.Alert_CONTAINER_IMAGE,
			PlatformComponent: false,
		},
		{
			Id: "3",
			Entity: &storage.Alert_Deployment_{
				Deployment: &storage.Alert_Deployment{
					Name:      "dep1",
					Namespace: "my-namespace",
				},
			},
			EntityType:        storage.Alert_DEPLOYMENT,
			PlatformComponent: false,
		},
		{
			Id: "4",
			Entity: &storage.Alert_Deployment_{
				Deployment: &storage.Alert_Deployment{
					Name:      "dep1",
					Namespace: "hive-suffix",
				},
			},
			EntityType:        storage.Alert_DEPLOYMENT,
			PlatformComponent: false,
		},
		{
			Id: "5",
			Entity: &storage.Alert_Deployment_{
				Deployment: &storage.Alert_Deployment{
					Name:      "dep1",
					Namespace: "openshift-123",
				},
			},
			EntityType:        storage.Alert_DEPLOYMENT,
			PlatformComponent: true,
		},
		{
			Id: "6",
			Entity: &storage.Alert_Deployment_{
				Deployment: &storage.Alert_Deployment{
					Name:      "dep1",
					Namespace: "stackrox",
				},
			},
			EntityType:        storage.Alert_DEPLOYMENT,
			PlatformComponent: true,
		},
	}
}

func testDeployments() []*storage.Deployment {
	return []*storage.Deployment{
		{
			Id:        "1",
			Namespace: "my-namespace",
		},
		{
			Id:        "2",
			Namespace: "prefix-aap",
		},
		{
			Id:        "3",
			Namespace: "kube-123",
		},
		{
			Id:        "4",
			Namespace: "open-cluster-management",
		},
	}
}

func expectedDeployments() []*storage.Deployment {
	return []*storage.Deployment{
		{
			Id:                "1",
			Namespace:         "my-namespace",
			PlatformComponent: false,
		},
		{
			Id:                "2",
			Namespace:         "prefix-aap",
			PlatformComponent: false,
		},
		{
			Id:                "3",
			Namespace:         "kube-123",
			PlatformComponent: true,
		},
		{
			Id:                "4",
			Namespace:         "open-cluster-management",
			PlatformComponent: true,
		},
	}
}

func walk() func(_ context.Context, _ *v1.Query, fn func(*storage.Alert) error) error {
	return func(_ context.Context, _ *v1.Query, fn func(*storage.Alert) error) error {
		for _, alert := range testAlerts() {
			err := fn(alert)
			if err != nil {
				return err
			}
		}
		return nil
	}
}
