package processtagsstore

import (
	"encoding/json"
	"sort"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/analystnotes"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"go.etcd.io/bbolt"
)

type storeImpl struct {
	bucketRef bolthelper.BucketRef
}

func sortMarshalAndPutTags(deploymentSubBucket *bbolt.Bucket, serializedKey []byte, tags []string) error {
	sort.Strings(tags)
	marshaled, err := json.Marshal(&tags)
	if err != nil {
		return errors.Wrap(err, "JSON marshaling")
	}
	if err := deploymentSubBucket.Put(serializedKey, marshaled); err != nil {
		return errors.Wrap(err, "putting into deployment sub-bucket")
	}
	return nil
}

func getExistingTags(deploymentSubBucket *bbolt.Bucket, serializedKey []byte) ([]string, error) {
	bytes := deploymentSubBucket.Get(serializedKey)
	if bytes == nil {
		return nil, nil
	}
	var tags []string
	if err := json.Unmarshal(bytes, &tags); err != nil {
		return nil, errors.Wrap(err, "JSON unmarshaling")
	}
	return tags, nil
}

func (s *storeImpl) GetTagsForProcessKey(key *analystnotes.ProcessNoteKey) ([]string, error) {
	if err := key.Validate(); err != nil {
		return nil, err
	}
	var tags []string
	err := s.bucketRef.View(func(b *bbolt.Bucket) error {
		deploymentSubBucket := b.Bucket([]byte(key.DeploymentID))
		if deploymentSubBucket == nil {
			return nil
		}
		var err error
		tags, err = getExistingTags(deploymentSubBucket, key.Serialize())
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tags, nil
}

var (
	errExitEarly = errors.New("early exit")
)

func (s *storeImpl) WalkTagsForDeployment(deploymentID string, f func(tag string) bool) error {
	seenTags := set.NewStringSet()
	return s.bucketRef.View(func(b *bbolt.Bucket) error {
		deploymentSubBucket := b.Bucket([]byte(deploymentID))
		if deploymentSubBucket == nil {
			return nil
		}
		err := deploymentSubBucket.ForEach(func(k, v []byte) error {
			var tags []string
			if err := json.Unmarshal(v, &tags); err != nil {
				return errors.Wrap(err, "JSON unmarshaling")
			}
			for _, tag := range tags {
				if added := seenTags.Add(tag); added {
					if shouldContinue := f(tag); !shouldContinue {
						return errExitEarly
					}
				}
			}
			return nil
		})
		if err != nil && err != errExitEarly {
			return err
		}
		return nil
	})
}

func (s *storeImpl) UpsertProcessTags(key *analystnotes.ProcessNoteKey, tags []string) error {
	if err := key.Validate(); err != nil {
		return err
	}
	return s.bucketRef.Update(func(b *bbolt.Bucket) error {
		deploymentSubBucket, err := b.CreateBucketIfNotExists([]byte(key.DeploymentID))
		if err != nil {
			return errors.Wrap(err, "creating deployment sub-bucket")
		}
		serializedKey := key.Serialize()
		existingTags, err := getExistingTags(deploymentSubBucket, serializedKey)
		if err != nil {
			return err
		}
		finalTags := sliceutils.StringUnion(existingTags, tags)
		return sortMarshalAndPutTags(deploymentSubBucket, serializedKey, finalTags)
	})
}

func (s *storeImpl) RemoveProcessTags(key *analystnotes.ProcessNoteKey, tags []string) error {
	if err := key.Validate(); err != nil {
		return err
	}
	return s.bucketRef.Update(func(b *bbolt.Bucket) error {
		deploymentSubBucket := b.Bucket([]byte(key.DeploymentID))
		if deploymentSubBucket == nil {
			return nil
		}
		serializedKey := key.Serialize()
		existingTags, err := getExistingTags(deploymentSubBucket, serializedKey)
		if err != nil {
			return err
		}
		finalTags := sliceutils.StringDifference(existingTags, tags)
		if len(finalTags) == 0 {
			return deploymentSubBucket.Delete(serializedKey)
		}
		return sortMarshalAndPutTags(deploymentSubBucket, serializedKey, finalTags)
	})
}
