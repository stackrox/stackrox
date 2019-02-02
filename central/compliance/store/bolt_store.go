package store

import (
	"errors"
	"fmt"

	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/timestamp"
)

const (
	maxFailedRuns = 10
)

var (
	resultsBucketName = []byte("compliance-run-results")

	resultsKey  = []byte("results")
	metadataKey = []byte("metadata")

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
	return &boltStore{
		resultsBucket: bolthelper.TopLevelRef(db, resultsBucketName),
	}, nil
}

type boltStore struct {
	resultsBucket bolthelper.BucketRef
}

func (s *boltStore) QueryControlResults(query *v1.Query) ([]*storage.ComplianceControlResult, error) {
	return nil, errors.New("not yet implemented")
}

func (s *boltStore) GetLatestRunResults(clusterID, standardID string) (ResultsWithStatus, error) {
	allResults, err := s.GetLatestRunResultsBatch([]string{clusterID}, []string{standardID})
	if err != nil {
		return ResultsWithStatus{}, err
	}
	if len(allResults) == 0 {
		return ResultsWithStatus{}, fmt.Errorf("no results found for cluster %q and standard %q", clusterID, standardID)
	}
	return allResults[compliance.ClusterStandardPair{ClusterID: clusterID, StandardID: standardID}], nil
}

func readResults(resultsBucket *bbolt.Bucket) (*storage.ComplianceRunMetadata, *storage.ComplianceRunResults, error) {
	metadataBytes := resultsBucket.Get(metadataKey)
	if metadataBytes == nil {
		return nil, nil, errors.New("bucket does not have a metadata entry")
	}
	var metadata storage.ComplianceRunMetadata
	if err := metadata.Unmarshal(metadataBytes); err != nil {
		return nil, nil, fmt.Errorf("unmarshalling metadata: %v", err)
	}

	if !metadata.GetSuccess() {
		return &metadata, nil, nil
	}

	resultsBytes := resultsBucket.Get(resultsKey)
	if resultsBytes == nil {
		return nil, nil, errors.New("metadata indicated success, but no results data was found")
	}

	var results storage.ComplianceRunResults
	if err := results.Unmarshal(resultsBytes); err != nil {
		return nil, nil, fmt.Errorf("unmarshalling results: %v", err)
	}
	results.RunMetadata = &metadata
	return &metadata, &results, nil
}

func getLatestRunResults(standardBucket *bbolt.Bucket) ResultsWithStatus {
	cursor := standardBucket.Cursor()

	var results ResultsWithStatus
	for latestRunBucketKey, _ := cursor.Last(); latestRunBucketKey != nil; latestRunBucketKey, _ = cursor.Prev() {
		runBucket := standardBucket.Bucket(latestRunBucketKey)
		if runBucket == nil {
			continue
		}

		metadata, runResults, err := readResults(runBucket)
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

func (s *boltStore) GetLatestRunResultsBatch(clusterIDs, standardIDs []string) (map[compliance.ClusterStandardPair]ResultsWithStatus, error) {
	results := make(map[compliance.ClusterStandardPair]ResultsWithStatus)
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

				resultsWithStatus := getLatestRunResults(standardBucket)
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

func (s *boltStore) GetLatestRunResultsFiltered(clusterIDFilter, standardIDFilter func(string) bool) (map[compliance.ClusterStandardPair]ResultsWithStatus, error) {
	results := make(map[compliance.ClusterStandardPair]ResultsWithStatus)
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

				resultsWithStatus := getLatestRunResults(standardBucket)
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
		return nil, fmt.Errorf("run has an invalid finish timestamp: %v", err)
	}
	microTS := timestamp.FromGoTime(finishTime)
	runKey := []byte(fmt.Sprintf("%016X:%s", microTS, runID))

	clusterBucket, err := root.CreateBucketIfNotExists(clusterKey)
	if err != nil {
		return nil, fmt.Errorf("creating bucket for cluster %q: %v", clusterID, err)
	}
	standardBucket, err := clusterBucket.CreateBucketIfNotExists(standardKey)
	if err != nil {
		return nil, fmt.Errorf("creating bucket for standard %q: %v", standardID, err)
	}

	runBucket, err := standardBucket.CreateBucket(runKey)
	if err != nil {
		return nil, fmt.Errorf("creating bucket for run %s: %v", string(runKey), err)
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

	serializedMD, err := metadata.Marshal()
	if err != nil {
		return fmt.Errorf("serializing metadata: %v", err)
	}
	serializedResults, err := runResults.Marshal()
	if err != nil {
		return fmt.Errorf("serializing results: %v", err)
	}

	return s.resultsBucket.Update(func(b *bbolt.Bucket) error {
		runBucket, err := createRunBucket(b, metadata)
		if err != nil {
			return fmt.Errorf("creating run bucket: %v", err)
		}
		if err := runBucket.Put(metadataKey, serializedMD); err != nil {
			return err
		}
		return runBucket.Put(resultsKey, serializedResults)
	})
}

func (s *boltStore) StoreFailure(metadata *storage.ComplianceRunMetadata) error {
	if metadata.Success || metadata.ErrorMessage == "" {
		return errors.New("metadata passed to StoreFailure must indicate failure and have an error message set")
	}

	serializedMD, err := metadata.Marshal()
	if err != nil {
		return fmt.Errorf("serializing metadata: %v", err)
	}

	return s.resultsBucket.Update(func(b *bbolt.Bucket) error {
		runBucket, err := createRunBucket(b, metadata)
		if err != nil {
			return fmt.Errorf("creating run bucket: %v", err)
		}
		return runBucket.Put(metadataKey, serializedMD)
	})
}

func (s *boltStore) clear() error {
	return s.resultsBucket.Update(func(b *bbolt.Bucket) error {
		return b.ForEach(func(k, _ []byte) error {
			return b.DeleteBucket(k)
		})
	})
}
