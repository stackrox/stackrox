package store

import (
	"errors"
	"fmt"

	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/timestamp"
)

var (
	resultsBucketName = []byte("compliance-run-results")
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

func (s *boltStore) GetLatestRunResults(clusterID, standardID string) (*storage.ComplianceRunResults, error) {
	allResults, err := s.GetLatestRunResultsBatch([]string{clusterID}, []string{standardID})
	if err != nil {
		return nil, err
	}
	if len(allResults) == 0 {
		return nil, fmt.Errorf("no results found for cluster %q and standard %q", clusterID, standardID)
	}
	return allResults[0], nil
}

func getLatestRunResults(standardBucket *bbolt.Bucket) (*storage.ComplianceRunResults, error) {
	_, latestRunResultsBytes := standardBucket.Cursor().Last()
	if latestRunResultsBytes == nil {
		return nil, nil
	}

	var latestRunResults storage.ComplianceRunResults
	if err := latestRunResults.Unmarshal(latestRunResultsBytes); err != nil {
		return nil, fmt.Errorf("unmarshalling run results: %v", err)
	}
	return &latestRunResults, nil
}

func (s *boltStore) GetLatestRunResultsBatch(clusterIDs, standardIDs []string) ([]*storage.ComplianceRunResults, error) {
	var results []*storage.ComplianceRunResults
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

				latestRunResults, err := getLatestRunResults(standardBucket)
				if err != nil {
					return err
				}
				results = append(results, latestRunResults)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (s *boltStore) GetLatestRunResultsFiltered(clusterIDFilter, standardIDFilter func(string) bool) ([]*storage.ComplianceRunResults, error) {
	var results []*storage.ComplianceRunResults
	err := s.resultsBucket.View(func(b *bbolt.Bucket) error {
		clusterCursor := b.Cursor()
		for clusterKey, _ := clusterCursor.First(); clusterKey != nil; clusterKey, _ = clusterCursor.Next() {
			if !clusterIDFilter(string(clusterKey)) {
				continue
			}

			clusterBucket := b.Bucket(clusterKey)
			if clusterBucket == nil {
				continue
			}

			standardCursor := clusterBucket.Cursor()
			for standardKey, _ := standardCursor.First(); standardKey != nil; standardKey, _ = standardCursor.Next() {
				if !standardIDFilter(string(standardKey)) {
					continue
				}

				standardBucket := clusterBucket.Bucket(standardKey)
				if standardBucket == nil {
					continue
				}

				latestRunResults, err := getLatestRunResults(standardBucket)
				if err != nil {
					return err
				}
				results = append(results, latestRunResults)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (s *boltStore) StoreRunResults(run *storage.ComplianceRunResults) error {
	clusterID := run.GetDomain().GetCluster().GetId()
	if clusterID == "" {
		return errors.New("run has an empty cluster ID")
	}
	clusterKey := []byte(clusterID)
	standardID := run.GetRunMetadata().GetStandardId()
	if standardID == "" {
		return errors.New("run has an empty standard ID")
	}
	standardKey := []byte(standardID)
	runID := run.GetRunMetadata().GetRunId()
	if runID == "" {
		return errors.New("run has an empty ID")
	}
	finishTime, err := types.TimestampFromProto(run.GetRunMetadata().GetFinishTimestamp())
	if err != nil {
		return fmt.Errorf("run has an invalid finish timestamp: %v", err)
	}
	microTS := timestamp.FromGoTime(finishTime)
	runKey := []byte(fmt.Sprintf("%016X:%s", microTS, runID))

	resultsData, err := run.Marshal()
	if err != nil {
		return fmt.Errorf("marshalling results: %v", err)
	}

	return s.resultsBucket.Update(func(b *bbolt.Bucket) error {
		clusterBucket, err := b.CreateBucketIfNotExists(clusterKey)
		if err != nil {
			return err
		}
		standardBucket, err := clusterBucket.CreateBucketIfNotExists(standardKey)
		if err != nil {
			return err
		}

		return standardBucket.Put(runKey, resultsData)
	})
}

func (s *boltStore) clear() error {
	return s.resultsBucket.Update(func(b *bbolt.Bucket) error {
		return b.ForEach(func(k, _ []byte) error {
			return b.DeleteBucket(k)
		})
	})
}
