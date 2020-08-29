package rocksdb

import (
	"bytes"
	"fmt"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/datastore/internal/store"
	dsTypes "github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/rocksdb"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/tecbot/gorocksdb"
)

const (
	maxFailedRuns = 10

	resultCacheExpiry = 30 * time.Second
)

var (
	readOptions  = generic.DefaultReadOptions()
	writeOptions = generic.DefaultWriteOptions()

	resultsBucketName = []byte("compliance-run-results")

	resultsKey  = []byte("results")
	metadataKey = []byte("metadata")
	stringsKey  = []byte("strings")

	log = logging.LoggerForModule()
)

// NewRocksdbStore returns a compliance results store that is backed by RocksDB.
func NewRocksdbStore(db *rocksdb.RocksDB) store.Store {
	globaldb.RegisterBucket(resultsBucketName, "ComplianceRunResults")
	cache := expiringcache.NewExpiringCache(resultCacheExpiry, expiringcache.UpdateExpirationOnGets)
	return &rocksdbStore{
		db:           db,
		cacheResults: cache,
	}
}

type rocksdbStore struct {
	db *rocksdb.RocksDB

	cacheResults expiringcache.Cache
}

type keyMaker struct {
	partialMetadataPrefix []byte
	partialResultsPrefix  []byte
	partialStringsPrefix  []byte
}

func (k *keyMaker) getMetadataIterationPrefix() []byte {
	return k.partialMetadataPrefix
}

func (k *keyMaker) getKeysForMetadata(metadata *storage.ComplianceRunMetadata) ([]byte, []byte, []byte, error) {
	runID := metadata.GetRunId()
	if runID == "" {
		return nil, nil, nil, errors.New("run has an empty ID")
	}
	finishTime, err := types.TimestampFromProto(metadata.GetFinishTimestamp())
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "run has an invalid finish timestamp")
	}

	tsBytes := []byte(fmt.Sprintf("%016X", timestamp.FromGoTime(finishTime)))
	// Invert the bits of each byte of the timestamp in order to have the most recent timestamp first
	for i, tsByte := range tsBytes {
		tsBytes[i] = -tsByte
	}
	separatorAndRunID := []byte(fmt.Sprintf(":%s", runID))
	tsAndRunIDPrefix := append(tsBytes, separatorAndRunID...)

	metadataKey := append([]byte{}, k.partialMetadataPrefix...)
	metadataKey = append(metadataKey, tsAndRunIDPrefix...)

	resultsKey := append([]byte{}, k.partialResultsPrefix...)
	resultsKey = append(resultsKey, tsAndRunIDPrefix...)

	stringsKey := append([]byte{}, k.partialStringsPrefix...)
	stringsKey = append(stringsKey, tsAndRunIDPrefix...)

	return metadataKey, resultsKey, stringsKey, nil
}

func getKeyMaker(clusterID, standardID string) *keyMaker {
	metadataPrefix, resultsPrefix, stringsPrefix := getClusterStandardPrefixes(clusterID, standardID)

	return &keyMaker{
		partialMetadataPrefix: metadataPrefix,
		partialResultsPrefix:  resultsPrefix,
		partialStringsPrefix:  stringsPrefix,
	}
}

func getClusterStandardPrefixes(clusterID, standardID string) ([]byte, []byte, []byte) {
	// trailing colon is intentional, this prefix will always be followed by a timestamp and a run ID
	partialPrefix := fmt.Sprintf("%s:%s:", clusterID, standardID)

	partialMetadataPrefix := []byte(fmt.Sprintf("%s:%s", string(metadataKey), partialPrefix))
	metadataPrefix := dbhelper.GetBucketKey(resultsBucketName, partialMetadataPrefix)

	partialResultsPrefix := []byte(fmt.Sprintf("%s:%s", string(resultsKey), partialPrefix))
	resultsPrefix := dbhelper.GetBucketKey(resultsBucketName, partialResultsPrefix)

	partialStringsPrefix := []byte(fmt.Sprintf("%s:%s", string(stringsKey), partialPrefix))
	stringsPrefix := dbhelper.GetBucketKey(resultsBucketName, partialStringsPrefix)

	return metadataPrefix, resultsPrefix, stringsPrefix
}

type getLatestResultsArgs struct {
	db       *rocksdb.RocksDB
	iterator *gorocksdb.Iterator
	keyMaker *keyMaker
	flags    dsTypes.GetFlags
}

type rocksdbResultsFuture struct {
	resultsWithStatus dsTypes.ResultsWithStatus
	once              sync.Once
}

