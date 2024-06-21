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
