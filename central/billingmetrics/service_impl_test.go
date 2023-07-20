package billingmetrics

import (
	"context"
	"testing"
	"time"

	mockstore "github.com/stackrox/rox/central/billingmetrics/store/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

type billingMetricsSvcSuite struct {
	suite.Suite

	store *mockstore.MockStore
	ctx   context.Context
}

func TestService(t *testing.T) {
	suite.Run(t, new(billingMetricsSvcSuite))
}

var _ suite.SetupTestSuite = (*billingMetricsSvcSuite)(nil)

func (s *billingMetricsSvcSuite) SetupTest() {
	mockCtrl := gomock.NewController(s.T())
	s.store = mockstore.NewMockStore(mockCtrl)
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *billingMetricsSvcSuite) TestGetMetrics() {
	now := time.Now()
	ts := protoconv.ConvertTimeToTimestamp(now)
	ts1 := protoconv.ConvertTimeToTimestamp(now.Add(1 * time.Hour))
	ts2 := protoconv.ConvertTimeToTimestamp(now.Add(2 * time.Hour))

	stored := []storage.BillingMetrics{{
		Ts: ts,
		Sr: &storage.BillingMetrics_SecuredResources{
			Nodes:      5,
			Millicores: 2,
		},
	}, {
		Ts: ts1,
		Sr: &storage.BillingMetrics_SecuredResources{
			Nodes:      1,
			Millicores: 100,
		},
	}}

	exp := &v1.BillingMetricsResponse{Record: []*v1.BillingMetricsResponse_BillingMetricsRecord{{
		Ts: ts,
		Metrics: &v1.SecuredResourcesMetrics{
			Nodes:      5,
			Millicores: 2,
		},
	}, {
		Ts: ts1,
		Metrics: &v1.SecuredResourcesMetrics{
			Nodes:      1,
			Millicores: 100,
		},
	}}}

	req := &v1.BillingMetricsRequest{
		From: ts,
		To:   ts2}

	s.store.EXPECT().Get(s.ctx, gomock.AssignableToTypeOf(ts), gomock.AssignableToTypeOf(ts2)).Times(1).Return(stored, nil)
	svc := New(s.store)
	res, err := svc.GetMetrics(s.ctx, req)
	s.Require().NoError(err)
	s.Equal(exp, res)
}

func (s *billingMetricsSvcSuite) TestGetMax() {
	now := time.Now()
	ts := protoconv.ConvertTimeToTimestamp(now)
	ts1 := protoconv.ConvertTimeToTimestamp(now.Add(1 * time.Hour))
	ts2 := protoconv.ConvertTimeToTimestamp(now.Add(2 * time.Hour))

	stored := []storage.BillingMetrics{{
		Ts: ts,
		Sr: &storage.BillingMetrics_SecuredResources{
			Nodes:      5,
			Millicores: 2,
		},
	}, {
		Ts: ts1,
		Sr: &storage.BillingMetrics_SecuredResources{
			Nodes:      1,
			Millicores: 100,
		},
	}}

	exp := &v1.BillingMetricsMaxResponse{
		NodesTs:      ts,
		Nodes:        5,
		MillicoresTs: ts1,
		Millicores:   100,
	}

	req := &v1.BillingMetricsRequest{From: ts, To: ts2}

	s.store.EXPECT().Get(s.ctx, gomock.AssignableToTypeOf(ts), gomock.AssignableToTypeOf(ts2)).Times(1).Return(stored, nil)
	svc := New(s.store)
	res, err := svc.GetMax(s.ctx, req)
	s.Require().NoError(err)
	s.Equal(exp, res)
}
