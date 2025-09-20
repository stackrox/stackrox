//go:build sql_integration

package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/imagev2/datastore/mapper"
	"github.com/stackrox/rox/central/testutils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

func TestServicePostgres(t *testing.T) {
	suite.Run(t, new(servicePostgresTestSuite))
}

type servicePostgresTestSuite struct {
	suite.Suite

	helper  *testutils.ExportServicePostgresTestHelper
	service Service
}

func (s *servicePostgresTestSuite) SetupTest() {
	s.helper = &testutils.ExportServicePostgresTestHelper{}
	err := s.helper.SetupTest(s.T())
	s.Require().NoError(err)
	s.service = New(s.helper.Deployments, s.helper.Images, s.helper.ImagesV2, s.helper.Pods)
}

func (s *servicePostgresTestSuite) createDeployment(deployment *storage.Deployment, id string) *storage.Deployment {
	deployment.Id = id
	return deployment
}

func (s *servicePostgresTestSuite) createPod(pod *storage.Pod, podID string, deploymentID string) *storage.Pod {
	pod.Id = podID
	pod.DeploymentId = deploymentID
	for _, instance := range pod.GetLiveInstances() {
		instance.ContainingPodId = podID
	}
	for _, terminated := range pod.GetTerminatedInstances() {
		for _, instance := range terminated.GetInstances() {
			instance.ContainingPodId = podID
		}
	}
	return pod
}

func (s *servicePostgresTestSuite) upsertDeployments(deploymentsByID map[string]*storage.Deployment) {
	upsertCtx := sac.WithAllAccess(context.Background())
	for _, deployment := range deploymentsByID {
		err := s.helper.Deployments.UpsertDeployment(upsertCtx, deployment)
		s.Require().NoError(err)
	}
	for _, image := range fixtures.DeploymentImages() {
		var err error
		if features.FlattenImageData.Enabled() {
			err = s.helper.ImagesV2.UpsertImage(upsertCtx, mapper.ConvertToV2(image))
		} else {
			err = s.helper.Images.UpsertImage(upsertCtx, image)
		}
		s.Require().NoError(err)
	}
}

func (s *servicePostgresTestSuite) upsertPods(pods []*storage.Pod) {
	upsertCtx := sac.WithAllAccess(context.Background())
	for _, pod := range pods {
		err := s.helper.Pods.UpsertPod(upsertCtx, pod)
		s.Require().NoError(err)
	}
}

