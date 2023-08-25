package check455

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestCheck(t *testing.T) {
	suite.Run(t, new(suiteImpl))
}

type suiteImpl struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *suiteImpl) SetupSuite() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *suiteImpl) TearDownSuite() {
	s.mockCtrl.Finish()
}

func (s *suiteImpl) TestPass() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()

	testNodes := s.nodes()

	testDeployments := []*storage.Deployment{
		{
			Id: uuid.NewV4().String(),
			Containers: []*storage.Container{
				{
					Volumes: []*storage.Volume{
						{
							Source: "/tmp/blah",
							Type:   "EmptyDir",
						},
					},
				},
			},
		},
		{
			Id: uuid.NewV4().String(),
			Containers: []*storage.Container{
				{
					Volumes: []*storage.Volume{
						{
							Source: "/tmp/blah",
							Type:   "EmptyDir",
						},
					},
				},
			},
		},
	}

	policies := []*storage.Policy{
		{
			PolicySections: []*storage.PolicySection{
				{
					SectionName: "section-1",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.VolumeSource,
							Values: []*storage.PolicyValue{
								{
									Value: "/etc/passwd",
								},
							},
						},
						{
							FieldName: fieldnames.VolumeType,
							Values: []*storage.PolicyValue{
								{
									Value: "EmptyDir",
								},
							},
						},
					},
				},
			},
			PolicyVersion: "1.1",
		},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().Deployments().AnyTimes().Return(toMap(testDeployments))
	data.EXPECT().Policies().AnyTimes().Return(policiesToMap(s.T(), policies))
	data.EXPECT().UnresolvedAlerts().AnyTimes().Return(nil)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments, nil, nil)
	err = run.Run(context.Background(), "standard", domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	s.NotNil(checkResults)

	for _, deployment := range domain.Deployments() {
		deploymentResults := checkResults.ForChild(deployment)
		s.NoError(deploymentResults.Error())
		if s.Len(deploymentResults.Evidence(), 1) {
			s.Equal(framework.PassStatus, deploymentResults.Evidence()[0].Status)
		}
	}
}

func (s *suiteImpl) TestFail() {
	check := s.verifyCheckRegistered()

	testCluster := s.cluster()

	testNodes := s.nodes()

	testDeployments := []*storage.Deployment{
		{
			Id:   uuid.NewV4().String(),
			Name: "foo",
			Containers: []*storage.Container{
				{
					Volumes: []*storage.Volume{
						{
							Source: "/etc/passwd",
							Type:   "HostPath (bare host directory volume)",
						},
					},
				},
			},
		},
	}

	policies := []*storage.Policy{
		{
			Id:   "policy1",
			Name: "policy-1",
			PolicySections: []*storage.PolicySection{
				{
					SectionName: "section-1",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.VolumeSource,
							Values: []*storage.PolicyValue{
								{
									Value: "/etc/passwd",
								},
							},
						},
						{
							FieldName: fieldnames.VolumeType,
							Values: []*storage.PolicyValue{
								{
									Value: "HostPath (bare host directory volume)",
								},
							},
						},
					},
				},
			},
			PolicyVersion: "1.1",
		},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().Deployments().AnyTimes().Return(toMap(testDeployments))
	data.EXPECT().Policies().AnyTimes().Return(policiesToMap(s.T(), policies))
	data.EXPECT().UnresolvedAlerts().AnyTimes().Return([]*storage.ListAlert{
		{
			Id:    "alert1",
			State: storage.ViolationState_ACTIVE,
			Policy: &storage.ListAlertPolicy{
				Id:   "policy1",
				Name: "policy-1",
			},
			Entity: &storage.ListAlert_Deployment{
				Deployment: &storage.ListAlertDeployment{
					Id:   testDeployments[0].Id,
					Name: testDeployments[0].Name,
				},
			},
		},
	})

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments, nil, nil)
	err = run.Run(context.Background(), "standard", domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	s.NotNil(checkResults)
	for _, deployment := range domain.Deployments() {
		deploymentResults := checkResults.ForChild(deployment)
		s.NotNil(deploymentResults)
		s.NoError(deploymentResults.Error())
		if s.Len(deploymentResults.Evidence(), 1) {
			s.Equal(framework.FailStatus, deploymentResults.Evidence()[0].Status)
		}
	}
}

// Helper functions for test data.
//////////////////////////////////

func (s *suiteImpl) verifyCheckRegistered() framework.Check {
	registry := framework.RegistrySingleton()
	check := registry.Lookup(standardID)
	s.NotNil(check)
	return check
}

func (s *suiteImpl) cluster() *storage.Cluster {
	return &storage.Cluster{
		Id: uuid.NewV4().String(),
	}
}

func (s *suiteImpl) nodes() []*storage.Node {
	return []*storage.Node{
		{
			Id: uuid.NewV4().String(),
		},
		{
			Id: uuid.NewV4().String(),
		},
	}
}

func toMap(in []*storage.Deployment) map[string]*storage.Deployment {
	merp := make(map[string]*storage.Deployment, len(in))
	for _, np := range in {
		merp[np.GetId()] = np
	}
	return merp
}

func policiesToMap(t *testing.T, in []*storage.Policy) map[string]*storage.Policy {
	merp := make(map[string]*storage.Policy, len(in))
	for _, np := range in {
		require.NoError(t, policyversion.EnsureConvertedToLatest(np))
		merp[np.GetId()] = np
	}
	return merp
}
