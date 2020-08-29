package rocksdb

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/datastore/internal/store"
	dsTypes "github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/rocksdb"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestRocksDBStore(t *testing.T) {
	suite.Run(t, new(RocksDBStoreTestSuite))
}

type RocksDBStoreTestSuite struct {
	suite.Suite

	db    *rocksdb.RocksDB
	store store.Store
}

func (s *RocksDBStoreTestSuite) SetupTest() {
	db, err := rocksdb.NewTemp("compliance_db_test")
	s.Require().NoError(err)
	s.db = db

	s.store = NewRocksdbStore(db)
}

func (s *RocksDBStoreTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.db)
}

func readFromDB(db *rocksdb.RocksDB, key []byte, protoObject proto.Message) error {
	slice, err := db.Get(generic.DefaultReadOptions(), key)
	if err != nil {
		return err
	}
	defer slice.Free()
	sliceBytes := slice.Data()
	err = proto.Unmarshal(sliceBytes, protoObject)
	return err
}

func (s *RocksDBStoreTestSuite) validateLatestResults(results *storage.ComplianceRunResults, flags dsTypes.GetFlags, failedRuns ...*storage.ComplianceRunMetadata) {
	dbResult, err := s.store.GetLatestRunResults(results.RunMetadata.ClusterId, results.RunMetadata.StandardId, flags)
	s.Require().NoError(err)
	s.Equal(results, dbResult.LastSuccessfulResults)
	s.Len(dbResult.FailedRuns, len(failedRuns))
	s.ElementsMatch(failedRuns, dbResult.FailedRuns)
}

func (s *RocksDBStoreTestSuite) TestStoreComplianceResult() {
	result := store.GetMockResult()
	err := s.store.StoreRunResults(result)
	s.Require().NoError(err)
	keyMaker := getKeyMaker(result.RunMetadata.ClusterId, result.RunMetadata.StandardId)

	metaKey, resKey, strKey, err := keyMaker.getKeysForMetadata(result.RunMetadata)
	s.Require().NoError(err)

	var dbResult storage.ComplianceRunResults
	err = readFromDB(s.db, resKey, &dbResult)
	s.Require().NoError(err)
	s.Equal(result, &dbResult)

	var dbMetadata storage.ComplianceRunMetadata
	err = readFromDB(s.db, metaKey, &dbMetadata)
	s.Require().NoError(err)
	s.Equal(result.RunMetadata, &dbMetadata)

	var dbStrings storage.ComplianceStrings
	err = readFromDB(s.db, strKey, &dbStrings)
	s.Require().NoError(err)
	s.NotNil(dbStrings)
}

func (s *RocksDBStoreTestSuite) TestStoreFailedComplianceResult() {
	result := store.GetMockResult()
	result.RunMetadata.Success = false
	s.Error(s.store.StoreRunResults(result))

	result = store.GetMockResult()
	result.RunMetadata = nil
	s.Error(s.store.StoreRunResults(result))
}

func (s *RocksDBStoreTestSuite) TestGetLatest() {
	newerResult := store.GetMockResult()
	olderResult := store.GetMockResult()
	olderResult.RunMetadata.FinishTimestamp.Seconds = olderResult.RunMetadata.FinishTimestamp.Seconds - 600
	olderResult.RunMetadata.RunId = "Test run ID 2"

	err := s.store.StoreRunResults(olderResult)
	s.Require().NoError(err)
	s.validateLatestResults(olderResult, 0)

	err = s.store.StoreRunResults(newerResult)
	s.Require().NoError(err)
	s.validateLatestResults(newerResult, 0)
}

func (s *RocksDBStoreTestSuite) TestStoreFailure() {
	oldResult := store.GetMockResult()
	failedResult := oldResult.RunMetadata.Clone()
	failedResult.Success = false
	failedResult.FinishTimestamp.Seconds = failedResult.FinishTimestamp.Seconds + 600
	failedResult.ErrorMessage = "Test error message"

	err := s.store.StoreRunResults(oldResult)
	s.Require().NoError(err)
	s.validateLatestResults(oldResult, 0)

	err = s.store.StoreFailure(failedResult)
	s.Require().NoError(err)
	s.validateLatestResults(oldResult, 0, failedResult)
}

