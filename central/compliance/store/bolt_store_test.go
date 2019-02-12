package store

import (
	"os"
	"testing"
	"time"

	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
)

type boltStoreTestSuite struct {
	suite.Suite

	db    *bbolt.DB
	store *boltStore
}

func TestBoltStore(t *testing.T) {
	s := new(boltStoreTestSuite)
	suite.Run(t, s)
}

func (s *boltStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(s.T().Name() + ".db")
	s.Require().NoError(err, "Failed to make BoltDB: %s", err)

	s.db = db
	s.store, err = newBoltStore(db)
	s.Require().NoError(err, "Failed to create store")
}

func (s *boltStoreTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
		os.Remove(s.db.Path())
	}
}

func (s *boltStoreTestSuite) SetupTest() {
	s.store.clear()
}

func (s *boltStoreTestSuite) TestGetOnEmpty() {
	results, err := s.store.GetLatestRunResults("foo", "bar", 0)
	s.Zero(results)
	s.Error(err)
}

func (s *boltStoreTestSuite) TestBatchGetOnEmpty() {
	results, err := s.store.GetLatestRunResultsBatch([]string{"cluster1"}, []string{"standard1, standard2"}, 0)
	s.NoError(err)
	s.Len(results, 0)
}

func (s *boltStoreTestSuite) TestFilteredGetOnEmpty() {
	truePred := func(string) bool { return true }
	results, err := s.store.GetLatestRunResultsFiltered(truePred, truePred, 0)
	s.NoError(err)
	s.Len(results, 0)
}

func (s *boltStoreTestSuite) TestStoreSuccesses() {
	time8am, _ := types.TimestampProto(time.Date(2019, 01, 16, 8, 0, 0, 0, time.UTC))
	time12pm, _ := types.TimestampProto(time.Date(2019, 01, 16, 12, 0, 0, 0, time.UTC))
	time4pm, _ := types.TimestampProto(time.Date(2019, 01, 16, 16, 0, 0, 0, time.UTC))
	time8pm, _ := types.TimestampProto(time.Date(2019, 01, 16, 20, 0, 0, 0, time.UTC))

	// Store results for standardA from 8am. These should then be returned as the most recent for cluster1, standardA.
	results1 := &storage.ComplianceRunResults{
		RunMetadata: &storage.ComplianceRunMetadata{
			RunId:           "run1",
			ClusterId:       "cluster1",
			StandardId:      "standardA",
			FinishTimestamp: time8am,
			Success:         true,
		},
	}

	err := s.store.StoreRunResults(results1)
	s.Require().NoError(err)

	storedResults, err := s.store.GetLatestRunResults("cluster1", "standardA", 0)
	s.Require().NoError(err)
	s.Equal(ResultsWithStatus{LastSuccessfulResults: results1}, storedResults)

	// Store results for standardB at 12pm. For cluster1, standardA the previous results should still be returned as the
	// most recent.
	results2 := &storage.ComplianceRunResults{
		RunMetadata: &storage.ComplianceRunMetadata{
			RunId:           "run2",
			ClusterId:       "cluster1",
			StandardId:      "standardB",
			FinishTimestamp: time12pm,
			Success:         true,
		},
	}

	err = s.store.StoreRunResults(results2)
	s.Require().NoError(err)

	storedResults, err = s.store.GetLatestRunResults("cluster1", "standardA", 0)
	s.Require().NoError(err)
	s.Equal(ResultsWithStatus{LastSuccessfulResults: results1}, storedResults)

	storedResults, err = s.store.GetLatestRunResults("cluster1", "standardB", 0)
	s.Require().NoError(err)
	s.Equal(ResultsWithStatus{LastSuccessfulResults: results2}, storedResults)

	// Store results for standardA from 8pm. These should now be the most recent results for cluster1, standardA.
	results3 := &storage.ComplianceRunResults{
		RunMetadata: &storage.ComplianceRunMetadata{
			RunId:           "run3",
			ClusterId:       "cluster1",
			StandardId:      "standardA",
			FinishTimestamp: time8pm,
			Success:         true,
		},
	}

	err = s.store.StoreRunResults(results3)
	s.Require().NoError(err)

	storedResults, err = s.store.GetLatestRunResults("cluster1", "standardA", 0)
	s.Require().NoError(err)
	s.Equal(ResultsWithStatus{LastSuccessfulResults: results3}, storedResults)

	// Store results for standardA from 4pm. The previous results from 8pm should still be the most recent for
	// cluster1, standardA.
	results4 := &storage.ComplianceRunResults{
		RunMetadata: &storage.ComplianceRunMetadata{
			RunId:           "run4",
			ClusterId:       "cluster1",
			StandardId:      "standardA",
			FinishTimestamp: time4pm,
			Success:         true,
		},
	}

	err = s.store.StoreRunResults(results4)
	s.Require().NoError(err)

	storedResults, err = s.store.GetLatestRunResults("cluster1", "standardA", 0)
	s.Require().NoError(err)
	s.Equal(ResultsWithStatus{LastSuccessfulResults: results3}, storedResults)
}

