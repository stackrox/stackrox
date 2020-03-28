package store

import (
	"fmt"
	"time"

	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/compliance"
	dsTypes "github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
)

const (
	maxFailedRuns = 10

	resultCacheExpiry = 30 * time.Second
)

var (
	resultsBucketName = []byte("compliance-run-results")

	resultsKey  = []byte("results")
	metadataKey = []byte("metadata")
	stringsKey  = []byte("strings")

	log = logging.LoggerForModule()
)

// NewBoltStore returns a compliance results store that is backed by bolt.
func NewBoltStore(db *bbolt.DB) (Store, error) {
	return newBoltStore(db)
}

func newBoltStore(db *bbolt.DB) (*boltStore, error) {
	if err := bolthelper.RegisterBucket(db, resultsBucketName); err != nil {
		return nil, err
	}
	cache := expiringcache.NewExpiringCache(resultCacheExpiry, expiringcache.UpdateExpirationOnGets)

	return &boltStore{
		resultsBucket: bolthelper.TopLevelRef(db, resultsBucketName),
		cacheResults:  cache,
	}, nil
}

type boltStore struct {
	resultsBucket bolthelper.BucketRef

	cacheResults expiringcache.Cache
}

type resultsFuture struct {
	resultsWithStatus dsTypes.ResultsWithStatus
	once              sync.Once
}

func (r *resultsFuture) Get(bucket *bbolt.Bucket, flags dsTypes.GetFlags) dsTypes.ResultsWithStatus {
	r.once.Do(func() {
		r.resultsWithStatus = getLatestRunResults(bucket, flags)
	})
	return r.resultsWithStatus
}

func (s *boltStore) GetLatestRunResults(clusterID, standardID string, flags dsTypes.GetFlags) (dsTypes.ResultsWithStatus, error) {
	allResults, err := s.GetLatestRunResultsBatch([]string{clusterID}, []string{standardID}, flags)
	if err != nil {
		return dsTypes.ResultsWithStatus{}, err
	}
	if len(allResults) == 0 {
		return dsTypes.ResultsWithStatus{}, fmt.Errorf("no results found for cluster %q and standard %q", clusterID, standardID)
	}
	return allResults[compliance.ClusterStandardPair{ClusterID: clusterID, StandardID: standardID}], nil
}

func loadMessageStrings(resultsBucket *bbolt.Bucket, resultsProto *storage.ComplianceRunResults) error {
	var stringsProto storage.ComplianceStrings
	stringsBytes := resultsBucket.Get(stringsKey)
	if stringsBytes != nil {
		if err := stringsProto.Unmarshal(stringsBytes); err != nil {
			return err
		}
	}
	if !reconstituteStrings(resultsProto, &stringsProto) {
		return errors.New("some message strings could not be loaded")
	}
	return nil
}

func readResults(resultsBucket *bbolt.Bucket, flags dsTypes.GetFlags) (*storage.ComplianceRunMetadata, *storage.ComplianceRunResults, error) {
	metadata, err := readMetadata(resultsBucket)
	if err != nil {
		return nil, nil, err
	}
	if !metadata.GetSuccess() {
		return metadata, nil, nil
	}
	resultsBytes := resultsBucket.Get(resultsKey)
	if resultsBytes == nil {
		return nil, nil, errors.New("metadata indicated success, but no results data was found")
	}

	var results storage.ComplianceRunResults
	if err := results.Unmarshal(resultsBytes); err != nil {
		return nil, nil, errors.Wrap(err, "unmarshalling results")
	}

	results.RunMetadata = metadata

	if flags&(dsTypes.WithMessageStrings|dsTypes.RequireMessageStrings) != 0 {
		if err := loadMessageStrings(resultsBucket, &results); err != nil {
			if flags&dsTypes.RequireMessageStrings != 0 {
				return nil, nil, errors.Wrap(err, "loading message strings")
			}
			log.Errorf("Could not load message strings for compliance run results: %v", err)
		}
	}
	return metadata, &results, nil
}

