package m48tom49

import (
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	dbTypes "github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/rocksdb"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
)

const (
	nanosecondsPerMicrosecond = 1000
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(complianceDomainExternalizationMigrationTestSuite))
}

type complianceDomainExternalizationMigrationTestSuite struct {
	suite.Suite

	db        *rocksdb.RocksDB
	databases *dbTypes.Databases
}

func (s *complianceDomainExternalizationMigrationTestSuite) SetupTest() {
	rocksDB, err := rocksdb.NewTemp(s.T().Name())
	s.NoError(err)

	s.db = rocksDB
	s.databases = &dbTypes.Databases{RocksDB: rocksDB.DB}
}

func (s *complianceDomainExternalizationMigrationTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.db)
}

func getMockComplianceRunResults() (*storage.ComplianceRunMetadata, *storage.ComplianceRunResults) {
	res := &storage.ComplianceRunResults{
		Domain: &storage.ComplianceDomain{
			Cluster: &storage.Cluster{
				Id:   "testcluster",
				Name: "Joseph Rules",
			},
			Nodes: map[string]*storage.Node{
				"node": {
					Id:   "testnode",
					Name: "Joseph is the best",
				},
			},
			Deployments: map[string]*storage.Deployment{
				"deployment": {
					Id:   "testdeployment",
					Name: "Joseph is super cool",
				},
			},
		},
		RunMetadata: &storage.ComplianceRunMetadata{
			RunId:           "abcd",
			StandardId:      "efgh",
			ClusterId:       "jklm",
			StartTimestamp:  types.TimestampNow(),
			FinishTimestamp: types.TimestampNow(),
			Success:         true,
		},
		ClusterResults: &storage.ComplianceRunResults_EntityResults{
			ControlResults: map[string]*storage.ComplianceResultValue{
				"some result": {
					Evidence: []*storage.ComplianceResultValue_Evidence{
						{
							MessageId: 1,
						},
					},
				},
			},
		},
	}

	return res.RunMetadata, res
}

func (s *complianceDomainExternalizationMigrationTestSuite) TestComplianceDomainMigration() {
	md, res := getMockComplianceRunResults()
	mdKey, resKey, err := makeKeys(md.ClusterId, md.StandardId, md.RunId, md.FinishTimestamp)
	s.Require().NoError(err)
	s.Require().NoError(storeRun(md, res, s.databases.RocksDB))

	mdSlice, err := s.databases.RocksDB.Get(generic.DefaultReadOptions(), mdKey)
	s.Require().NoError(err)
	s.Require().True(mdSlice.Exists())
	var dbMd storage.ComplianceRunMetadata
	s.Require().NoError(proto.Unmarshal(mdSlice.Data(), &dbMd))
	s.Equal(md, &dbMd)

	resSlice, err := s.databases.RocksDB.Get(generic.DefaultReadOptions(), resKey)
	s.Require().NoError(err)
	s.Require().True(resSlice.Exists())
	var dbRes storage.ComplianceRunResults
	s.Require().NoError(proto.Unmarshal(resSlice.Data(), &dbRes))
	s.Equal(res, &dbRes)

	domKey := migrateDomain(s.databases.RocksDB, mdKey, md)
	s.Require().NotNil(domKey)

	resSlice, err = s.databases.RocksDB.Get(generic.DefaultReadOptions(), resKey)
	s.Require().NoError(err)
	s.Require().True(resSlice.Exists())
	s.Require().NoError(proto.Unmarshal(resSlice.Data(), &dbRes))
	s.Nil(dbRes.GetDomain())

	domSlice, err := s.databases.RocksDB.Get(generic.DefaultReadOptions(), domKey)
	s.Require().NoError(err)
	s.Require().True(resSlice.Exists())
	var dbDomain storage.ComplianceDomain
	s.Require().NoError(proto.Unmarshal(domSlice.Data(), &dbDomain))
	s.NotEmpty(dbDomain.GetId())
	// Domain ID should be a random UUID, set it in the expected value because it doesn't matter so long as it's non-empty
	res.Domain.Id = dbDomain.GetId()
	s.Equal(res.GetDomain(), &dbDomain)
}

func storeRun(metadata *storage.ComplianceRunMetadata, results *storage.ComplianceRunResults, rocksDB *gorocksdb.DB) error {
	mdKey, resKey, err := makeKeys(metadata.GetClusterId(), metadata.GetStandardId(), metadata.GetRunId(), metadata.GetFinishTimestamp())
	if err != nil {
		return err
	}

	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()

	mdBytes, err := metadata.Marshal()
	if err != nil {
		return err
	}
	batch.Put(mdKey, mdBytes)
	if !metadata.GetSuccess() {
		return rocksDB.Write(defaultWriteOptions, batch)
	}

	resultBytes, err := results.Marshal()
	if err != nil {
		return err
	}

	batch.Put(resKey, resultBytes)

	return rocksDB.Write(defaultWriteOptions, batch)
}

func makeKeys(clusterID, standardID, runID string, finishTimeProto *types.Timestamp) ([]byte, []byte, error) {
	finishTime, err := types.TimestampFromProto(finishTimeProto)
	if err != nil {
		return nil, nil, fmt.Errorf("run has an invalid finish timestamp: %s", err.Error())
	}
	microTS := finishTime.UnixNano() / nanosecondsPerMicrosecond
	tsBytes := []byte(fmt.Sprintf("%016X", microTS))
	// Invert the bits of each byte of the timestamp to reverse the lexicographic sort order
	for i, tsByte := range tsBytes {
		tsBytes[i] = -tsByte
	}

	mdKey := makeKey(string(metadataKey), clusterID, standardID, runID, tsBytes)
	runKey := makeKey(string(resultsKey), clusterID, standardID, runID, tsBytes)
	return mdKey, runKey, nil
}

func makeKey(keyType, clusterID, standardID, runID string, tsBytes []byte) []byte {
	partialKey := []byte(fmt.Sprintf("%s:%s:%s:", keyType, clusterID, standardID))
	runIDAndSeparator := []byte(fmt.Sprintf(":%s", runID))
	partialKey = append(partialKey, tsBytes...)
	return append(partialKey, runIDAndSeparator...)
}