func (s *boltStoreTestSuite) TestStoreFailures() {
	time8am, _ := types.TimestampProto(time.Date(2019, 01, 16, 8, 0, 0, 0, time.UTC))
	time12pm, _ := types.TimestampProto(time.Date(2019, 01, 16, 12, 0, 0, 0, time.UTC))
	time4pm, _ := types.TimestampProto(time.Date(2019, 01, 16, 16, 0, 0, 0, time.UTC))
	time8pm, _ := types.TimestampProto(time.Date(2019, 01, 16, 20, 0, 0, 0, time.UTC))
	time10pm, _ := types.TimestampProto(time.Date(2019, 01, 16, 22, 0, 0, 0, time.UTC))

	// Store results for standardA from 8am. These should then be returned as the most recent for cluster1, standardA.
	run1MD := &storage.ComplianceRunMetadata{
		RunId:           "run1",
		ClusterId:       "cluster1",
		StandardId:      "standardA",
		Success:         false,
		ErrorMessage:    "some failure",
		FinishTimestamp: time8am,
	}

	err := s.store.StoreFailure(run1MD)
	s.Require().NoError(err)

	storedResults, err := s.store.GetLatestRunResults("cluster1", "standardA", 0)
	s.Require().NoError(err)
	s.Equal(ResultsWithStatus{FailedRuns: []*storage.ComplianceRunMetadata{run1MD}}, storedResults)

	run2Results := &storage.ComplianceRunResults{
		RunMetadata: &storage.ComplianceRunMetadata{
			RunId:           "run2",
			ClusterId:       "cluster1",
			StandardId:      "standardA",
			FinishTimestamp: time12pm,
			Success:         true,
		},
	}

	err = s.store.StoreRunResults(run2Results)
	s.Require().NoError(err)

	storedResults, err = s.store.GetLatestRunResults("cluster1", "standardA", 0)
	s.Require().NoError(err)
	s.Equal(ResultsWithStatus{LastSuccessfulResults: run2Results}, storedResults)

	run3MD := &storage.ComplianceRunMetadata{
		RunId:           "run3",
		ClusterId:       "cluster1",
		StandardId:      "standardA",
		Success:         false,
		ErrorMessage:    "another failure",
		FinishTimestamp: time4pm,
	}

	err = s.store.StoreFailure(run3MD)
	s.Require().NoError(err)

	storedResults, err = s.store.GetLatestRunResults("cluster1", "standardA", 0)
	s.Require().NoError(err)
	s.Equal(ResultsWithStatus{LastSuccessfulResults: run2Results, FailedRuns: []*storage.ComplianceRunMetadata{run3MD}}, storedResults)

	run4MD := &storage.ComplianceRunMetadata{
		RunId:           "run4",
		ClusterId:       "cluster1",
		StandardId:      "standardA",
		Success:         false,
		ErrorMessage:    "yet another failure",
		FinishTimestamp: time8pm,
	}

	err = s.store.StoreFailure(run4MD)
	s.Require().NoError(err)

	storedResults, err = s.store.GetLatestRunResults("cluster1", "standardA", 0)
	s.Require().NoError(err)
	s.Equal(ResultsWithStatus{LastSuccessfulResults: run2Results, FailedRuns: []*storage.ComplianceRunMetadata{run4MD, run3MD}}, storedResults)

	run5Results := &storage.ComplianceRunResults{
		RunMetadata: &storage.ComplianceRunMetadata{
			RunId:           "run5",
			ClusterId:       "cluster1",
			StandardId:      "standardA",
			FinishTimestamp: time10pm,
			Success:         true,
		},
	}

	err = s.store.StoreRunResults(run5Results)
	s.Require().NoError(err)

	storedResults, err = s.store.GetLatestRunResults("cluster1", "standardA", 0)
	s.Require().NoError(err)
	s.Equal(ResultsWithStatus{LastSuccessfulResults: run5Results}, storedResults)
}
