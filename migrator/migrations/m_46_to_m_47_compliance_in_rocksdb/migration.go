package m45tom46

import (
	"errors"
	"fmt"

	protoTypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
	"github.com/tecbot/gorocksdb"
	"go.etcd.io/bbolt"
)

const (
	nanosecondsPerMicrosecond = 1000
)

var (
	migration = types.Migration{
		StartingSeqNum: 46,
		VersionAfter:   storage.Version{SeqNum: 47},
		Run:            migrateComplianceRuns,
	}

	defaultWriteOptions = generic.DefaultWriteOptions()
)

func init() {
	migrations.MustRegisterMigration(migration)
}

var (
	resultsBucketName = []byte("compliance-run-results")

	resultsKey  = []byte("results")
	metadataKey = []byte("metadata")
	stringsKey  = []byte("strings")
)

func migrateComplianceRuns(databases *types.Databases) error {
	resultsBucket := bolthelpers.TopLevelRef(databases.BoltDB, resultsBucketName)
	return resultsBucket.View(func(b *bbolt.Bucket) error {
		clusterCursor := b.Cursor()
		for clusterKey, _ := clusterCursor.First(); clusterKey != nil; clusterKey, _ = clusterCursor.Next() {
			clusterBucket := b.Bucket(clusterKey)
			if clusterBucket == nil {
				continue
			}

			standardCursor := clusterBucket.Cursor()
			for standardKey, _ := standardCursor.First(); standardKey != nil; standardKey, _ = standardCursor.Next() {
				standardBucket := clusterBucket.Bucket(standardKey)
				if standardBucket == nil {
					continue
				}

				cursor := standardBucket.Cursor()

				for latestRunBucketKey, _ := cursor.Last(); latestRunBucketKey != nil; latestRunBucketKey, _ = cursor.Prev() {
					runBucket := standardBucket.Bucket(latestRunBucketKey)
					if runBucket == nil {
						continue
					}

					if err := migrateRunResultsToRocksDB(runBucket, databases.RocksDB); err != nil {
						log.WriteToStderrf("Error migrating a run for cluster %s, standard %s: %s", string(clusterKey), string(standardKey), err.Error())
					}
				}
			}
		}
		return nil
	})
}

func migrateRunResultsToRocksDB(runBucket *bbolt.Bucket, rocksDB *gorocksdb.DB) error {
	metadata, err := readMetadata(runBucket)
	if err != nil {
		return err
	}
	if !metadata.GetSuccess() {
		return storeRun(metadata, nil, nil, rocksDB)
	}
	results, err := readResults(runBucket)
	if err != nil {
		return err
	}
	results.RunMetadata = metadata
	strings, err := readStrings(runBucket)
	if err != nil {
		return err
	}
	return storeRun(metadata, results, strings, rocksDB)
}

func readMetadata(resultsBucket *bbolt.Bucket) (*storage.ComplianceRunMetadata, error) {
	metadataBytes := resultsBucket.Get(metadataKey)
	if metadataBytes == nil {
		return nil, errors.New("bucket does not have a metadata entry")
	}
	var metadata storage.ComplianceRunMetadata
	if err := metadata.Unmarshal(metadataBytes); err != nil {
		return nil, fmt.Errorf("unmarshalling metadata: %s", err.Error())
	}
	return &metadata, nil
}

func readResults(resultsBucket *bbolt.Bucket) (*storage.ComplianceRunResults, error) {
	resultsBytes := resultsBucket.Get(resultsKey)
	if resultsBytes == nil {
		return nil, errors.New("metadata indicated success, but no results data was found")
	}
	var results storage.ComplianceRunResults
	if err := results.Unmarshal(resultsBytes); err != nil {
		return nil, fmt.Errorf("unmarshalling results: %s", err.Error())
	}
	return &results, nil
}

func readStrings(resultsBucket *bbolt.Bucket) (*storage.ComplianceStrings, error) {
	stringsBytes := resultsBucket.Get(stringsKey)
	if stringsBytes == nil {
		return nil, errors.New("no strings were found")
	}
	var strings storage.ComplianceStrings
	if err := strings.Unmarshal(stringsBytes); err != nil {
		return nil, fmt.Errorf("unmarshalling results: %s", err.Error())

	}
	return &strings, nil
}

func storeRun(metadata *storage.ComplianceRunMetadata, results *storage.ComplianceRunResults, strings *storage.ComplianceStrings, rocksDB *gorocksdb.DB) error {
	mdKey, resKey, strKey, err := makeKeys(metadata.GetClusterId(), metadata.GetStandardId(), metadata.GetRunId(), metadata.GetFinishTimestamp())
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

	stringsBytes, err := strings.Marshal()
	if err != nil {
		return err
	}

	batch.Put(resKey, resultBytes)
	batch.Put(strKey, stringsBytes)

	return rocksDB.Write(defaultWriteOptions, batch)
}

func makeKeys(clusterID, standardID, runID string, finishTimeProto *protoTypes.Timestamp) ([]byte, []byte, []byte, error) {
	finishTime, err := protoTypes.TimestampFromProto(finishTimeProto)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("run has an invalid finish timestamp: %s", err.Error())
	}
	microTS := finishTime.UnixNano() / nanosecondsPerMicrosecond
	tsBytes := []byte(fmt.Sprintf("%016X", microTS))
	// Invert the bits of each byte of the timestamp to reverse the lexicographic sort order
	for i, tsByte := range tsBytes {
		tsBytes[i] = -tsByte
	}

	mdKey := makeKey(string(metadataKey), clusterID, standardID, runID, tsBytes)
	runKey := makeKey(string(resultsKey), clusterID, standardID, runID, tsBytes)
	strKey := makeKey(string(stringsKey), clusterID, standardID, runID, tsBytes)
	return mdKey, runKey, strKey, nil
}

func makeKey(keyType, clusterID, standardID, runID string, tsBytes []byte) []byte {
	partialKey := []byte(fmt.Sprintf("%s:%s:%s:", keyType, clusterID, standardID))
	runIDAndSeparator := []byte(fmt.Sprintf(":%s", runID))
	partialKey = append(partialKey, tsBytes...)
	partialKey = append(partialKey, runIDAndSeparator...)
	return rocksdbmigration.GetPrefixedKey(resultsBucketName, partialKey)
}
