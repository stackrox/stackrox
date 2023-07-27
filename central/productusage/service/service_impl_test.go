package usage

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	mockstore "github.com/stackrox/rox/central/productusage/datastore/securedunits/mocks"
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
	now := time.Now()
	ts := protoconv.ConvertTimeToTimestamp(now)
	ts1 := protoconv.ConvertTimeToTimestamp(now.Add(1 * time.Hour))
	ts2 := protoconv.ConvertTimeToTimestamp(now.Add(2 * time.Hour))

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

	req := &v1.TimeRange{From: ts, To: ts2}

	s.store.EXPECT().Walk(context.Background(), ts, ts2, gomock.Any()).Times(1).
		DoAndReturn(
			func(_ context.Context, _ *types.Timestamp, _ *types.Timestamp, fn func(*storage.SecuredUnits) error) error {
				_ = fn(stored[0])
				_ = fn(stored[1])
				return nil
			},
		)
	svc := New(s.store)
	res, err := svc.GetMaxSecuredUnitsUsage(context.Background(), req)
	s.Require().NoError(err)
	s.Equal(exp, res)
}

func (s *usageSvcSuite) TestGetCurrentUsage() {
	now := time.Now()
	ts := protoconv.ConvertTimeToTimestamp(now)
	ts1 := protoconv.ConvertTimeToTimestamp(now.Add(1 * time.Hour))
	ts2 := protoconv.ConvertTimeToTimestamp(now.Add(2 * time.Hour))

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

	req := &v1.TimeRange{From: ts, To: ts2}

	s.store.EXPECT().Walk(context.Background(), ts, ts2, gomock.Any()).Times(1).
		DoAndReturn(
			func(_ context.Context, _ *types.Timestamp, _ *types.Timestamp, fn func(*storage.SecuredUnits) error) error {
				_ = fn(stored[0])
				_ = fn(stored[1])
				return nil
			},
		)
	svc := New(s.store)
	res, err := svc.GetMaxSecuredUnitsUsage(context.Background(), req)
	s.Require().NoError(err)
	s.Equal(exp, res)
}
