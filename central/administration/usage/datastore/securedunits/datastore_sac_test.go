//go:build sql_integration

package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/administration/usage/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestSecuredUnitsDatastoreSAC(t *testing.T) {
	suite.Run(t, new(securedUnitsDatastoreSACSuite))
}

type securedUnitsDatastoreSACSuite struct {
	suite.Suite

	datastore DataStore

	pgTestBase *pgtest.TestPostgres

	testContexts map[string]context.Context

	mockCtrl      *gomock.Controller
	mockClusterDS *MockclusterDataStore
}

func (s *securedUnitsDatastoreSACSuite) SetupSuite() {
	s.pgTestBase = pgtest.ForT(s.T())
	s.NotNil(s.pgTestBase)

	s.testContexts = testutils.GetGloballyScopedTestContexts(
		context.Background(),
		s.T(),
		resources.Integration, // Alternative resource
		resources.Administration,
	)
}

func (s *securedUnitsDatastoreSACSuite) TearDownSuite() {
	s.pgTestBase.DB.Close()
}

func (s *securedUnitsDatastoreSACSuite) SetupTest() {
	// Truncate the table before each test
	ctx := sac.WithAllAccess(context.Background())
	_, err := s.pgTestBase.Exec(ctx, "TRUNCATE secured_units CASCADE")
	s.Require().NoError(err)

	// Create mock cluster datastore
	s.mockCtrl = gomock.NewController(s.T())
	s.mockClusterDS = NewMockclusterDataStore(s.mockCtrl)

	// Configure mock to return empty cluster list by default
	s.mockClusterDS.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{}, nil).AnyTimes()

	// Create new datastore instance
	store := postgres.New(s.pgTestBase.DB)
	s.datastore = New(store, s.mockClusterDS)
}

func (s *securedUnitsDatastoreSACSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

// Helper functions

func getTestSecuredUnits() *storage.SecuredUnits {
	return &storage.SecuredUnits{
		Id:          uuid.NewV4().String(),
		Timestamp:   protocompat.TimestampNow(),
		NumNodes:    10,
		NumCpuUnits: 100,
	}
}

func getTestSecuredUnitsWithTime(t time.Time) *storage.SecuredUnits {
	return &storage.SecuredUnits{
		Id:          uuid.NewV4().String(),
		Timestamp:   protocompat.ConvertTimeToTimestampOrNil(&t),
		NumNodes:    10,
		NumCpuUnits: 100,
	}
}

// Test cases for Add operation (write)

func (s *securedUnitsDatastoreSACSuite) TestAdd() {
	cases := testutils.GenericGlobalSACWriteTestCases(testutils.VerbAdd)

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			testUnits := getTestSecuredUnits()
			err := s.datastore.Add(ctx, testUnits)

			if c.ExpectError {
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)

				// Verify the object was actually added
				unrestrictedCtx := sac.WithAllAccess(context.Background())
				var found *storage.SecuredUnits
				walkErr := s.datastore.Walk(unrestrictedCtx, time.Time{}, time.Time{}, func(obj *storage.SecuredUnits) error {
					if obj.GetId() == testUnits.GetId() {
						found = obj
					}
					return nil
				})
				s.NoError(walkErr)
				s.NotNil(found)
				protoassert.Equal(s.T(), testUnits, found)
			}
		})
	}
}

// Test cases for Walk operation (read)

func (s *securedUnitsDatastoreSACSuite) TestWalk() {
	// Setup: Add test data with unrestricted access
	unrestrictedCtx := sac.WithAllAccess(context.Background())
	testUnits := getTestSecuredUnits()
	err := s.datastore.Add(unrestrictedCtx, testUnits)
	s.Require().NoError(err)

	cases := testutils.GenericGlobalSACReadTestCases("walk")

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			var count int
			err := s.datastore.Walk(ctx, time.Time{}, time.Time{}, func(obj *storage.SecuredUnits) error {
				count++
				protoassert.Equal(s.T(), testUnits, obj)
				return nil
			})

			if c.ExpectError {
				s.ErrorIs(err, c.ExpectedError)
				s.Equal(0, count, "Should not have iterated over any objects")
			} else {
				s.NoError(err)
				if c.ExpectedFound {
					s.Equal(1, count, "Should have found exactly one object")
				} else {
					s.Equal(0, count, "Should not have found any objects")
				}
			}
		})
	}
}

// Test cases for GetMaxNumNodes operation (read)

