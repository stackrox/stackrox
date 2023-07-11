package billingmetrics

import (
	"context"
	"testing"
	"time"

	mockstore "github.com/stackrox/rox/central/billing_metrics/store/mocks"
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

type maximusSvcSuite struct {
	suite.Suite

	store *mockstore.MockStore
	ctx   context.Context
}

var _ suite.SetupTestSuite = (*maximusSvcSuite)(nil)

func (s *maximusSvcSuite) SetupTest() {
	mockCtrl := gomock.NewController(s.T())
	s.store = mockstore.NewMockStore(mockCtrl)
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *maximusSvcSuite) TestGetMaximum() {
	exp := &v1.MaximumValueResponse{Metric: "test", Value: 10, Ts: protoconv.ConvertTimeToTimestamp(time.Time{})}
	req := &v1.MaximumValueRequest{Metric: "test"}
	s.store.EXPECT().Get(s.ctx, req).Times(1).Return(exp, true, nil)
	svc := New(s.store)
	res, err := svc.GetMaximum(s.ctx, req)
	s.Require().NoError(err)
	s.Equal(exp, res)

	s.store.EXPECT().Get(s.ctx, req).Times(1).Return(nil, false, nil)
	res, err = svc.GetMaximum(s.ctx, req)
	s.Require().NoError(err)
	s.Nil(res)
}

func (s *maximusSvcSuite) TestDeleteMaximum() {
	req := &v1.MaximumValueRequest{Metric: "test"}
	s.store.EXPECT().Delete(s.ctx, req).Times(1).Return(nil, nil)
	svc := New(s.store)
	res, err := svc.DeleteMaximum(s.ctx, req)
	s.Require().NoError(err)
	s.Nil(res)
}

func (s *maximusSvcSuite) TestPostMaximum() {
	req := &v1.MaximumValueUpdateRequest{Metric: "test", Value: 20, Ts: protoconv.ConvertTimeToTimestamp(time.Time{})}
	s.store.EXPECT().Upsert(s.ctx, req).Times(1).Return(nil, nil)
	svc := New(s.store)
	res, err := svc.PostMaximum(s.ctx, req)
	s.Require().NoError(err)
	s.Nil(res)
}
