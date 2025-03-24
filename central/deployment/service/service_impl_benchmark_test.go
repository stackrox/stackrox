//go:build sql_integration

package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stackrox/rox/central/testutils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"google.golang.org/grpc"
)

func BenchmarkService_Export(b *testing.B) {
	testHelper := &testutils.ExportServicePostgresTestHelper{}
	err := testHelper.SetupTest(b)
	if err != nil {
		b.Error(err)
	}
	svc := New(testHelper.Deployments, nil, nil, nil, nil, nil)
	benchmarkFunc := getExportServiceBenchmark(testHelper, svc)
	testHelper.InjectDataAndRunBenchmark(b, false, benchmarkFunc)
}

func getExportServiceBenchmark(
	helper *testutils.ExportServicePostgresTestHelper,
	svc Service,
) func(b *testing.B) {
	return func(b *testing.B) {
		conn, closeFunc, err := pkgGRPC.CreateTestGRPCStreamingService(
			helper.Ctx,
			b,
			func(registrar grpc.ServiceRegistrar) {
				v1.RegisterDeploymentServiceServer(registrar, svc)
			},
		)
		if err != nil {
			b.Error(err)
		}
		defer closeFunc()

		testCases := testutils.GetExportTestCases()
		for _, testCase := range testCases {
			b.Run(testCase.Name, func(ib *testing.B) {
				request := &v1.ExportDeploymentRequest{Timeout: 3600}
				if testCase.TargetNamespace != "" {
					request.Query = fmt.Sprintf("Namespace:%s", testCase.TargetNamespace)
				}
				client := v1.NewDeploymentServiceClient(conn)
				ib.ResetTimer()
				for i := 0; i < ib.N; i++ {
					err = receiveWorkloads(helper.Ctx, client, request)
					if err != nil {
						ib.Error(err)
					}
				}
			})
		}
	}
}

func receiveWorkloads(
	ctx context.Context,
	client v1.DeploymentServiceClient,
	request *v1.ExportDeploymentRequest,
) error {
	out, err := client.ExportDeployments(ctx, request)
	if err != nil {
		return err
	}
	for {
		_, err := out.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
	}
	return nil
}