func (s *securedUnitsDatastoreSACSuite) TestGetMaxNumNodes() {
	// Setup: Add test data with varying NumNodes
	unrestrictedCtx := sac.WithAllAccess(context.Background())
	baseTime := time.Now().Add(-1 * time.Hour)

	units1 := getTestSecuredUnitsWithTime(baseTime.Add(10 * time.Minute))
	units1.NumNodes = 5
	err := s.datastore.Add(unrestrictedCtx, units1)
	s.Require().NoError(err)

	units2 := getTestSecuredUnitsWithTime(baseTime.Add(20 * time.Minute))
	units2.NumNodes = 15 // Maximum
	err = s.datastore.Add(unrestrictedCtx, units2)
	s.Require().NoError(err)

	units3 := getTestSecuredUnitsWithTime(baseTime.Add(30 * time.Minute))
	units3.NumNodes = 10
	err = s.datastore.Add(unrestrictedCtx, units3)
	s.Require().NoError(err)

	cases := testutils.GenericGlobalSACReadTestCases("get max num nodes")

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			result, err := s.datastore.GetMaxNumNodes(ctx, time.Time{}, time.Time{})

			if c.ExpectError {
				s.ErrorIs(err, c.ExpectedError)
				s.Nil(result)
			} else {
				s.NoError(err)
				if c.ExpectedFound {
					s.NotNil(result)
					s.Equal(int64(15), result.GetNumNodes(), "Should return the record with maximum NumNodes")
					protoassert.Equal(s.T(), units2, result)
				} else {
					s.Nil(result)
				}
			}
		})
	}
}

// Test cases for GetMaxNumCPUUnits operation (read)

func (s *securedUnitsDatastoreSACSuite) TestGetMaxNumCPUUnits() {
	// Setup: Add test data with varying NumCpuUnits
	unrestrictedCtx := sac.WithAllAccess(context.Background())
	baseTime := time.Now().Add(-1 * time.Hour)

	units1 := getTestSecuredUnitsWithTime(baseTime.Add(10 * time.Minute))
	units1.NumCpuUnits = 50
	err := s.datastore.Add(unrestrictedCtx, units1)
	s.Require().NoError(err)

	units2 := getTestSecuredUnitsWithTime(baseTime.Add(20 * time.Minute))
	units2.NumCpuUnits = 200 // Maximum
	err = s.datastore.Add(unrestrictedCtx, units2)
	s.Require().NoError(err)

	units3 := getTestSecuredUnitsWithTime(baseTime.Add(30 * time.Minute))
	units3.NumCpuUnits = 100
	err = s.datastore.Add(unrestrictedCtx, units3)
	s.Require().NoError(err)

	cases := testutils.GenericGlobalSACReadTestCases("get max num cpu units")

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			result, err := s.datastore.GetMaxNumCPUUnits(ctx, time.Time{}, time.Time{})

			if c.ExpectError {
				s.ErrorIs(err, c.ExpectedError)
				s.Nil(result)
			} else {
				s.NoError(err)
				if c.ExpectedFound {
					s.NotNil(result)
					s.Equal(int64(200), result.GetNumCpuUnits(), "Should return the record with maximum NumCpuUnits")
					protoassert.Equal(s.T(), units2, result)
				} else {
					s.Nil(result)
				}
			}
		})
	}
}

// Test cases for GetCurrentUsage operation (read)

func (s *securedUnitsDatastoreSACSuite) TestGetCurrentUsage() {
	// Note: GetCurrentUsage reads from the in-memory cache, not persistent storage
	// The cache is populated via UpdateUsage calls

	cases := testutils.GenericGlobalSACReadTestCasesNoAccessNoError("get current usage")

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			result, err := s.datastore.GetCurrentUsage(ctx)

			if c.ExpectError {
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
				// The cache is empty initially, so result will be a zero-valued SecuredUnits
				s.NotNil(result)
			}
		})
	}
}

// Test cases for AggregateAndReset operation (write)

func (s *securedUnitsDatastoreSACSuite) TestAggregateAndReset() {
	// Setup: Populate the cache with some data via UpdateUsage
	unrestrictedCtx := sac.WithAllAccess(context.Background())
	testUnits := getTestSecuredUnits()
	testUnits.NumNodes = 5
	testUnits.NumCpuUnits = 50
	err := s.datastore.UpdateUsage(unrestrictedCtx, "test-cluster-1", testUnits)
	s.Require().NoError(err)

	cases := testutils.GenericGlobalSACWriteTestCases("aggregate and reset")

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			result, err := s.datastore.AggregateAndReset(ctx)

			if c.ExpectError {
				s.ErrorIs(err, c.ExpectedError)
				s.Nil(result)
			} else {
				s.NoError(err)
				s.NotNil(result)
				// After a successful aggregate, the cache should be reset
				// and the next call should return zero values
				result2, err2 := s.datastore.GetCurrentUsage(unrestrictedCtx)
				s.NoError(err2)
				s.NotNil(result2)
				s.Equal(int64(0), result2.GetNumNodes())
				s.Equal(int64(0), result2.GetNumCpuUnits())
			}

			// Re-populate cache for next test iteration
			if !c.ExpectError {
				err = s.datastore.UpdateUsage(unrestrictedCtx, "test-cluster-1", testUnits)
				s.Require().NoError(err)
			}
		})
	}
}

// Test cases for UpdateUsage operation (write)

