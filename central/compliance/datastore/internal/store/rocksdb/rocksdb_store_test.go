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
	domainCache.RemoveAll()
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
	result, _ := store.GetMockResult()
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
	s.NotEmpty(dbStrings.GetStrings())
}

func (s *RocksDBStoreTestSuite) TestStoreDomain() {
	result, domain := store.GetMockResult()

	s.Require().NoError(s.store.StoreRunResults(result))
	dbResult, err := s.store.GetSpecificRunResults(result.GetRunMetadata().GetClusterId(), result.GetRunMetadata().GetStandardId(), result.GetRunMetadata().GetRunId(), dsTypes.WithMessageStrings)
	s.Require().NoError(err)
	s.Nil(dbResult.LastSuccessfulResults.Domain)

	s.Require().NoError(s.store.StoreComplianceDomain(domain))
	dbResult, err = s.store.GetSpecificRunResults(result.GetRunMetadata().GetClusterId(), result.GetRunMetadata().GetStandardId(), result.GetRunMetadata().GetRunId(), dsTypes.WithMessageStrings)
	s.Require().NoError(err)
	s.Equal(domain, dbResult.LastSuccessfulResults.Domain)
}

func (s *RocksDBStoreTestSuite) TestStoreFailedComplianceResult() {
	result, _ := store.GetMockResult()
	result.RunMetadata.Success = false
	s.Error(s.store.StoreRunResults(result))

	result, _ = store.GetMockResult()
	result.RunMetadata = nil
	s.Error(s.store.StoreRunResults(result))
}

func (s *RocksDBStoreTestSuite) TestGetLatest() {
	newerResult, _ := store.GetMockResult()
	olderResult, _ := store.GetMockResult()
	olderResult.RunMetadata.FinishTimestamp.Seconds = olderResult.RunMetadata.FinishTimestamp.Seconds - 600
	olderResult.RunMetadata.RunId = "Test run ID 2"
	expectedNewerResult := newerResult.Clone()
	expectedOlderResult := olderResult.Clone()

	err := s.store.StoreRunResults(olderResult)
	s.Require().NoError(err)
	s.validateLatestResults(expectedOlderResult, dsTypes.WithMessageStrings)

	err = s.store.StoreRunResults(newerResult)
	s.Require().NoError(err)
	s.validateLatestResults(expectedNewerResult, dsTypes.WithMessageStrings)
}

func (s *RocksDBStoreTestSuite) TestStoreFailure() {
	oldResult, _ := store.GetMockResult()
	failedResult := oldResult.RunMetadata.Clone()
	failedResult.Success = false
	failedResult.FinishTimestamp.Seconds = failedResult.FinishTimestamp.Seconds + 600
	failedResult.ErrorMessage = "Test error message"

	err := s.store.StoreRunResults(oldResult.Clone())
	s.Require().NoError(err)
	s.validateLatestResults(oldResult, dsTypes.WithMessageStrings)

	err = s.store.StoreFailure(failedResult)
	s.Require().NoError(err)
	s.validateLatestResults(oldResult, dsTypes.WithMessageStrings, failedResult)
}

func (s *RocksDBStoreTestSuite) TestGetSpecificRun() {
	justRight, _ := store.GetMockResult()
	tooEarly := justRight.Clone()
	tooEarly.RunMetadata.RunId = "Too early"
	tooEarly.RunMetadata.FinishTimestamp.Seconds = tooEarly.RunMetadata.FinishTimestamp.Seconds - 600
	tooLate := justRight.Clone()
	tooLate.RunMetadata.RunId = "Too late"
	tooLate.RunMetadata.FinishTimestamp.Seconds = tooLate.RunMetadata.FinishTimestamp.Seconds + 600

	err := s.store.StoreRunResults(tooEarly)
	s.Require().NoError(err)

	err = s.store.StoreRunResults(justRight.Clone())
	s.Require().NoError(err)

	err = s.store.StoreRunResults(tooLate)
	s.Require().NoError(err)

	dbResults, err := s.store.GetSpecificRunResults(justRight.RunMetadata.ClusterId, justRight.RunMetadata.StandardId, justRight.RunMetadata.RunId, dsTypes.WithMessageStrings)
	s.Require().NoError(err)
	s.Equal(justRight, dbResults.LastSuccessfulResults)
	s.Empty(dbResults.FailedRuns)
}

