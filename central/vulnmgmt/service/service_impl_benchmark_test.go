//go:build sql_integration

package service

import (
	"fmt"
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

	svc := New(testHelper.Deployments, testHelper.Images)
	benchmarkFunc := getExportServiceBenchmark(testHelper, svc)
	testHelper.InjectDataAndRunBenchmark(b, true, benchmarkFunc)
}

func getExportServiceBenchmark(
	helper *testutils.ExportServicePostgresTestHelper,
	service Service,
) func(b *testing.B) {
	return func(b *testing.B) {
		conn, closeFunc, err := pkgGRPC.CreateTestGRPCStreamingService(
			helper.Ctx,
			b,
			func(registrar grpc.ServiceRegistrar) {
				v1.RegisterVulnMgmtServiceServer(registrar, service)
			},
		)
		if err != nil {
			b.Error(err)
		}
		defer closeFunc()

		testCases := testutils.GetExportTestCases()
		for _, testCase := range testCases {
			b.Run(testCase.Name, func(ib *testing.B) {
				request := &v1.VulnMgmtExportWorkloadsRequest{Timeout: 3600}
				if testCase.TargetNamespace != "" {
					request.Query = fmt.Sprintf("Namespace:%s", testCase.TargetNamespace)
				}

				client := v1.NewVulnMgmtServiceClient(conn)
				ib.ResetTimer()
				for i := 0; i < ib.N; i++ {
					_, err = receiveWorkloads(helper.Ctx, ib, client, request, true)
					if err != nil {
						ib.Error(err)
					}
				}
			})
		}
	}
}
