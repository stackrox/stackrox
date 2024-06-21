//go:build sql_integration

package service

import (
	"fmt"
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
	svc := New(testHelper.Deployments, testHelper.Images)
	benchmarkFunc := getExportServiceBenchmark(testHelper, svc)
	testHelper.InjectDataAndRunBenchmark(b, true, benchmarkFunc)
	/*
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
		imageNamesByIDs := make(map[string]*storage.ImageName)

		total := 0
		for ix, delta := range deltas {
			total += delta
			fmt.Println(time.Now().UTC().Unix(), "Injecting", delta, "images")
			addedImageIDs, addedImageNamesByID, err := testHelper.InjectImages(b, delta)
			if err != nil {
				b.Error(err)
			}
			imageIDs = append(imageIDs, addedImageIDs...)
			for imageID, imageName := range addedImageNamesByID {
				imageNamesByIDs[imageID] = imageName
			}
			fmt.Println(time.Now().UTC().Unix(), "Injecting", delta, "deployments")
			err = testHelper.InjectDeployments(b, delta, imageIDs, imageNamesByIDs)
			if err != nil {
				b.Error(err)
			}
			fmt.Println(time.Now().UTC().Unix(), "Test iteration", ix+1)
			b.Run(fmt.Sprintf("%d", total), getExportServiceBenchmark(testHelper, svc))
		}
	
	*/
}

func getExportServiceBenchmark(
	helper *testutils.ExportServicePostgresTestHelper,
	service Service,
) func(b *testing.B) {
	return func(b *testing.B) {
		conn, closeFunc, err := helper.CreateGRPCStreamingService(
			b,
			func(registrar grpc.ServiceRegistrar) {
				v1.RegisterVulnMgmtServiceServer(registrar, service)
			},
		)
		if err != nil {
			b.Error(err)
		}
		defer closeFunc()

		testCases := testutils.GetBaseTestCases()
		for _, testCase := range testCases {
			b.Run(testCase.Name, func(ib *testing.B) {
				request := &v1.VulnMgmtExportWorkloadsRequest{Timeout: 3600}
				if testCase.TargetNamespace != "" {
					request.Query = fmt.Sprintf("Namespace:%s", testCase.TargetNamespace)
				}

				client := v1.NewVulnMgmtServiceClient(conn)
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