func (s *RocksDBStoreTestSuite) TestGetLatestRunResultsByClusterAndStandard() {
	filterIn, _ := store.GetMockResult()
	s.Require().NoError(s.store.StoreRunResults(filterIn.Clone()))

	filterInOld, _ := store.GetMockResult()
	filterInOld.RunMetadata.FinishTimestamp.Seconds = filterInOld.RunMetadata.FinishTimestamp.Seconds - 600
	s.Require().NoError(s.store.StoreRunResults(filterInOld))

	filterOutCluster, _ := store.GetMockResult()
	filterOutCluster.RunMetadata.ClusterId = "Not this cluster!"
	s.Require().NoError(s.store.StoreRunResults(filterOutCluster))

	filterOutStandard, _ := store.GetMockResult()
	filterOutStandard.RunMetadata.StandardId = "Not this standard!"
	s.Require().NoError(s.store.StoreRunResults(filterOutStandard))

	filterOutClusterAndStandard, _ := store.GetMockResult()
	filterOutClusterAndStandard.RunMetadata.ClusterId = "Another bad cluster"
	filterOutClusterAndStandard.RunMetadata.StandardId = "Another bad standard"
	s.Require().NoError(s.store.StoreRunResults(filterOutClusterAndStandard))

	clusterIDs := []string{filterIn.RunMetadata.ClusterId}
	standardIDs := []string{filterIn.RunMetadata.StandardId}

	resultMap, err := s.store.GetLatestRunResultsByClusterAndStandard(clusterIDs, standardIDs, dsTypes.WithMessageStrings)
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
	standardOne, _ := store.GetMockResult()
	s.Require().NoError(s.store.StoreRunResults(standardOne))

	standardTwo, _ := store.GetMockResult()
	standardTwo.RunMetadata.StandardId = "Bla bla bla"
	s.Require().NoError(s.store.StoreRunResults(standardTwo))

	standardFilterOut, _ := store.GetMockResult()
	standardFilterOut.RunMetadata.StandardId = "Joseph Rules"
	s.Require().NoError(s.store.StoreRunResults(standardFilterOut))

	clusterFilterOut, _ := store.GetMockResult()
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
	results, _ := store.GetMockResult()
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

func (s *RocksDBStoreTestSuite) TestSameDomain() {
	testRunOne, _ := store.GetMockResult()
	testRunTwo, _ := store.GetMockResult()
	testRunTwo.RunMetadata.RunId = "some other run ID"
	testRunTwo.RunMetadata.ClusterId = "some other cluster ID"
	testRunTwo.RunMetadata.StandardId = "Joseph Rules"
	s.Require().NoError(s.store.StoreRunResults(testRunOne.Clone()))
	s.Require().NoError(s.store.StoreRunResults(testRunTwo.Clone()))

	latest, err := s.store.GetLatestRunResultsBatch(
		[]string{
			testRunOne.RunMetadata.ClusterId,
			testRunTwo.RunMetadata.ClusterId,
		},
		[]string{
			testRunOne.RunMetadata.StandardId,
			testRunTwo.RunMetadata.StandardId,
		},
		dsTypes.WithMessageStrings,
	)
	s.Require().NoError(err)

	s.Require().Len(latest, 2)
	lastSuccessful := make([]*storage.ComplianceRunResults, 0, 2)
	for _, latestRun := range latest {
		lastSuccessful = append(lastSuccessful, latestRun.LastSuccessfulResults)
	}
	s.Contains(lastSuccessful, testRunOne)
	s.Contains(lastSuccessful, testRunTwo)
	// The two ComplianceRunResults should have the same Domain
	s.Equal(lastSuccessful[0].Domain, lastSuccessful[1].Domain)
}
