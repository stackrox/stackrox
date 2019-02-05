package check455

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestCheck(t *testing.T) {
	t.Parallel()
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
			Fields: &storage.PolicyFields{
				VolumePolicy: &storage.VolumePolicy{
					Source: "/etc/passwd",
				},
			},
		},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().Deployments().AnyTimes().Return(toMap(testDeployments))
	data.EXPECT().Policies().AnyTimes().Return(policiesToMap(policies))

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments)
	err = run.Run(context.Background(), domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	s.NotNil(checkResults)
	s.Equal(framework.PassStatus, checkResults.Evidence()[0].Status)
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
							Type: "HostPath (bare host directory volume)",
						},
					},
				},
			},
		},
		{
			Id:   uuid.NewV4().String(),
			Name: "boo",
			Containers: []*storage.Container{
				{
					Volumes: []*storage.Volume{
						{
							Type: "HostPath (bare host directory volume)",
						},
					},
				},
			},
		},
	}

	policies := []*storage.Policy{
		{
			Fields: &storage.PolicyFields{
				VolumePolicy: &storage.VolumePolicy{
					Destination: "/etc/passwd",
				},
			},
		},
	}

	data := mocks.NewMockComplianceDataRepository(s.mockCtrl)
	data.EXPECT().Deployments().AnyTimes().Return(toMap(testDeployments))
	data.EXPECT().Policies().AnyTimes().Return(policiesToMap(policies))

	run, err := framework.NewComplianceRun(check)
	s.NoError(err)

	domain := framework.NewComplianceDomain(testCluster, testNodes, testDeployments)
	err = run.Run(context.Background(), domain, data)
	s.NoError(err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	s.NotNil(checkResults)
	s.Equal(framework.FailStatus, checkResults.Evidence()[0].Status)
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

func policiesToMap(in []*storage.Policy) map[string]*storage.Policy {
	merp := make(map[string]*storage.Policy, len(in))
	for _, np := range in {
		merp[np.GetId()] = np
	}
	return merp
}