func readMetadata(resultsBucket *bbolt.Bucket) (*storage.ComplianceRunMetadata, error) {
	metadataBytes := resultsBucket.Get(metadataKey)
	if metadataBytes == nil {
		return nil, errors.New("bucket does not have a metadata entry")
	}
	var metadata storage.ComplianceRunMetadata
	if err := metadata.Unmarshal(metadataBytes); err != nil {
		return nil, errors.Wrap(err, "unmarshalling metadata")
	}
	return &metadata, nil
}

func getLatestRunResults(standardBucket *bbolt.Bucket, flags dsTypes.GetFlags) dsTypes.ResultsWithStatus {
	cursor := standardBucket.Cursor()

	var results dsTypes.ResultsWithStatus
	for latestRunBucketKey, _ := cursor.Last(); latestRunBucketKey != nil; latestRunBucketKey, _ = cursor.Prev() {
		runBucket := standardBucket.Bucket(latestRunBucketKey)
		if runBucket == nil {
			continue
		}

		metadata, runResults, err := readResults(runBucket, flags)
		if err != nil {
			log.Errorf("Could not read results from bucket %s: %v", string(latestRunBucketKey), err)
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

func getSpecificRunResults(standardBucket *bbolt.Bucket, runID string, flags dsTypes.GetFlags) (dsTypes.ResultsWithStatus, error) {
	cursor := standardBucket.Cursor()

	var results dsTypes.ResultsWithStatus
	found := false
	for latestRunBucketKey, _ := cursor.Last(); latestRunBucketKey != nil && !found; latestRunBucketKey, _ = cursor.Prev() {
		_, bucketRunID := stringutils.Split2(string(latestRunBucketKey), ":")
		if bucketRunID != runID {
			continue
		}

		runBucket := standardBucket.Bucket(latestRunBucketKey)
		if runBucket == nil {
			return dsTypes.ResultsWithStatus{}, errors.Errorf("unexpected bolt DB structure: key %v does not reference a bucket", string(latestRunBucketKey))
		}

		metadata, runResults, err := readResults(runBucket, flags)
		if err != nil {
			return dsTypes.ResultsWithStatus{}, errors.Errorf("could not read results from bucket %s: %v", string(latestRunBucketKey), err)
		}

		if runResults == nil {
			results.FailedRuns = []*storage.ComplianceRunMetadata{metadata}
		} else {
			results.LastSuccessfulResults = runResults
		}

		found = true // breaks loop
	}
	if !found {
		return dsTypes.ResultsWithStatus{}, errors.Errorf("compliance results for run ID %q not found", runID)
	}

	return results, nil
}

func (s *boltStore) GetSpecificRunResults(clusterID, standardID, runID string, flags dsTypes.GetFlags) (dsTypes.ResultsWithStatus, error) {
	var results dsTypes.ResultsWithStatus
	err := s.resultsBucket.View(func(b *bbolt.Bucket) error {
		clusterBucket := b.Bucket([]byte(clusterID))
		if clusterBucket == nil {
			return errors.Errorf("no compliance runs for cluster %q found", clusterID)
		}
		standardBucket := clusterBucket.Bucket([]byte(standardID))
		if standardBucket == nil {
			return errors.Errorf("no compliance runs for standard %q in cluster %q found", standardID, clusterID)
		}

		var err error
		results, err = getSpecificRunResults(standardBucket, runID, flags)
		return err
	})

	if err != nil {
		return dsTypes.ResultsWithStatus{}, err
	}
	return results, nil
}

func (s *boltStore) GetLatestRunResultsBatch(clusterIDs, standardIDs []string, flags dsTypes.GetFlags) (map[compliance.ClusterStandardPair]dsTypes.ResultsWithStatus, error) {
	results := make(map[compliance.ClusterStandardPair]dsTypes.ResultsWithStatus)
	err := s.resultsBucket.View(func(b *bbolt.Bucket) error {
		for _, clusterID := range clusterIDs {
			clusterBucket := b.Bucket([]byte(clusterID))
			if clusterBucket == nil {
				continue
			}
			for _, standardID := range standardIDs {
				standardBucket := clusterBucket.Bucket([]byte(standardID))
				if standardBucket == nil {
					continue
				}

				pair := compliance.ClusterStandardPair{
					ClusterID:  clusterID,
					StandardID: standardID,
				}

				// Top level caches (cluster, standard) tuple and returns an expiring cache that is keyed off the flags
				flagCache := s.cacheResults.GetOrSet(pair, expiringcache.NewExpiringCache(resultCacheExpiry)).(expiringcache.Cache)

				future := &resultsFuture{}
				future = flagCache.GetOrSet(flags, future).(*resultsFuture)
				results[pair] = future.Get(standardBucket, flags)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func getLatestRunMetadata(standardBucket *bbolt.Bucket) dsTypes.ComplianceRunsMetadata {
	cursor := standardBucket.Cursor()
	var results dsTypes.ComplianceRunsMetadata
	for latestRunBucketKey, _ := cursor.Last(); latestRunBucketKey != nil; latestRunBucketKey, _ = cursor.Prev() {
		runBucket := standardBucket.Bucket(latestRunBucketKey)
		if runBucket == nil {
			continue
		}
		metadata, err := readMetadata(runBucket)
		if err != nil {
			log.Errorf("Could not read results from bucket %s: %v", string(latestRunBucketKey), err)
			continue
		}
		if metadata == nil {
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

func (s *boltStore) GetLatestRunMetadataBatch(clusterID string, standardIDs []string) (map[compliance.ClusterStandardPair]dsTypes.ComplianceRunsMetadata, error) {
	results := make(map[compliance.ClusterStandardPair]dsTypes.ComplianceRunsMetadata)
	clusterIDs := []string{clusterID}
	err := s.resultsBucket.View(func(b *bbolt.Bucket) error {
		for _, clusterID := range clusterIDs {
			clusterBucket := b.Bucket([]byte(clusterID))
			if clusterBucket == nil {
				continue
			}
			for _, standardID := range standardIDs {
				standardBucket := clusterBucket.Bucket([]byte(standardID))
				if standardBucket == nil {
					continue
				}
				metadata := getLatestRunMetadata(standardBucket)
				results[compliance.ClusterStandardPair{ClusterID: clusterID, StandardID: standardID}] = metadata
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (s *boltStore) GetLatestRunResultsFiltered(clusterIDFilter, standardIDFilter func(string) bool, flags dsTypes.GetFlags) (map[compliance.ClusterStandardPair]dsTypes.ResultsWithStatus, error) {
	results := make(map[compliance.ClusterStandardPair]dsTypes.ResultsWithStatus)
	err := s.resultsBucket.View(func(b *bbolt.Bucket) error {
		clusterCursor := b.Cursor()
		for clusterKey, _ := clusterCursor.First(); clusterKey != nil; clusterKey, _ = clusterCursor.Next() {
			clusterID := string(clusterKey)
			if !clusterIDFilter(clusterID) {
				continue
			}

			clusterBucket := b.Bucket(clusterKey)
			if clusterBucket == nil {
				continue
			}

			standardCursor := clusterBucket.Cursor()
			for standardKey, _ := standardCursor.First(); standardKey != nil; standardKey, _ = standardCursor.Next() {
				standardID := string(standardKey)
				if !standardIDFilter(standardID) {
					continue
				}

				standardBucket := clusterBucket.Bucket(standardKey)
				if standardBucket == nil {
					continue
				}

				resultsWithStatus := getLatestRunResults(standardBucket, flags)
				results[compliance.ClusterStandardPair{ClusterID: clusterID, StandardID: standardID}] = resultsWithStatus
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func createRunBucket(root *bbolt.Bucket, metadata *storage.ComplianceRunMetadata) (*bbolt.Bucket, error) {
	clusterID := metadata.GetClusterId()
	if clusterID == "" {
		return nil, errors.New("run has an empty cluster ID")
	}
	clusterKey := []byte(clusterID)
	standardID := metadata.GetStandardId()
	if standardID == "" {
		return nil, errors.New("run has an empty standard ID")
	}
	standardKey := []byte(standardID)
	runID := metadata.GetRunId()
	if runID == "" {
		return nil, errors.New("run has an empty ID")
	}
	finishTime, err := types.TimestampFromProto(metadata.GetFinishTimestamp())
	if err != nil {
		return nil, errors.Wrap(err, "run has an invalid finish timestamp")
	}
	microTS := timestamp.FromGoTime(finishTime)
	runKey := []byte(fmt.Sprintf("%016X:%s", microTS, runID))

	clusterBucket, err := root.CreateBucketIfNotExists(clusterKey)
	if err != nil {
		return nil, errors.Wrapf(err, "creating bucket for cluster %q", clusterID)
	}
	standardBucket, err := clusterBucket.CreateBucketIfNotExists(standardKey)
	if err != nil {
		return nil, errors.Wrapf(err, "creating bucket for standard %q", standardID)
	}

	runBucket, err := standardBucket.CreateBucket(runKey)
	if err != nil {
		return nil, errors.Wrapf(err, "creating bucket for run %s", string(runKey))
	}

	return runBucket, nil
}

func (s *boltStore) StoreRunResults(runResults *storage.ComplianceRunResults) error {
	metadata := runResults.GetRunMetadata()
	if metadata == nil {
		return errors.New("run results have no metadata")
	}
	if !metadata.GetSuccess() {
		return errors.New("metadata indicates failure")
	}

	pair := compliance.ClusterStandardPair{ClusterID: metadata.ClusterId, StandardID: metadata.StandardId}
	s.cacheResults.Remove(pair)

	serializedMD, err := metadata.Marshal()
	if err != nil {
		return errors.Wrap(err, "serializing metadata")
	}

	stringsProto := externalizeStrings(runResults)
	serializedStrings, err := stringsProto.Marshal()
	if err != nil {
		return errors.Wrap(err, "serializing message strings")
	}

	serializedResults, err := runResults.Marshal()
	if err != nil {
		return errors.Wrap(err, "serializing results")
	}

	return s.resultsBucket.Update(func(b *bbolt.Bucket) error {
		runBucket, err := createRunBucket(b, metadata)
		if err != nil {
			return errors.Wrap(err, "creating run bucket")
		}
		if err := runBucket.Put(metadataKey, serializedMD); err != nil {
			return err
		}
		if err := runBucket.Put(resultsKey, serializedResults); err != nil {
			return err
		}
		return runBucket.Put(stringsKey, serializedStrings)
	})
}

func (s *boltStore) StoreFailure(metadata *storage.ComplianceRunMetadata) error {
	if metadata.Success || metadata.ErrorMessage == "" {
		return errors.New("metadata passed to StoreFailure must indicate failure and have an error message set")
	}

	pair := compliance.ClusterStandardPair{ClusterID: metadata.ClusterId, StandardID: metadata.StandardId}
	s.cacheResults.Remove(pair)

	serializedMD, err := metadata.Marshal()
	if err != nil {
		return errors.Wrap(err, "serializing metadata")
	}

	return s.resultsBucket.Update(func(b *bbolt.Bucket) error {
		runBucket, err := createRunBucket(b, metadata)
		if err != nil {
			return errors.Wrap(err, "creating run bucket")
		}
		return runBucket.Put(metadataKey, serializedMD)
	})
}

func (s *boltStore) clear() error {
	s.cacheResults.RemoveAll()
	return s.resultsBucket.Update(func(b *bbolt.Bucket) error {
		return b.ForEach(func(k, _ []byte) error {
			return b.DeleteBucket(k)
		})
	})
}
