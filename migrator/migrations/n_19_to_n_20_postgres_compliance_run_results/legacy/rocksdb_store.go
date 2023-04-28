// This file was originally generated with
// //go:generate cp ../../../../central/compliance/datastore/internal/store/rocksdb/rocksdb_store.go .

package legacy

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/rocksdb"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/tecbot/gorocksdb"
)

var (
	readOptions  = generic.DefaultReadOptions()
	writeOptions = generic.DefaultWriteOptions()

	resultsBucketName = []byte("compliance-run-results")

	resultsKey = dbhelper.GetBucketKey(resultsBucketName, []byte("results"))
)

// New returns a compliance metadata store that is backed by RocksDB.
func New(db *rocksdb.RocksDB) (Store, error) {
	return &rocksdbStore{
		db: db,
	}, nil
}

type rocksdbStore struct {
	db *rocksdb.RocksDB
}

type keyMaker struct {
	partialResultsPrefix []byte
}

func (k *keyMaker) getKeysForMetadata(metadata *storage.ComplianceRunMetadata) ([]byte, error) {
	runID := metadata.GetRunId()
	if runID == "" {
		return nil, errors.New("run has an empty ID")
	}
	tsBytes := []byte(fmt.Sprintf("%016X", timestamp.FromGoTime(time.Now())))
	// Invert the bits of each byte of the timestamp in order to have the most recent timestamp first
	for i, tsByte := range tsBytes {
		tsBytes[i] = -tsByte
	}
	separatorAndRunID := []byte(fmt.Sprintf(":%s", runID))
	tsAndRunIDPrefix := append(tsBytes, separatorAndRunID...)

	key := append([]byte{}, k.partialResultsPrefix...)
	key = append(key, tsAndRunIDPrefix...)

	return key, nil
}

func getKeyMaker(clusterID, standardID string) *keyMaker {
	resultsPrefix := getClusterStandardPrefixes(clusterID, standardID)

	return &keyMaker{
		partialResultsPrefix: resultsPrefix,
	}
}

func getClusterStandardPrefixes(clusterID, standardID string) []byte {
	// trailing colon is intentional, this prefix will always be followed by a timestamp and a run ID
	partialPrefix := fmt.Sprintf("%s:%s:", clusterID, standardID)
	resultsPrefix := getPrefix(string(resultsKey), partialPrefix)
	return resultsPrefix
}

func getPrefix(leftPrefix, rightPrefix string) []byte {
	return []byte(leftPrefix + ":" + rightPrefix)
}

func unmarshalResults(iterator *gorocksdb.Iterator) (*storage.ComplianceRunResults, error) {
	bytes := iterator.Value().Data()
	if len(bytes) == 0 {
		return nil, errors.New("results data empty")
	}
	var results storage.ComplianceRunResults
	if err := results.Unmarshal(bytes); err != nil {
		return nil, errors.Wrap(err, "unmarshalling results")
	}
	return &results, nil
}

func (r *rocksdbStore) Walk(_ context.Context, fn func(obj *storage.ComplianceRunResults) error) error {
	iterator := r.db.NewIterator(readOptions)
	defer iterator.Close()
	// Runs are sorted by time so we must iterate over each key to see if it has the correct run ID.
	for iterator.Seek(resultsKey); iterator.ValidForPrefix(resultsKey); iterator.Next() {
		result, err := unmarshalResults(iterator)
		if err != nil {
			return err
		}
		if err = fn(result); err != nil {
			return err
		}
	}
	return nil
}

func (r *rocksdbStore) UpsertMany(_ context.Context, objs []*storage.ComplianceRunResults) error {
	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()

	for _, obj := range objs {
		metadata := obj.GetRunMetadata()
		clusterID := metadata.GetClusterId()
		standardID := metadata.GetStandardId()

		serializedResults, err := obj.Marshal()
		if err != nil {
			return errors.Wrap(err, "serializing results")
		}

		maker := getKeyMaker(clusterID, standardID)
		rKey, err := maker.getKeysForMetadata(metadata)
		if err != nil {
			return err
		}

		// Store results under the key "compliance-run-results\x00results:CLUSTER:STANDARD:REVERSE_TIMESTAMP:RUN_ID
		batch.Put(rKey, serializedResults)
	}
	return r.db.Write(writeOptions, batch)
}
