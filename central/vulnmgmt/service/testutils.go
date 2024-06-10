package service

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type servicePostgresTestSuiteInternals struct {
	ctx         context.Context
	pool        *pgtest.TestPostgres
	deployments deploymentDataStore.DataStore
	images      imageDataStore.DataStore
	service     Service
}

func setupTest(tb testing.TB, suite *servicePostgresTestSuiteInternals) error {
	suite.ctx = sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment, resources.Image),
		),
	)
	suite.pool = pgtest.ForT(tb)
	if suite.pool == nil {
		return errors.New("failed to create database")
	}
	deploymentStore, err := deploymentDataStore.GetTestPostgresDataStore(tb, suite.pool)
	if err != nil {
		return err
	}
	suite.deployments = deploymentStore
	suite.images = imageDataStore.GetTestPostgresDataStore(tb, suite.pool)
	suite.service = New(deploymentStore, suite.images)
	return nil
}

func cleanupTest(tb testing.TB, suite *servicePostgresTestSuiteInternals) {
	suite.pool.Teardown(tb)
	suite.pool.Close()
}

// getAuthInterceptor returns a GRPC Steam Interceptor that overrides the server context to make sure all calls are authenticated.
func getAuthInterceptor(ctx context.Context) func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &grpcMiddleware.WrappedServerStream{
			ServerStream:   ss,
			WrappedContext: ctx,
		})
	}
}

func createGRPCWorkloadsService(suite *servicePostgresTestSuiteInternals) (*grpc.ClientConn, func(), error) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer(grpc.StreamInterceptor(getAuthInterceptor(suite.ctx)))
	v1.RegisterVulnMgmtServiceServer(server, suite.service)

	go func() {
		utils.IgnoreError(func() error { return server.Serve(listener) })
	}()

	conn, err := grpc.DialContext(suite.ctx, "",
		grpc.WithContextDialer(
			func(ctx context.Context, _ string) (net.Conn, error) {
				return listener.DialContext(ctx)
			},
		),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}

	closeFunc := func() {
		utils.IgnoreError(listener.Close)
		server.Stop()
	}
	return conn, closeFunc, nil
}

func receiveWorkloads(
	ctx context.Context,
	client v1.VulnMgmtServiceClient,
	request *v1.VulnMgmtExportWorkloadsRequest,
	swallow bool,
) ([]*v1.VulnMgmtExportWorkloadsResponse, error) {
	out, err := client.VulnMgmtExportWorkloads(ctx, request)
	if err != nil {
		return nil, err
	}
	var results []*v1.VulnMgmtExportWorkloadsResponse
	for {
		chunk, err := out.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if !swallow {
			results = append(results, chunk)
		}
	}
	return results, nil
}
