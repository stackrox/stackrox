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
		storage.Deployment_builder{
			Id: uuid.NewV4().String(),
			Containers: []*storage.Container{
				storage.Container_builder{
					Volumes: []*storage.Volume{
						storage.Volume_builder{
							Source: "/tmp/blah",
							Type:   "EmptyDir",
						}.Build(),
					},
				}.Build(),
			},
		}.Build(),
		storage.Deployment_builder{
			Id: uuid.NewV4().String(),
			Containers: []*storage.Container{
				storage.Container_builder{
					Volumes: []*storage.Volume{
						storage.Volume_builder{
							Source: "/tmp/blah",
							Type:   "EmptyDir",
						}.Build(),
					},
				}.Build(),
			},
		}.Build(),
	}

	policies := []*storage.Policy{
		storage.Policy_builder{
			PolicySections: []*storage.PolicySection{
				storage.PolicySection_builder{
					SectionName: "section-1",
					PolicyGroups: []*storage.PolicyGroup{
						storage.PolicyGroup_builder{
							FieldName: fieldnames.VolumeSource,
							Values: []*storage.PolicyValue{
								storage.PolicyValue_builder{
									Value: "/etc/passwd",
								}.Build(),
							},
						}.Build(),
						storage.PolicyGroup_builder{
							FieldName: fieldnames.VolumeType,
							Values: []*storage.PolicyValue{
								storage.PolicyValue_builder{
									Value: "EmptyDir",
								}.Build(),
							},
						}.Build(),
					},
				}.Build(),
			},
			PolicyVersion: "1.1",
		}.Build(),
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().Deployments().AnyTimes().Return(toMap(testDeployments))
	data.EXPECT().Policies().AnyTimes().Return(policiesToMap(s.T(), policies))
	data.EXPECT().UnresolvedAlerts().AnyTimes().Return(nil)

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments, nil)
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
		storage.Deployment_builder{
			Id:   uuid.NewV4().String(),
			Name: "foo",
			Containers: []*storage.Container{
				storage.Container_builder{
					Volumes: []*storage.Volume{
						storage.Volume_builder{
							Source: "/etc/passwd",
							Type:   "HostPath (bare host directory volume)",
						}.Build(),
					},
				}.Build(),
			},
		}.Build(),
	}

	policies := []*storage.Policy{
		storage.Policy_builder{
			Id:   "policy1",
			Name: "policy-1",
			PolicySections: []*storage.PolicySection{
				storage.PolicySection_builder{
					SectionName: "section-1",
					PolicyGroups: []*storage.PolicyGroup{
						storage.PolicyGroup_builder{
							FieldName: fieldnames.VolumeSource,
							Values: []*storage.PolicyValue{
								storage.PolicyValue_builder{
									Value: "/etc/passwd",
								}.Build(),
							},
						}.Build(),
						storage.PolicyGroup_builder{
							FieldName: fieldnames.VolumeType,
							Values: []*storage.PolicyValue{
								storage.PolicyValue_builder{
									Value: "HostPath (bare host directory volume)",
								}.Build(),
							},
						}.Build(),
					},
				}.Build(),
			},
			PolicyVersion: "1.1",
		}.Build(),
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().Deployments().AnyTimes().Return(toMap(testDeployments))
	data.EXPECT().Policies().AnyTimes().Return(policiesToMap(s.T(), policies))
	data.EXPECT().UnresolvedAlerts().AnyTimes().Return([]*storage.ListAlert{
		storage.ListAlert_builder{
			Id:    "alert1",
			State: storage.ViolationState_ACTIVE,
			Policy: storage.ListAlertPolicy_builder{
				Id:   "policy1",
				Name: "policy-1",
			}.Build(),
			Deployment: storage.ListAlertDeployment_builder{
				Id:   testDeployments[0].GetId(),
				Name: testDeployments[0].GetName(),
			}.Build(),
		}.Build(),
	})

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments, nil)
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
	cluster := &storage.Cluster{}
	cluster.SetId(uuid.NewV4().String())
	return cluster
}

func (s *suiteImpl) nodes() []*storage.Node {
	node := &storage.Node{}
	node.SetId(uuid.NewV4().String())
	node2 := &storage.Node{}
	node2.SetId(uuid.NewV4().String())
	return []*storage.Node{
		node,
		node2,
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
