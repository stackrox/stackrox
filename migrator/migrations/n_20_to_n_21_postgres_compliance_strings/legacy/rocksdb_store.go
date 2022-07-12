package legacy

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/rocksdb"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/tecbot/gorocksdb"
)

var (
	readOptions  = generic.DefaultReadOptions()
	writeOptions = generic.DefaultWriteOptions()

	resultsBucketName = []byte("compliance-run-results")

	stringsKey = dbhelper.GetBucketKey(resultsBucketName, []byte("strings"))

	log = logging.LoggerForModule()
)

// New returns a compliance results store that is backed by RocksDB.
func New(db *rocksdb.RocksDB) (Store, error) {
	globaldb.RegisterBucket(resultsBucketName, "ComplianceRunResults")
	return &rocksdbStore{
		db: db,
	}, nil
}

type rocksdbStore struct {
	db *rocksdb.RocksDB
}

func getClusterStandardPrefixes(clusterID, standardID string) []byte {
	// trailing colon is intentional, this prefix will always be followed by a timestamp and a run ID
	partialPrefix := fmt.Sprintf("%s:%s:", clusterID, standardID)
	stringsPrefix := getPrefix(string(stringsKey), partialPrefix)
	return stringsPrefix
}

func getPrefix(leftPrefix, rightPrefix string) []byte {
	return []byte(leftPrefix + ":" + rightPrefix)
}

func unmarshalMessageStrings(iterator *gorocksdb.Iterator) (*storage.ComplianceStrings, error) {
	bytes := iterator.Value().Data()
	if len(bytes) == 0 {
		return nil, errors.New("results data empty")
	}
	var stringsProto storage.ComplianceStrings
	if err := stringsProto.Unmarshal(bytes); err != nil {
		return nil, err
	}
	return &stringsProto, nil
}

func (r *rocksdbStore) Walk(ctx context.Context, fn func(obj *storage.ComplianceStrings) error) error {
	iterator := r.db.NewIterator(readOptions)
	defer iterator.Close()
	// Runs are sorted by time so we must iterate over each key to see if it has the correct run ID.
	for iterator.Seek(stringsKey); iterator.ValidForPrefix(stringsKey); iterator.Next() {
		result, err := unmarshalMessageStrings(iterator)
		if err != nil {
			return err
		}
		if err = fn(result); err != nil {
			return err
		}
	}
	return nil
}

func (r *rocksdbStore) createKay() []byte {
	tsBytes := uuid.NewV4().Bytes()
	// Invert the bits of each byte of the timestamp in order to have the most recent timestamp first
	for i, tsByte := range tsBytes {
		tsBytes[i] = -tsByte
	}
	separatorAndRunID := []byte(fmt.Sprintf(":%s", "runID"))
	tsAndRunIDPrefix := append(tsBytes, separatorAndRunID...)
	stringsPrefix := getClusterStandardPrefixes("cluster", "standard")
	key := append([]byte{}, stringsPrefix...)
	key = append(stringsKey, tsAndRunIDPrefix...)
	return key
}

func (r *rocksdbStore) UpsertMany(ctx context.Context, objs []*storage.ComplianceStrings) error {
	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()

	for _, obj := range objs {
		key := r.createKay()
		serialized, err := obj.Marshal()
		if err != nil {
			return errors.Wrap(err, "serializing results")
		}

		// Store results under the key "compliance-run-results\x00results:CLUSTER:STANDARD:REVERSE_TIMESTAMP:RUN_ID
		batch.Put(key, serialized)
	}
	return r.db.Write(writeOptions, batch)
}
