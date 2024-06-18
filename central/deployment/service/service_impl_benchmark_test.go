//go:build sql_integration

package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"

	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const (
	namespace10pct = "Namespace10%"
	namepsace90pct = "Namespace90%"
)

func BenchmarkService_Export(b *testing.B) {
	testSuite := &servicePostgresTestHelper{}
	err := setupTest(b, testSuite)
	if err != nil {
		b.Error(err)
	}
	defer cleanupTest(b, testSuite)

	total := 0

	scaleDeltas := []int{500, 500, 1000, 3000, 5000 /*, 10000*/}
	for _, delta := range scaleDeltas {
		total += delta
		err = injectRandomDeployments(testSuite, delta)
		if err != nil {
			b.Error(err)
		}
		b.Run(fmt.Sprintf("%d", total), getServiceBenchmark(testSuite))
	}
}

type servicePostgresTestHelper struct {
	ctx         context.Context
	pool        *pgtest.TestPostgres
	deployments deploymentDataStore.DataStore
	service     Service
}

func setupTest(tb testing.TB, suite *servicePostgresTestHelper) error {
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
	suite.service = New(deploymentStore, nil, nil, nil, nil, nil)
	return nil
}

func cleanupTest(tb testing.TB, suite *servicePostgresTestHelper) {
	suite.pool.Teardown(tb)
	suite.pool.Close()
}

func injectRandomDeployments(
	suite *servicePostgresTestHelper,
	count int,
) error {
	for i := 0; i < count; i++ {
		d := &storage.Deployment{}
		err := testutils.FullInit(d, testutils.UniqueInitializer(), testutils.JSONFieldsFilter)
		if err != nil {
			return err
		}
		nContainers := i%3 + 1
		containers := make([]*storage.Container, 0, 3)
		for j := 0; j < nContainers; j++ {
			container := &storage.Container{}
			err := testutils.FullInit(container, testutils.UniqueInitializer(), testutils.JSONFieldsFilter)
			if err != nil {
				return err
			}
			containers = append(containers, container)
		}
		if i%10 == 9 {
			d.Namespace = namespace10pct
		} else {
			d.Namespace = namepsace90pct
		}
		d.Containers = containers
		ctx := sac.WithAllAccess(context.Background())
		err = suite.deployments.UpsertDeployment(ctx, d)
		if err != nil {
			return err
		}
	}
	return nil
}

func getServiceBenchmark(suite *servicePostgresTestHelper) func(b *testing.B) {
	return func(b *testing.B) {
		testScenarios := []struct {
			name            string
			query           string
			targetNamespace string
		}{
			{
				name: "No Query",
			},
			{
				name:            "Query 10% of dataset",
				targetNamespace: namespace10pct,
			},
			{
				name:            "Query 90% of dataset",
				targetNamespace: namepsace90pct,
			},
		}

		for _, scenario := range testScenarios {
			b.Run(scenario.name, func(b *testing.B) {
				request := &v1.ExportDeploymentRequest{Timeout: 3600}
				if scenario.targetNamespace != "" {
					request.Query = fmt.Sprintf("Namespace:%s", scenario.targetNamespace)
				}
				conn, closeFunc, err := createGRPCWorkloadsService(suite)
				if err != nil {
					b.Error(err)
				}
				defer closeFunc()

				client := v1.NewDeploymentServiceClient(conn)
				for i := 0; i < b.N; i++ {
					_, err = receiveWorkloads(suite.ctx, client, request, true)
					if err != nil {
						b.Error(err)
					}
				}
			})
		}
	}
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

func createGRPCWorkloadsService(suite *servicePostgresTestHelper) (*grpc.ClientConn, func(), error) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer(grpc.StreamInterceptor(getAuthInterceptor(suite.ctx)))
	v1.RegisterDeploymentServiceServer(server, suite.service)

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
	client v1.DeploymentServiceClient,
	request *v1.ExportDeploymentRequest,
	swallow bool,
) ([]*v1.ExportDeploymentResponse, error) {
	out, err := client.ExportDeployments(ctx, request)
	if err != nil {
		return nil, err
	}
	var results []*v1.ExportDeploymentResponse
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
