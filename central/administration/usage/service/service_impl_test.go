package service

import (
	"context"
	"testing"
	"time"

	mockstore "github.com/stackrox/rox/central/administration/usage/datastore/securedunits/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

type usageSvcSuite struct {
	suite.Suite

	store *mockstore.MockDataStore
}

func TestService(t *testing.T) {
	suite.Run(t, new(usageSvcSuite))
}

var _ suite.SetupTestSuite = (*usageSvcSuite)(nil)

func (s *usageSvcSuite) SetupTest() {
	mockCtrl := gomock.NewController(s.T())
	s.store = mockstore.NewMockDataStore(mockCtrl)
}

func (s *usageSvcSuite) TestGetMaxUsage() {
	from := time.Now()
	to := from.Add(2 * time.Hour)

	ts := protoconv.ConvertTimeToTimestamp(from)
	ts1 := protoconv.ConvertTimeToTimestamp(from.Add(1 * time.Hour))

	stored := []*storage.SecuredUnits{{
		Timestamp:   ts,
		NumNodes:    5,
		NumCpuUnits: 2,
	}, {
		Timestamp:   ts1,
		NumNodes:    1,
		NumCpuUnits: 100,
	}}

	exp := &v1.MaxSecuredUnitsUsageResponse{
		MaxNodesAt:    ts,
		MaxNodes:      5,
		MaxCpuUnitsAt: ts1,
		MaxCpuUnits:   100,
	}

	req := &v1.TimeRange{From: ts, To: protoconv.ConvertTimeToTimestamp(to)}

	s.store.EXPECT().GetMaxNumNodes(context.Background(),
		from.UTC(), to.UTC()).Times(1).Return(stored[0], nil)
	s.store.EXPECT().GetMaxNumCPUUnits(context.Background(),
		from.UTC(), to.UTC()).Times(1).Return(stored[1], nil)

	svc := New(s.store)
	res, err := svc.GetMaxSecuredUnitsUsage(context.Background(), req)
	s.Require().NoError(err)
	s.Equal(exp, res)
}

func (s *usageSvcSuite) TestGetCurrentUsage() {
	stored := &storage.SecuredUnits{
		NumNodes:    5,
		NumCpuUnits: 2,
	}

	exp := &v1.SecuredUnitsUsageResponse{
		NumNodes:    5,
		NumCpuUnits: 2,
	}

	s.store.EXPECT().GetCurrentUsage(context.Background()).Times(1).
		Return(stored, nil)
	svc := New(s.store)
	res, err := svc.GetCurrentSecuredUnitsUsage(context.Background(), &v1.Empty{})
	s.Require().NoError(err)
	s.Equal(exp, res)
}
