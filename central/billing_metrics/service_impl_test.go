package billingmetrics

import (
	"context"
	"testing"

	store "github.com/stackrox/rox/central/billing_metrics/store"
	mockstore "github.com/stackrox/rox/central/billing_metrics/store/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

type maximusSvcSuite struct {
	suite.Suite

	store store.Store
	ctx   context.Context
}

func (s *maximusSvcSuite) SetupTest() {
	mockCtrl := gomock.NewController(s.T())
	s.store = mockstore.NewMockStore(mockCtrl)
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *maximusSvcSuite) TestGetMaximumValue(t *testing.T) {
	svc := New(s.store)
	req := &v1.MaximumValueRequest{Metric: "test"}
	svc.GetMaximum(s.ctx, &req)
}
