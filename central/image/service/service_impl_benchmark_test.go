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
	svc := New(testHelper.Images, nil, nil, nil, nil, nil, nil, nil)
	benchmarkFunc := getExportServiceBenchmark(testHelper, svc)
	testHelper.InjectDataAndRunBenchmark(b, true, benchmarkFunc)
}

func getExportServiceBenchmark(
	helper *testutils.ExportServicePostgresTestHelper,
	svc Service,
) func(b *testing.B) {
	return func(b *testing.B) {
		conn, closeFunc, err := helper.CreateGRPCStreamingService(
			b,
			func(registrar grpc.ServiceRegistrar) {
				v1.RegisterImageServiceServer(registrar, svc)
			},
		)
		if err != nil {
			b.Error(err)
		}
		defer closeFunc()

		testCases := testutils.GetBaseTestCases()
		for _, testCase := range testCases {
			b.Run(testCase.Name, func(ib *testing.B) {
				request := &v1.ExportImageRequest{Timeout: 3600}
				if testCase.TargetNamespace != "" {
					request.Query = fmt.Sprintf("Namespace:%s", testCase.TargetNamespace)
				}

				client := v1.NewImageServiceClient(conn)
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
	client v1.ImageServiceClient,
	request *v1.ExportImageRequest,
	swallow bool,
) ([]*v1.ExportImageResponse, error) {
	out, err := client.ExportImages(ctx, request)
	if err != nil {
		return nil, err
	}
	var results []*v1.ExportImageResponse
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
