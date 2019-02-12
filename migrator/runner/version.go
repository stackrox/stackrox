package runner

import (
	"errors"
	"fmt"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
)

var (
	versionBucketName = []byte("version")
	versionKey        = []byte("\x00")
)

// getCurrentSeqNum returns the current seq-num found in the DB.
// A returned value of 0 means that the version bucket was not found in the DB;
// this special value is only returned when we're upgrading from a version pre-2.4.
func getCurrentSeqNum(db *bolt.DB) (int, error) {
	bucketExists, err := bolthelpers.BucketExists(db, versionBucketName)
	if err != nil {
		return 0, fmt.Errorf("checking for version bucket existence: %v", err)
	}
	if !bucketExists {
		return 0, nil
	}
	versionBucket := bolthelpers.TopLevelRef(db, versionBucketName)
	versionBytes, err := bolthelpers.RetrieveElementAtKey(versionBucket, versionKey)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve version: %v", err)
	}
	if versionBytes == nil {
		return 0, errors.New("INVALID STATE: a version bucket existed, but no version was found")
	}
	version := new(storage.Version)
	err = proto.Unmarshal(versionBytes, version)
	if err != nil {
		return 0, fmt.Errorf("unmarshaling version proto: %v", err)
	}
	return int(version.GetSeqNum()), nil
}

func updateVersion(db *bolt.DB, newVersion *storage.Version) error {
	versionBucket := bolthelpers.TopLevelRef(db, versionBucketName)
	bytes, err := proto.Marshal(newVersion)
	if err != nil {
		return fmt.Errorf("marshaling version %+v: %v", newVersion, err)
	}
	return versionBucket.Update(func(b *bolt.Bucket) error {
		return b.Put(versionKey, bytes)
	})
}