func (s *securedUnitsDatastoreSACSuite) TestUpdateUsage() {
	cases := testutils.GenericGlobalSACWriteTestCases("update usage")

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			testUnits := getTestSecuredUnits()
			testUnits.NumNodes = 7
			testUnits.NumCpuUnits = 70
			err := s.datastore.UpdateUsage(ctx, "test-cluster-1", testUnits)

			if c.ExpectError {
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)

				// Verify the cache was updated
				unrestrictedCtx := sac.WithAllAccess(context.Background())
				current, err := s.datastore.GetCurrentUsage(unrestrictedCtx)
				s.NoError(err)
				s.NotNil(current)
				s.Equal(testUnits.GetNumNodes(), current.GetNumNodes())
				s.Equal(testUnits.GetNumCpuUnits(), current.GetNumCpuUnits())

				// Reset cache for next iteration
				_, err = s.datastore.AggregateAndReset(unrestrictedCtx)
				s.NoError(err)
			}
		})
	}
}

// Test time range filtering for Walk

func (s *securedUnitsDatastoreSACSuite) TestWalkTimeRange() {
	unrestrictedCtx := sac.WithAllAccess(context.Background())
	baseTime := time.Now().Add(-2 * time.Hour)

	// Add data at different times
	units1 := getTestSecuredUnitsWithTime(baseTime.Add(10 * time.Minute))
	err := s.datastore.Add(unrestrictedCtx, units1)
	s.Require().NoError(err)

	units2 := getTestSecuredUnitsWithTime(baseTime.Add(1 * time.Hour))
	err = s.datastore.Add(unrestrictedCtx, units2)
	s.Require().NoError(err)

	units3 := getTestSecuredUnitsWithTime(baseTime.Add(90 * time.Minute))
	err = s.datastore.Add(unrestrictedCtx, units3)
	s.Require().NoError(err)

	// Test with read access - should be able to walk within time range
	readCtx := s.testContexts[testutils.UnrestrictedReadCtx]
	var count int
	from := baseTime
	to := baseTime.Add(70 * time.Minute)
	err = s.datastore.Walk(readCtx, from, to, func(obj *storage.SecuredUnits) error {
		count++
		return nil
	})
	s.NoError(err)
	s.Equal(2, count, "Should find exactly 2 objects within time range")

	// Test with no access - should fail
	noAccessCtx := s.testContexts[testutils.NoAccessCtx]
	count = 0
	err = s.datastore.Walk(noAccessCtx, from, to, func(obj *storage.SecuredUnits) error {
		count++
		return nil
	})
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	s.Equal(0, count, "Should not iterate with no access")
}

// Test time range filtering for GetMaxNumNodes

func (s *securedUnitsDatastoreSACSuite) TestGetMaxNumNodesTimeRange() {
	unrestrictedCtx := sac.WithAllAccess(context.Background())
	baseTime := time.Now().Add(-2 * time.Hour)

	// Add data at different times with different node counts
	units1 := getTestSecuredUnitsWithTime(baseTime.Add(10 * time.Minute))
	units1.NumNodes = 20 // Max overall, but outside range
	err := s.datastore.Add(unrestrictedCtx, units1)
	s.Require().NoError(err)

	units2 := getTestSecuredUnitsWithTime(baseTime.Add(1 * time.Hour))
	units2.NumNodes = 15 // Max within range
	err = s.datastore.Add(unrestrictedCtx, units2)
	s.Require().NoError(err)

	units3 := getTestSecuredUnitsWithTime(baseTime.Add(90 * time.Minute))
	units3.NumNodes = 10 // Within range but not max
	err = s.datastore.Add(unrestrictedCtx, units3)
	s.Require().NoError(err)

	// Test with read access and time range
	readCtx := s.testContexts[testutils.UnrestrictedReadCtx]
	from := baseTime.Add(30 * time.Minute)
	to := baseTime.Add(2 * time.Hour)
	result, err := s.datastore.GetMaxNumNodes(readCtx, from, to)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(int64(15), result.GetNumNodes(), "Should return max within time range, not overall max")
}

// Test time range filtering for GetMaxNumCPUUnits

func (s *securedUnitsDatastoreSACSuite) TestGetMaxNumCPUUnitsTimeRange() {
	unrestrictedCtx := sac.WithAllAccess(context.Background())
	baseTime := time.Now().Add(-2 * time.Hour)

	// Add data at different times with different CPU counts
	units1 := getTestSecuredUnitsWithTime(baseTime.Add(10 * time.Minute))
	units1.NumCpuUnits = 300 // Max overall, but outside range
	err := s.datastore.Add(unrestrictedCtx, units1)
	s.Require().NoError(err)

	units2 := getTestSecuredUnitsWithTime(baseTime.Add(1 * time.Hour))
	units2.NumCpuUnits = 150 // Max within range
	err = s.datastore.Add(unrestrictedCtx, units2)
	s.Require().NoError(err)

	units3 := getTestSecuredUnitsWithTime(baseTime.Add(90 * time.Minute))
	units3.NumCpuUnits = 100 // Within range but not max
	err = s.datastore.Add(unrestrictedCtx, units3)
	s.Require().NoError(err)

	// Test with read access and time range
	readCtx := s.testContexts[testutils.UnrestrictedReadCtx]
	from := baseTime.Add(30 * time.Minute)
	to := baseTime.Add(2 * time.Hour)
	result, err := s.datastore.GetMaxNumCPUUnits(readCtx, from, to)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(int64(150), result.GetNumCpuUnits(), "Should return max within time range, not overall max")
}
