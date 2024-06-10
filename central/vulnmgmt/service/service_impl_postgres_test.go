//go:build sql_integration

package service

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"

	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/pkg/errors"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestServicePostgres(t *testing.T) {
	suite.Run(t, new(servicePostgresTestSuite))
}

type servicePostgresTestSuite struct {
	suite.Suite

	ctx         context.Context
	pool        *pgtest.TestPostgres
	deployments deploymentDS.DataStore
	images      imageDS.DataStore
	service     Service
}

func (s *servicePostgresTestSuite) SetupTest() {
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment, resources.Image),
		),
	)
	s.pool = pgtest.ForT(s.T())
	s.Require().NotNil(s.pool)

	deployments, err := deploymentDS.GetTestPostgresDataStore(s.T(), s.pool)
	s.Require().NoError(err)
	s.deployments = deployments
	s.images = imageDS.GetTestPostgresDataStore(s.T(), s.pool)
	s.service = New(s.deployments, s.images)
}

func (s *servicePostgresTestSuite) TearDownTest() {
	s.pool.Teardown(s.T())
	s.pool.Close()
}

// authInterceptor overrides the server context to make sure all calls are authenticated.
func (s *servicePostgresTestSuite) authInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return handler(srv, &grpcMiddleware.WrappedServerStream{
		ServerStream:   ss,
		WrappedContext: s.ctx,
	})
}

func (s *servicePostgresTestSuite) createGRPCWorkloadsService(ctx context.Context) (*grpc.ClientConn, func()) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer(grpc.StreamInterceptor(s.authInterceptor))
	v1.RegisterVulnMgmtServiceServer(server, s.service)

	go func() {
		utils.IgnoreError(func() error { return server.Serve(listener) })
	}()

	conn, err := grpc.DialContext(ctx, "",
		grpc.WithContextDialer(
			func(ctx context.Context, _ string) (net.Conn, error) {
				return listener.DialContext(ctx)
			},
		),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)

	closeFunc := func() {
		utils.IgnoreError(listener.Close)
		server.Stop()
	}
	return conn, closeFunc
}

func (s *servicePostgresTestSuite) createDeployment(deployment *storage.Deployment, id string) *storage.Deployment {
	deployment.Id = id
	return deployment
}

func (s *servicePostgresTestSuite) upsertDeployments(deployments []*storage.Deployment) {
	upsertCtx := sac.WithAllAccess(context.Background())
	for _, deployment := range deployments {
		err := s.deployments.UpsertDeployment(upsertCtx, deployment)
		s.Require().NoError(err)
	}
	for _, image := range fixtures.DeploymentImages() {
		err := s.images.UpsertImage(upsertCtx, image)
		s.Require().NoError(err)
	}
}

func (s *servicePostgresTestSuite) receiveWorkloads(client v1.VulnMgmtServiceClient,
	request *v1.VulnMgmtExportWorkloadsRequest,
) []*v1.VulnMgmtExportWorkloadsResponse {
	out, err := client.VulnMgmtExportWorkloads(s.ctx, request)
	s.Require().NoError(err)
	var results []*v1.VulnMgmtExportWorkloadsResponse
	for {
		chunk, err := out.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		results = append(results, chunk)
	}
	return results
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
			conn, closeFunc := s.createGRPCWorkloadsService(s.ctx)
			defer closeFunc()
			client := v1.NewVulnMgmtServiceClient(conn)
			results := s.receiveWorkloads(client, request)

			// The images are the same for all deployments to simplify the assertions.
			expectedImages := fixtures.DeploymentImages()
			expectedImageIDs := []string{expectedImages[0].GetId(), expectedImages[1].GetId()}

			// We cannot perform a full assert on the response because it contains variable data
			// and timestamps.
			s.Require().Len(results, len(c.expected))
			for i := range results {
				s.Assert().Contains(c.deployments, results[i].Deployment)

				var imageIDs []string
				for _, image := range results[i].Images {
					imageIDs = append(imageIDs, image.GetId())
				}
				s.Assert().ElementsMatch(expectedImageIDs, imageIDs)
			}
		})
	}
}
