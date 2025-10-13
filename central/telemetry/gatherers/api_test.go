package gatherers

import (
	"testing"

	"github.com/stackrox/rox/pkg/grpc/metrics/mocks"
	"github.com/stackrox/rox/pkg/telemetry/data"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
)

var (
	mockGRPCCalls = map[string]map[codes.Code]int64{
		"test.grpc.path": {
			codes.OK: 1227,
		},
	}
	mockGRPCPanics = map[string]map[string]int64{
		"otherTest.grpc.path": {
			"Joseph Rules": 1337,
		},
	}

	mockHTTPCalls = map[string]map[int]int64{
		"combine stats": {
			420: 1227,
		},
	}
	mockHTTPPanics = map[string]map[string]int64{
		"combine stats": {
			"Joseph Rules": 1337,
		},
	}
)

func TestAPIMetrics(t *testing.T) {
	suite.Run(t, new(apiGathererTestSuite))
}

type apiGathererTestSuite struct {
	suite.Suite

	gatherer        *apiGatherer
	mockGRPCMetrics *mocks.MockGRPCMetrics
	mockHTTPMetrics *mocks.MockHTTPMetrics
	mockCtrl        *gomock.Controller
}

func (s *apiGathererTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockGRPCMetrics = mocks.NewMockGRPCMetrics(s.mockCtrl)
	s.mockHTTPMetrics = mocks.NewMockHTTPMetrics(s.mockCtrl)
	s.gatherer = newAPIGatherer(s.mockGRPCMetrics, s.mockHTTPMetrics)
}

func (s *apiGathererTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func getTestableStats(apiStats *data.APIStats) (*data.GRPCMethod, *data.GRPCMethod, *data.HTTPRoute) {
	var apiStat, apiPanic *data.GRPCMethod
	httpCombined := apiStats.HTTP[0]
	for _, stat := range apiStats.GRPC {
		if len(stat.NormalInvocations) > 0 {
			apiStat = stat
			continue
		}
		if len(stat.PanicInvocations) > 0 {
			apiPanic = stat
			continue
		}
	}
	return apiStat, apiPanic, httpCombined
}

func (s *apiGathererTestSuite) TestAPIGatherer() {
	s.mockGRPCMetrics.EXPECT().GetMetrics().Return(mockGRPCCalls, mockGRPCPanics)
	s.mockHTTPMetrics.EXPECT().GetMetrics().Return(mockHTTPCalls, mockHTTPPanics)
	apiStats := s.gatherer.Gather()
	s.NotNil(apiStats)

	s.Len(apiStats.GRPC, 2)
	s.Len(apiStats.HTTP, 1)

	grpcStat, grpcPanic, httpCombined := getTestableStats(apiStats)
	s.Equal("test.grpc.path", grpcStat.Method)
	s.Len(grpcStat.NormalInvocations, 1)
	grpcNormalInvocation := grpcStat.NormalInvocations[0]
	s.Equal(codes.OK, grpcNormalInvocation.Code)
	s.EqualValues(1227, grpcNormalInvocation.Count)

	s.Equal("otherTest.grpc.path", grpcPanic.Method)
	s.Len(grpcPanic.PanicInvocations, 1)
	grpcPanicInvocation := grpcPanic.PanicInvocations[0]
	s.Equal("Joseph Rules", grpcPanicInvocation.PanicDesc)
	s.EqualValues(1337, grpcPanicInvocation.Count)

	s.Equal("combine stats", httpCombined.Route)
	s.Len(httpCombined.NormalInvocations, 1)
	s.Len(httpCombined.PanicInvocations, 1)
	httpStat := httpCombined.NormalInvocations[0]
	httpPanic := httpCombined.PanicInvocations[0]
	s.Equal(420, httpStat.StatusCode)
	s.Equal(int64(1227), httpStat.Count)
	s.Equal("Joseph Rules", httpPanic.PanicDesc)
	s.Equal(int64(1337), httpPanic.Count)
}