func (s *RocksDBStoreTestSuite) TestGetSpecificRun() {
	justRight := store.GetMockResult()
	tooEarly := justRight.Clone()
	tooEarly.RunMetadata.RunId = "Too early"
	tooEarly.RunMetadata.FinishTimestamp.Seconds = tooEarly.RunMetadata.FinishTimestamp.Seconds - 600
	tooLate := justRight.Clone()
	tooLate.RunMetadata.RunId = "Too late"
	tooLate.RunMetadata.FinishTimestamp.Seconds = tooLate.RunMetadata.FinishTimestamp.Seconds + 600

	err := s.store.StoreRunResults(tooEarly)
	s.Require().NoError(err)

	err = s.store.StoreRunResults(justRight)
	s.Require().NoError(err)

	err = s.store.StoreRunResults(tooLate)
	s.Require().NoError(err)

	dbResults, err := s.store.GetSpecificRunResults(justRight.RunMetadata.ClusterId, justRight.RunMetadata.StandardId, justRight.RunMetadata.RunId, 0)
	s.Require().NoError(err)
	s.Equal(justRight, dbResults.LastSuccessfulResults)
	s.Empty(dbResults.FailedRuns)
}

func (s *RocksDBStoreTestSuite) TestGetLatestRunResultsByClusterAndStandard() {
	filterIn := store.GetMockResult()
	s.Require().NoError(s.store.StoreRunResults(filterIn))

	filterInOld := store.GetMockResult()
	filterInOld.RunMetadata.FinishTimestamp.Seconds = filterInOld.RunMetadata.FinishTimestamp.Seconds - 600
	s.Require().NoError(s.store.StoreRunResults(filterInOld))

	filterOutCluster := store.GetMockResult()
	filterOutCluster.RunMetadata.ClusterId = "Not this cluster!"
	s.Require().NoError(s.store.StoreRunResults(filterOutCluster))

	filterOutStandard := store.GetMockResult()
	filterOutStandard.RunMetadata.StandardId = "Not this standard!"
	s.Require().NoError(s.store.StoreRunResults(filterOutStandard))

	filterOutClusterAndStandard := store.GetMockResult()
	filterOutClusterAndStandard.RunMetadata.ClusterId = "Another bad cluster"
	filterOutClusterAndStandard.RunMetadata.StandardId = "Another bad standard"
	s.Require().NoError(s.store.StoreRunResults(filterOutClusterAndStandard))

	clusterIDs := []string{filterIn.RunMetadata.ClusterId}
	standardIDs := []string{filterIn.RunMetadata.StandardId}

	resultMap, err := s.store.GetLatestRunResultsByClusterAndStandard(clusterIDs, standardIDs, 0)
	s.Require().NoError(err)
	expectedPair := compliance.ClusterStandardPair{
		ClusterID:  filterIn.RunMetadata.ClusterId,
		StandardID: filterIn.RunMetadata.StandardId,
	}
	s.Len(resultMap, 1)
	s.Require().Contains(resultMap, expectedPair)
	result := resultMap[expectedPair]
	s.Equal(filterIn, result.LastSuccessfulResults)
	s.Empty(result.FailedRuns)
}

