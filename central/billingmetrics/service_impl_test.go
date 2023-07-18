package billingmetrics

import (
	"context"
	"testing"
	"time"

	mockstore "github.com/stackrox/rox/central/billingmetrics/store/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
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

var _ suite.SetupTestSuite = (*billingMetricsSvcSuite)(nil)

func (s *billingMetricsSvcSuite) SetupTest() {
	mockCtrl := gomock.NewController(s.T())
	s.store = mockstore.NewMockStore(mockCtrl)
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *billingMetricsSvcSuite) TestGetMetrics() {
	exp := &v1.BillingMetricsResponse{Record: []*v1.BillingMetricsResponse_BillingMetricsRecord{{
		Ts:      protoconv.ConvertTimeToTimestamp(time.Time{}),
		Metrics: &v1.SecuredResourcesMetrics{},
	}}}
	req := &v1.BillingMetricsRequest{
		From: protoconv.ConvertTimeToTimestamp(time.Time{}),
		To:   protoconv.ConvertTimeToTimestamp(time.Time{})}

	s.store.EXPECT().Get(s.ctx, nil, nil).Times(1).Return(exp, nil)
	svc := New(s.store)
	res, err := svc.GetMetrics(s.ctx, req)
	s.Require().NoError(err)
	s.Equal(exp, res)

	s.store.EXPECT().Get(s.ctx, nil, nil).Times(1).Return(nil, nil)
	res, err = svc.GetMetrics(s.ctx, req)
	s.Require().NoError(err)
	s.Nil(res)
}

func (s *billingMetricsSvcSuite) TestPutMetrics() {
	req := &v1.BillingMetricsInsertRequest{
		Ts:      protoconv.ConvertTimeToTimestamp(time.Time{}),
		Metrics: &v1.SecuredResourcesMetrics{Nodes: 5, Millicores: 50},
	}
	s.store.EXPECT().Insert(s.ctx, req).Times(1).Return(nil, nil)
	svc := New(s.store)
	res, err := svc.PutMetrics(s.ctx, req)
	s.Require().NoError(err)
	s.Nil(res)
}
