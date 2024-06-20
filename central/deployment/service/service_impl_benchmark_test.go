//go:build sql_integration

package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stackrox/rox/central/testutils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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

	total := 0
	deltas := []int{500}
	// The test runs by default with a lower scale as smoke test
	// in the benchmark unit tests. To test at higher scales (takes time),
	// run the test with ROX_SCALE_TEST set to a non-empty value
	// in the test environment.
	scale := os.Getenv("ROX_SCALE_TEST")
	if scale != "" {
		deltas = []int{500, 500, 1000, 3000, 5000}
	}
	baseImages, err := testutils.GetBaseImageSet()
	if err != nil {
		b.Error(err)
	}
	imageIDs := make([]string, 0, len(baseImages))
	imageNamesByID := make(map[string]*storage.ImageName, len(baseImages))
	for _, img := range baseImages {
		imageID := img.GetId()
		imageName := img.GetName()
		imageIDs = append(imageIDs, imageID)
		imageNamesByID[imageID] = imageName
	}
	for _, delta := range deltas {
		total += delta
		err := testHelper.InjectDeployments(b, delta, imageIDs, imageNamesByID)
		if err != nil {
			b.Error(err)
		}
		b.Run(fmt.Sprintf("%d", total), getExportServiceBenchmark(testHelper, svc))
	}
}

/*
Results obtained from a local run with optimisation of the cached store query behaviour:

 DB Size        | No query         | Query 10% of db  | Query 90% of db  |
 (# deployment) | Elapsed ns       | Elapsed ns       | Elapsed ns       |
----------------+------------------+------------------+------------------+
            500 |       10.191.996 |        3.237.156 |       10.658.958 |
 | 20.384 | 64.743 | 23.686 |
----------------+------------------+------------------+------------------+
          1.000 |       26.700.841 |        9.303.390 |       19.931.979 |
                |           26.701 |           93.034 |           22.147 |
----------------+------------------+------------------+------------------+
          2.000 |       38.337.124 |        9.265.710 |       39.797.756 |
                |           19.169 |           46.329 |           22.110 |
----------------+------------------+------------------+------------------+
          5.000 |       93.571.165 |       32.733.619 |      167.714.609 |
                |           18.714 |           65.467 |           37.269 |
----------------+------------------+------------------+------------------+
         10.000 |      212.681.414 |       48.441.410 |      362.524.211 |
                |           21.268 |           48.441 |           40.280 |
----------------+------------------+------------------+------------------+
         20.000 |      521.376.049 |       80.402.127 |      545.397.712 |
                |           26.068 |           40.201 |           30.294 |
----------------+------------------+------------------+------------------+
*/

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