func (s *RocksDBStoreTestSuite) TestGetLatestRunMetadataBatch() {
	standardOne := store.GetMockResult()
	s.Require().NoError(s.store.StoreRunResults(standardOne))

	standardTwo := store.GetMockResult()
	standardTwo.RunMetadata.StandardId = "Bla bla bla"
	s.Require().NoError(s.store.StoreRunResults(standardTwo))

	standardFilterOut := store.GetMockResult()
	standardFilterOut.RunMetadata.StandardId = "Joseph Rules"
	s.Require().NoError(s.store.StoreRunResults(standardFilterOut))

	clusterFilterOut := store.GetMockResult()
	clusterFilterOut.RunMetadata.ClusterId = "Agdjklgrkjl"
	s.Require().NoError(s.store.StoreRunResults(clusterFilterOut))

	resultsMap, err := s.store.GetLatestRunMetadataBatch(standardOne.RunMetadata.ClusterId, []string{standardOne.RunMetadata.StandardId, standardTwo.RunMetadata.StandardId})
	s.Require().NoError(err)
	s.Len(resultsMap, 2)

	expectedPairOne := compliance.ClusterStandardPair{
		ClusterID:  standardOne.RunMetadata.ClusterId,
		StandardID: standardOne.RunMetadata.StandardId,
	}
	s.Require().Contains(resultsMap, expectedPairOne)
	resultUnderTest := resultsMap[expectedPairOne]
	s.Equal(standardOne.RunMetadata, resultUnderTest.LastSuccessfulRunMetadata)
	s.Empty(resultUnderTest.FailedRunsMetadata)

	expectedPairTwo := compliance.ClusterStandardPair{
		ClusterID:  standardTwo.RunMetadata.ClusterId,
		StandardID: standardTwo.RunMetadata.StandardId,
	}
	s.Require().Contains(resultsMap, expectedPairTwo)
	resultUnderTest = resultsMap[expectedPairTwo]
	s.Equal(standardTwo.RunMetadata, resultUnderTest.LastSuccessfulRunMetadata)
	s.Empty(resultUnderTest.FailedRunsMetadata)
}

func (s *RocksDBStoreTestSuite) TestGetOnEmpty() {
	results, err := s.store.GetLatestRunResults("foo", "bar", 0)
	s.Zero(results)
	s.Error(err)
}

func (s *RocksDBStoreTestSuite) TestBatchGetOnEmpty() {
	results, err := s.store.GetLatestRunResultsBatch([]string{"cluster1"}, []string{"standard1, standard2"}, 0)
	s.NoError(err)
	s.Len(results, 0)
}

func (s *RocksDBStoreTestSuite) TestGetLatestRunResultsByClusterAndStandardEmpty() {
	clusterIDs := []string{"some ID"}
	standardIDs := []string{"some ID"}
	results, err := s.store.GetLatestRunResultsByClusterAndStandard(clusterIDs, standardIDs, 0)
	s.NoError(err)
	s.Len(results, 0)
}

func (s *RocksDBStoreTestSuite) TestStoreAndRetrieveExternalizedStrings() {
	resultKey := "testResult"
	message := "This string should get externalized"
	results := store.GetMockResult()
	results.ClusterResults = &storage.ComplianceRunResults_EntityResults{
		ControlResults: map[string]*storage.ComplianceResultValue{
			resultKey: {
				Evidence: []*storage.ComplianceResultValue_Evidence{
					{
						Message: message,
					},
				},
			},
		},
	}

	expectedResultsWithoutExternalizedStrings := results.Clone()
	expectedResultsWithoutExternalizedStrings.ClusterResults = &storage.ComplianceRunResults_EntityResults{
		ControlResults: map[string]*storage.ComplianceResultValue{
			resultKey: {
				Evidence: []*storage.ComplianceResultValue_Evidence{
					{
						MessageId: 1,
					},
				},
			},
		},
	}

	expectedResultsWithExternalizedStrings := results.Clone()
	expectedResultsWithExternalizedStrings.ClusterResults = &storage.ComplianceRunResults_EntityResults{
		ControlResults: map[string]*storage.ComplianceResultValue{
			resultKey: {
				Evidence: []*storage.ComplianceResultValue_Evidence{
					{
						Message: message,
					},
				},
			},
		},
	}

	s.Require().NoError(s.store.StoreRunResults(results))
	s.validateLatestResults(expectedResultsWithoutExternalizedStrings, 0)
	s.validateLatestResults(expectedResultsWithExternalizedStrings, dsTypes.WithMessageStrings)
}