func (r *rocksdbResultsFuture) Get(getArgs *getLatestResultsArgs) dsTypes.ResultsWithStatus {
	r.once.Do(func() {
		r.resultsWithStatus = getLatestRunResultsRocksdb(getArgs)
	})
	return r.resultsWithStatus
}

func (r *rocksdbStore) GetSpecificRunResults(clusterID, standardID, runID string, flags dsTypes.GetFlags) (dsTypes.ResultsWithStatus, error) {
	var results dsTypes.ResultsWithStatus

	runIDBytes := []byte(runID)
	keyMaker := getKeyMaker(clusterID, standardID)
	iterationPrefix := keyMaker.getMetadataIterationPrefix()
	iterator := r.db.NewIterator(readOptions)
	defer iterator.Close()
	// Runs are sorted by time so we must iterate over each key to see if it has the correct run ID.
	for iterator.Seek(iterationPrefix); iterator.ValidForPrefix(iterationPrefix); iterator.Next() {
		curKey := iterator.Key().Data()
		if !bytes.HasSuffix(curKey, runIDBytes) {
			continue
		}

		getArgs := &getLatestResultsArgs{
			db:       r.db,
			iterator: iterator,
			keyMaker: keyMaker,
			flags:    flags,
		}
		metadata, runResults, err := unmarshalResults(getArgs)
		if err != nil {
			return dsTypes.ResultsWithStatus{}, errors.Wrapf(err, "could not read results for key %s", curKey)
		}

		if runResults == nil {
			results.FailedRuns = []*storage.ComplianceRunMetadata{metadata}
		} else {
			results.LastSuccessfulResults = runResults
		}
		return results, nil
	}
	return dsTypes.ResultsWithStatus{}, errors.Errorf("compliance results for run ID %q not found", runID)
}

func (r *rocksdbStore) GetLatestRunResults(clusterID, standardID string, flags dsTypes.GetFlags) (dsTypes.ResultsWithStatus, error) {
	allResults, err := r.GetLatestRunResultsBatch([]string{clusterID}, []string{standardID}, flags)
	if err != nil {
		return dsTypes.ResultsWithStatus{}, err
	}
	if len(allResults) == 0 {
		return dsTypes.ResultsWithStatus{}, fmt.Errorf("no results found for cluster %q and standard %q", clusterID, standardID)
	}
	return allResults[compliance.ClusterStandardPair{ClusterID: clusterID, StandardID: standardID}], nil
}

func getLatestRunResultsRocksdb(getArgs *getLatestResultsArgs) dsTypes.ResultsWithStatus {
	var results dsTypes.ResultsWithStatus
	iterationPrefix := getArgs.keyMaker.getMetadataIterationPrefix()
	for ; getArgs.iterator.ValidForPrefix(iterationPrefix); getArgs.iterator.Next() {
		metadata, runResults, err := unmarshalResults(getArgs)
		if err != nil {
			log.Errorf("Could not read results from prefix %s: %v", string(getArgs.iterator.Key().Data()), err)
			continue
		}

		if runResults == nil && len(results.FailedRuns) < maxFailedRuns {
			results.FailedRuns = append(results.FailedRuns, metadata)
		} else if runResults != nil {
			results.LastSuccessfulResults = runResults
			break
		}
	}

	return results
}

func unmarshalMessageStrings(getArgs *getLatestResultsArgs, stringsKey []byte, resultsProto *storage.ComplianceRunResults) error {
	var stringsProto storage.ComplianceStrings
	stringsSlice, err := getArgs.db.Get(readOptions, stringsKey)
	if err != nil {
		return err
	}
	defer stringsSlice.Free()
	stringsBytes := stringsSlice.Data()
	if stringsBytes != nil {
		if err := stringsProto.Unmarshal(stringsBytes); err != nil {
			return err
		}
	}
	if !store.ReconstituteStrings(resultsProto, &stringsProto) {
		return errors.New("some message strings could not be loaded")
	}
	return nil
}

