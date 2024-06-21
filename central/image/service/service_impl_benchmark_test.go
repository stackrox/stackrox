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
	/*
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
		imageIDs := make([]string, 0)
		imageNamesByID := make(map[string]*storage.ImageName)
		for ix, delta := range deltas {
			fmt.Println(time.Now().UTC().Unix(), "Injecting", delta, "images")
			addedImageIDs, addedImageNamesByID, err := testHelper.InjectImages(b, delta)
			if err != nil {
				b.Error(err)
			}
			imageIDs = append(imageIDs, addedImageIDs...)
			for imageID, imageName := range addedImageNamesByID {
				imageNamesByID[imageID] = imageName
			}
			// Inject deployments to map images to namespaces and allow filtering query to work.
			fmt.Println(time.Now().UTC().Unix(), "Injecting", delta, "deployments")
			err = testHelper.InjectDeployments(b, delta, imageIDs, imageNamesByID)
			if err != nil {
				b.Error(err)
			}
			fmt.Println(time.Now().UTC().Unix(), "Test iteration", ix+1)
			total += delta
			b.Run(fmt.Sprintf("%d", total), getExportServiceBenchmark(testHelper, svc))
		}

	*/
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
