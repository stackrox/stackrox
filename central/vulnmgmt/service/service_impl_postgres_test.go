//go:build sql_integration

package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/testutils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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
	s.service = New(s.helper.Deployments, s.helper.Images)
}

func (s *servicePostgresTestSuite) createDeployment(deployment *storage.Deployment, id string) *storage.Deployment {
	deployment.Id = id
	return deployment
}

func (s *servicePostgresTestSuite) upsertDeployments(deployments []*storage.Deployment) {
	upsertCtx := sac.WithAllAccess(context.Background())
	for _, deployment := range deployments {
		err := s.helper.Deployments.UpsertDeployment(upsertCtx, deployment)
		s.Require().NoError(err)
	}
	for _, image := range fixtures.DeploymentImages() {
		err := s.helper.Images.UpsertImage(upsertCtx, image)
		s.Require().NoError(err)
	}
}

func (s *servicePostgresTestSuite) TestExport() {
	cases := []struct {
		name        string
		deployments []*storage.Deployment
		query       string
		expected    []*storage.Deployment
	}{
		{
			name: "zero deployments",
		},
		{
			name: "one deployment no query",
			deployments: []*storage.Deployment{
				s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment1),
			},
			expected: []*storage.Deployment{
				s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment1),
			},
		},
		{
			name: "multiple deployments no query",
			deployments: []*storage.Deployment{
				s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment1),
				s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment2),
				s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment3),
			},
			expected: []*storage.Deployment{
				s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment1),
				s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment2),
				s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment3),
			},
		},
		{
			name: "multiple deployments with query",
			deployments: []*storage.Deployment{
				s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment1),
				s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment2),
				s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment3),
			},
			query: fmt.Sprintf("Deployment ID:%s", fixtureconsts.Deployment2),
			expected: []*storage.Deployment{
				s.createDeployment(fixtures.GetDeployment(), fixtureconsts.Deployment2),
			},
		},
		{
			name: "duplicate image deployment",
			deployments: []*storage.Deployment{
				s.createDeployment(fixtures.DuplicateImageDeployment(), fixtureconsts.Deployment6),
			},
			query: fmt.Sprintf("Deployment ID:%s", fixtureconsts.Deployment6),
			expected: []*storage.Deployment{
				s.createDeployment(fixtures.DuplicateImageDeployment(), fixtureconsts.Deployment6),
			},
		},
	}
	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			s.upsertDeployments(c.deployments)

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
			s.Require().Len(results, len(c.expected))
			for i := range results {
				protoassert.SliceContains(s.T(), c.deployments, results[i].Deployment)

				var imageIDs []string
				for _, image := range results[i].Images {
					imageIDs = append(imageIDs, image.GetId())
				}
				s.Assert().ElementsMatch(expectedImageIDs, imageIDs)
			}
		})
	}
}