func unmarshalResults(getArgs *getLatestResultsArgs) (*storage.ComplianceRunMetadata, *storage.ComplianceRunResults, error) {
	metadata, err := unmarshalMetadata(getArgs.iterator)
	if err != nil {
		return nil, nil, err
	}
	if !metadata.GetSuccess() {
		return metadata, nil, nil
	}

	_, resultKey, stringsKey, err := getArgs.keyMaker.getKeysForMetadata(metadata)
	if err != nil {
		return nil, nil, err
	}

	resultsSlice, err := getArgs.db.Get(readOptions, resultKey)
	if err != nil {
		return nil, nil, err
	}
	defer resultsSlice.Free()
	resultsBytes := resultsSlice.Data()
	if len(resultsBytes) == 0 {
		return nil, nil, errors.New("metadata indicated success, but no results data was found")
	}

	var results storage.ComplianceRunResults
	if err := results.Unmarshal(resultsBytes); err != nil {
		return nil, nil, errors.Wrap(err, "unmarshalling results")
	}

	results.RunMetadata = metadata

	if getArgs.flags&(dsTypes.WithMessageStrings|dsTypes.RequireMessageStrings) != 0 {
		if err := unmarshalMessageStrings(getArgs, stringsKey, &results); err != nil {
			if getArgs.flags&dsTypes.RequireMessageStrings != 0 {
				return nil, nil, errors.Wrap(err, "loading message strings")
			}
			log.Errorf("Could not load message strings for compliance run results: %v", err)
		}
	}
	return metadata, &results, nil
}

func unmarshalMetadata(iterator *gorocksdb.Iterator) (*storage.ComplianceRunMetadata, error) {
	metadataBytes := iterator.Value().Data()
	if len(metadataBytes) == 0 {
		return nil, errors.New("prefix metadata is empty")
	}
	var metadata storage.ComplianceRunMetadata
	if err := metadata.Unmarshal(metadataBytes); err != nil {
		return nil, errors.Wrap(err, "unmarshalling metadata")
	}
	return &metadata, nil
}

func (r *rocksdbStore) GetLatestRunResultsBatch(clusterIDs, standardIDs []string, flags dsTypes.GetFlags) (map[compliance.ClusterStandardPair]dsTypes.ResultsWithStatus, error) {
	if err := r.db.IncRocksDBInProgressOps(); err != nil {
		return nil, err
	}
	defer r.db.DecRocksDBInProgressOps()

	results := make(map[compliance.ClusterStandardPair]dsTypes.ResultsWithStatus)
	for _, clusterID := range clusterIDs {
		for _, standardID := range standardIDs {
			// Call in a func, the iterator uses a defer to ensure it closes properly
			func() {
				keyMaker := getKeyMaker(clusterID, standardID)
				// Seek to the latest metadata for this cluster/standard pair
				prefix := keyMaker.getMetadataIterationPrefix()
				clusterStandardIterator := r.db.NewIterator(readOptions)
				defer clusterStandardIterator.Close()
				clusterStandardIterator.Seek(prefix)
				if !clusterStandardIterator.ValidForPrefix(prefix) {
					return
				}

				pair := compliance.ClusterStandardPair{
					ClusterID:  clusterID,
					StandardID: standardID,
				}

				// Top level caches (cluster, standard) tuple and returns an expiring cache that is keyed off the flags
				flagCache := r.cacheResults.GetOrSet(pair, expiringcache.NewExpiringCache(resultCacheExpiry)).(expiringcache.Cache)

				future := &rocksdbResultsFuture{}
				future = flagCache.GetOrSet(flags, future).(*rocksdbResultsFuture)
				getArgs := &getLatestResultsArgs{
					db:       r.db,
					iterator: clusterStandardIterator,
					keyMaker: keyMaker,
					flags:    flags,
				}
				results[pair] = future.Get(getArgs)
			}()
		}
	}
	return results, nil
}

func (r *rocksdbStore) GetLatestRunResultsByClusterAndStandard(clusterIDs, standardIDs []string, flags dsTypes.GetFlags) (map[compliance.ClusterStandardPair]dsTypes.ResultsWithStatus, error) {
	results := make(map[compliance.ClusterStandardPair]dsTypes.ResultsWithStatus)
	for _, clusterID := range clusterIDs {
		for _, standardID := range standardIDs {
			func() {
				keyMaker := getKeyMaker(clusterID, standardID)
				iterator := r.db.NewIterator(readOptions)
				defer iterator.Close()
				iterator.Seek(keyMaker.getMetadataIterationPrefix())
				if !iterator.ValidForPrefix(keyMaker.getMetadataIterationPrefix()) {
					// Might want to record missing instead of just returning
					return
				}
				getArgs := &getLatestResultsArgs{
					db:       r.db,
					iterator: iterator,
					keyMaker: keyMaker,
					flags:    flags,
				}
				resultsWithStatus := getLatestRunResultsRocksdb(getArgs)
				results[compliance.ClusterStandardPair{ClusterID: clusterID, StandardID: standardID}] = resultsWithStatus
			}()
		}
	}

	return results, nil
}