func (s *servicePostgresTestSuite) TestExport() {
	cases := []struct {
		name                    string
		deploymentsByID         map[string]*storage.Deployment
		pods                    []*storage.Pod
		query                   string
		expectedDeploymentIDs   []string
		expectedLivePodsByDepID map[string]int32
	}{
		{
			name: "zero deployments",
		},
		{
			name: "one deployment no query",
			deploymentsByID: map[string]*storage.Deployment{
				fixtureconsts.Deployment1: s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment1),
			},
			pods: []*storage.Pod{
				s.createPod(fixtures.GetPod(), fixtureconsts.PodUID1, fixtureconsts.Deployment1),
			},
			expectedDeploymentIDs: []string{
				fixtureconsts.Deployment1,
			},
			expectedLivePodsByDepID: map[string]int32{
				fixtureconsts.Deployment1: 1,
			},
		},
		{
			name: "multiple deployments no query",
			deploymentsByID: map[string]*storage.Deployment{
				fixtureconsts.Deployment1: s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment1),
				fixtureconsts.Deployment2: s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment2),
				fixtureconsts.Deployment3: s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment3),
			},
			pods: []*storage.Pod{
				s.createPod(fixtures.GetPod(), fixtureconsts.PodUID1, fixtureconsts.Deployment1),
				s.createPod(fixtures.GetPod(), fixtureconsts.PodUID2, fixtureconsts.Deployment2),
				s.createPod(fixtures.GetPod(), fixtureconsts.PodUID3, fixtureconsts.Deployment2),
			},
			expectedDeploymentIDs: []string{
				fixtureconsts.Deployment1,
				fixtureconsts.Deployment2,
				fixtureconsts.Deployment3,
			},
			expectedLivePodsByDepID: map[string]int32{
				fixtureconsts.Deployment1: 1,
				fixtureconsts.Deployment2: 2,
				fixtureconsts.Deployment3: 0,
			},
		},
		{
			name: "multiple deployments with query",
			deploymentsByID: map[string]*storage.Deployment{
				fixtureconsts.Deployment1: s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment1),
				fixtureconsts.Deployment2: s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment2),
				fixtureconsts.Deployment3: s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment3),
			},
			pods: []*storage.Pod{
				s.createPod(fixtures.GetPod(), fixtureconsts.PodUID1, fixtureconsts.Deployment1),
				s.createPod(fixtures.GetPod(), fixtureconsts.PodUID2, fixtureconsts.Deployment2),
				s.createPod(fixtures.GetPod(), fixtureconsts.PodUID3, fixtureconsts.Deployment3),
			},
			query: fmt.Sprintf("Deployment ID:%s", fixtureconsts.Deployment2),
			expectedDeploymentIDs: []string{
				fixtureconsts.Deployment2,
			},
			expectedLivePodsByDepID: map[string]int32{
				fixtureconsts.Deployment2: 1,
			},
		},
		{
			name: "duplicate image deployment",
			deploymentsByID: map[string]*storage.Deployment{
				fixtureconsts.Deployment6: s.createDeployment(fixtures.DuplicateImageDeployment(), fixtureconsts.Deployment6),
			},
			pods: []*storage.Pod{
				s.createPod(fixtures.GetPod(), fixtureconsts.PodUID1, fixtureconsts.Deployment6),
				s.createPod(fixtures.GetPod(), fixtureconsts.PodUID2, fixtureconsts.Deployment6),
				s.createPod(fixtures.GetPod(), fixtureconsts.PodUID3, fixtureconsts.Deployment6),
			},
			query: fmt.Sprintf("Deployment ID:%s", fixtureconsts.Deployment6),
			expectedDeploymentIDs: []string{
				fixtureconsts.Deployment6,
			},
			expectedLivePodsByDepID: map[string]int32{
				fixtureconsts.Deployment6: 3,
			},
		},
	}
	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			s.upsertDeployments(c.deploymentsByID)
			s.upsertPods(c.pods)

			request := &v1.VulnMgmtExportWorkloadsRequest{Timeout: 5, Query: c.query}
			conn, closeFunc, err := pkgGRPC.CreateTestGRPCStreamingService(
				s.helper.Ctx,
				s.T(),
				func(registrar grpc.ServiceRegistrar) {
					v1.RegisterVulnMgmtServiceServer(registrar, s.service)
				},
			)
			s.Require().NoError(err)
			defer closeFunc()
			client := v1.NewVulnMgmtServiceClient(conn)
			results, err := receiveWorkloads(s.helper.Ctx, s.T(), client, request, false)
			s.Require().NoError(err)

			// The images are the same for all deployments to simplify the assertions.
			expectedImages := fixtures.DeploymentImages()
			expectedImageIDs := []string{expectedImages[0].GetId(), expectedImages[1].GetId()}

			// We cannot perform a full assert on the response because it contains variable data
			// and timestamps.
			s.Require().Len(results, len(c.expectedDeploymentIDs))
			for i := range results {
				depID := results[i].Deployment.GetId()
				s.Assert().Contains(c.expectedDeploymentIDs, depID)
				protoassert.Equal(s.T(), c.deploymentsByID[depID], results[i].Deployment)

				s.Assert().Equal(c.expectedLivePodsByDepID[depID], results[i].GetLivePods())

				var imageIDs []string
				for _, image := range results[i].Images {
					imageIDs = append(imageIDs, image.GetId())
				}
				s.Assert().ElementsMatch(expectedImageIDs, imageIDs)
			}
		})
	}
}
