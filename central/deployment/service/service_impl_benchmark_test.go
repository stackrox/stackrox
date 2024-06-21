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
	"google.golang.org/grpc"
)

func BenchmarkService_Export(b *testing.B) {
	testHelper := &testutils.ExportServicePostgresTestHelper{}
	err := testHelper.SetupTest(b)
	if err != nil {
		b.Error(err)
	}
	defer testHelper.TearDownTest(b)
	svc := New(testHelper.Deployments, nil, nil, nil, nil, nil)
	benchmarkFunc := getExportServiceBenchmark(testHelper, svc)
	testHelper.InjectDataAndRunBenchmark(b, false, benchmarkFunc)
}

func getExportServiceBenchmark(
	helper *testutils.ExportServicePostgresTestHelper,
	svc Service,
) func(b *testing.B) {
	return func(b *testing.B) {
		testCases := testutils.GetBaseTestCases()
		conn, closeFunc, err := helper.CreateGRPCStreamingService(
			b,
			func(registrar grpc.ServiceRegistrar) {
				v1.RegisterDeploymentServiceServer(registrar, svc)
			},
		)
		if err != nil {
			b.Error(err)
		}
		defer closeFunc()

		for _, testCase := range testCases {
			b.Run(testCase.Name, func(ib *testing.B) {
				request := &v1.ExportDeploymentRequest{Timeout: 3600}
				if testCase.TargetNamespace != "" {
					request.Query = fmt.Sprintf("Namespace:%s", testCase.TargetNamespace)
				}
				client := v1.NewDeploymentServiceClient(conn)
				ib.ResetTimer()
				for i := 0; i < ib.N; i++ {
					_, err = receiveWorkloads(helper.Ctx, client, request, true)
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