func (r *rocksdbStore) getLatestRunMetadata(keyMaker *keyMaker) dsTypes.ComplianceRunsMetadata {
	var results dsTypes.ComplianceRunsMetadata

	prefix := keyMaker.getMetadataIterationPrefix()
	iterator := r.db.NewIterator(readOptions)
	defer iterator.Close()
	for iterator.Seek(prefix); iterator.ValidForPrefix(prefix); iterator.Next() {
		metadata, err := unmarshalMetadata(iterator)
		if err != nil {
			log.Errorf("Could not read results for key %s: %v", string(iterator.Key().Data()), err)
			continue
		}

		if !metadata.GetSuccess() && len(results.FailedRunsMetadata) < maxFailedRuns {
			results.FailedRunsMetadata = append(results.FailedRunsMetadata, metadata)
		} else if metadata.GetSuccess() {
			results.LastSuccessfulRunMetadata = metadata
			break
		}
	}
	return results
}

func (r *rocksdbStore) GetLatestRunMetadataBatch(clusterID string, standardIDs []string) (map[compliance.ClusterStandardPair]dsTypes.ComplianceRunsMetadata, error) {
	results := make(map[compliance.ClusterStandardPair]dsTypes.ComplianceRunsMetadata)
	for _, standardID := range standardIDs {
		keyMaker := getKeyMaker(clusterID, standardID)
		metadata := r.getLatestRunMetadata(keyMaker)
		results[compliance.ClusterStandardPair{ClusterID: clusterID, StandardID: standardID}] = metadata
	}

	return results, nil
}

func (r *rocksdbStore) StoreRunResults(runResults *storage.ComplianceRunResults) error {
	metadata := runResults.GetRunMetadata()
	if metadata == nil {
		return errors.New("run results have no metadata")
	}
	if !metadata.GetSuccess() {
		return errors.New("metadata indicates failure")
	}

	clusterID := metadata.GetClusterId()
	if clusterID == "" {
		return errors.New("run has an empty cluster ID")
	}
	standardID := metadata.GetStandardId()
	if standardID == "" {
		return errors.New("run has an empty standard ID")
	}

	pair := compliance.ClusterStandardPair{ClusterID: clusterID, StandardID: standardID}
	r.cacheResults.Remove(pair)

	serializedMD, err := metadata.Marshal()
	if err != nil {
		return errors.Wrap(err, "serializing metadata")
	}

	stringsProto := store.ExternalizeStrings(runResults)
	serializedStrings, err := stringsProto.Marshal()
	if err != nil {
		return errors.Wrap(err, "serializing message strings")
	}

	serializedResults, err := runResults.Marshal()
	if err != nil {
		return errors.Wrap(err, "serializing results")
	}

	if err := r.db.IncRocksDBInProgressOps(); err != nil {
		return errors.Wrap(err, "communicating with RocksDB")
	}
	defer r.db.DecRocksDBInProgressOps()

	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()

	keyMaker := getKeyMaker(clusterID, standardID)
	mdKey, rKey, sKey, err := keyMaker.getKeysForMetadata(metadata)
	if err != nil {
		return err
	}

	// Store metadata under the key "compliance-run-results\x00metadata:CLUSTER:STANDARD:REVERSE_TIMESTAMP:RUN_ID
	batch.Put(mdKey, serializedMD)
	// Store results under the key "compliance-run-results\x00results:CLUSTER:STANDARD:REVERSE_TIMESTAMP:RUN_ID
	batch.Put(rKey, serializedResults)
	// Store externalized strings under the key "compliance-run-results\x00strings:CLUSTER:STANDARD:REVERSE_TIMESTAMP:RUN_ID
	batch.Put(sKey, serializedStrings)

	if err := r.db.Write(writeOptions, batch); err != nil {
		return errors.Wrap(err, "writing to DB")
	}

	return nil
}

func (r *rocksdbStore) StoreFailure(metadata *storage.ComplianceRunMetadata) error {
	if metadata.Success || metadata.ErrorMessage == "" {
		return errors.New("metadata passed to StoreFailure must indicate failure and have an error message set")
	}

	pair := compliance.ClusterStandardPair{ClusterID: metadata.ClusterId, StandardID: metadata.StandardId}
	r.cacheResults.Remove(pair)

	serializedMD, err := metadata.Marshal()
	if err != nil {
		return errors.Wrap(err, "serializing metadata")
	}

	keyMaker := getKeyMaker(metadata.GetClusterId(), metadata.GetStandardId())
	mdKey, _, _, err := keyMaker.getKeysForMetadata(metadata)
	if err != nil {
		return errors.Wrap(err, "creating metadata key")
	}
	err = r.db.Put(writeOptions, mdKey, serializedMD)
	return errors.Wrap(err, "storing metadata")
}
