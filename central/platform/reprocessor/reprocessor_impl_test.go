package reprocessor

import (
	"context"
	"testing"

	alertDSMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	deploymentDSMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestPlatformReprocessorImpl(t *testing.T) {
	suite.Run(t, new(platformReprocessorImplTestSuite))
}

type platformReprocessorImplTestSuite struct {
	suite.Suite

	reprocessor         *platformReprocessorImpl
	alertDatastore      *alertDSMocks.MockDataStore
	deploymentDatastore *deploymentDSMocks.MockDataStore

	mockCtrl *gomock.Controller
}

func (s *platformReprocessorImplTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.alertDatastore = alertDSMocks.NewMockDataStore(s.mockCtrl)
	s.deploymentDatastore = deploymentDSMocks.NewMockDataStore(s.mockCtrl)

	s.reprocessor = &platformReprocessorImpl{
		alertDatastore:      s.alertDatastore,
		deploymentDatastore: s.deploymentDatastore,
		platformMatcher:     platformmatcher.Singleton(),
	}
}

func (s *platformReprocessorImplTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *platformReprocessorImplTestSuite) TestRunReprocessing() {
	ctx := sac.WithAllAccess(context.Background())

	// Needs reprocessing is false
	s.alertDatastore.EXPECT().Count(gomock.Any(), gomock.Any()).Return(0, nil).Times(1)
	s.deploymentDatastore.EXPECT().Count(gomock.Any(), gomock.Any()).Return(0, nil).Times(1)

	s.alertDatastore.EXPECT().SearchRawAlerts(gomock.Any(), gomock.Any()).Times(0)
	s.deploymentDatastore.EXPECT().SearchRawDeployments(gomock.Any(), gomock.Any()).Times(0)
	s.reprocessor.runReprocessing()

	// Alerts and deployments are updated
	s.alertDatastore.EXPECT().Count(gomock.Any(), gomock.Any()).Return(6, nil).Times(1)
	s.deploymentDatastore.EXPECT().Count(gomock.Any(), gomock.Any()).Return(4, nil).Times(1)

	s.alertDatastore.EXPECT().SearchRawAlerts(ctx, gomock.Any()).Return(testAlerts(), nil).Times(1)
	s.alertDatastore.EXPECT().SearchRawAlerts(ctx, gomock.Any()).Return(nil, nil).Times(1)
	s.alertDatastore.EXPECT().UpsertAlerts(ctx, expectedAlerts()).Return(nil).Times(1)

	s.deploymentDatastore.EXPECT().SearchRawDeployments(ctx, gomock.Any()).Return(testDeployments(), nil).Times(1)
	s.deploymentDatastore.EXPECT().SearchRawDeployments(ctx, gomock.Any()).Return(nil, nil).Times(1)

	expectedDeps := expectedDeployments()
	s.deploymentDatastore.EXPECT().UpsertDeployment(ctx, expectedDeps[0]).Return(nil).Times(1)
	s.deploymentDatastore.EXPECT().UpsertDeployment(ctx, expectedDeps[1]).Return(nil).Times(1)
	s.deploymentDatastore.EXPECT().UpsertDeployment(ctx, expectedDeps[2]).Return(nil).Times(1)
	s.deploymentDatastore.EXPECT().UpsertDeployment(ctx, expectedDeps[3]).Return(nil).Times(1)

	s.reprocessor.runReprocessing()
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
					Namespace: "openshift-operators",
				},
			},
		},
		{
			Id: "5",
			Entity: &storage.Alert_Deployment_{
				Deployment: &storage.Alert_Deployment{
					Name:      "dep1",
					Namespace: "openshift123",
				},
			},
		},
		{
			Id: "6",
			Entity: &storage.Alert_Deployment_{
				Deployment: &storage.Alert_Deployment{
					Name:      "dep1",
					Namespace: "redhat123",
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
					Namespace: "openshift-operators",
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
					Namespace: "openshift123",
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
					Namespace: "redhat123",
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
			Namespace: "openshift-operators",
		},
		{
			Id:        "3",
			Namespace: "openshift123",
		},
		{
			Id:        "4",
			Namespace: "redhat123",
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
			Namespace:         "openshift-operators",
			PlatformComponent: false,
		},
		{
			Id:                "3",
			Namespace:         "openshift123",
			PlatformComponent: true,
		},
		{
			Id:                "4",
			Namespace:         "redhat123",
			PlatformComponent: true,
		},
	}
}
