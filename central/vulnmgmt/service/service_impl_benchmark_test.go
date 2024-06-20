//go:build sql_integration

package service

import (
	"fmt"
	"os"
	"testing"
	"time"

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
	svc := New(testHelper.Deployments, testHelper.Images)
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
}

/*
Sample run outcome.

Data injection phase (bench results removed from output)

BenchmarkService_Export
1718789959 Injecting base images
1718790077 Injecting base deployments
1718790082 Starting actual tests
... Results with a 500 item DB
1718790102 Injecting 500 extra images
1718790179 Injecting 500 extra deployments
1718790183 Continuing actual tests
... Results with a 1.000 item DB
1718790221 Injecting 4000 extra images
1718791150 Injecting 4000 extra deployments
1718791190 Continuing actual tests (2)
... Results with a 5.000 item DB
1718791574 Injecting 5000 extra images
1718792407 Injecting 5000 extra deployments
1718792448 Continuing actual tests (3)
... Results with a 10.000 item DB

Benchmark results

 DB Size        | No query         | Query 10% of db  | Query 90% of db  |
 (# deployment  |                  |                  |                  |
 and # images)  | Elapsed ns       | Elapsed ns       | Elapsed ns       |
----------------+------------------+------------------+------------------+
            500 |    9.165.222.465 |    1.896.871.287 |    9.100.526.550 |
                |       18.330.445 |       37.937.426 |       20.223.392 |
----------------+------------------+------------------+------------------+
          1.000 |   17.562.401.077 |    3.695.793.796 |   16.655.958.011 |
                |       17.562.401 |       36.957.938 |       18.503.287 |
----------------+------------------+------------------+------------------+
          5.000 |  232.410.473.350 |   17.445.117.452 |  133.301.396.811 |
                |       46.482.095 |       34.890.235 |       29.622.533 |
----------------+------------------+------------------+------------------+
         10.000 |  385.138.425.396 |   35.631.981.696 |  302.740.800.784 |
                |       38.513.843 |       35.631.981 |       33.637.867 |
----------------+------------------+------------------+------------------+

The structure of the service call is:
- Walk by query on deployments
  - pull elements directly from cache if empty query
  - pull elements from DB otherwise
  [ for each deployment :
    - for each container in the deployment:
      - identify the container image ID
      - get the container image from cache
      - on cache miss, get the container image from DB
        - get the image metadata from DB (table images)
        - get ImageCVEEdge objects (table image_cve_edges)
        - get CVE IDs from the edge objects
        - get ImageComponentEdge objects (table image_component_edges)
        - get component IDs from edge objects
        - get ImageComponent objects (table image_components)
        - get (Image)ComponentCVEEdge objects (table image_component_cve_edges)
        - get ImageCVE objects (table image_cves)
        - reconstruct image object from fetched elements
  ]
*/

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
						b.Error(err)
					}
				}
			})
		}
	}
}
