package runner

import (
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
// A returned value of 0 means that no seq-num was found (since 0 is an invalid seq-num in our scheme).
func getCurrentSeqNum(db *bolt.DB) (int, error) {
	versionBucket := bolthelpers.TopLevelRef(db, versionBucketName)
	versionBytes, err := bolthelpers.RetrieveElementAtKey(versionBucket, versionKey)
	if err != nil {
		return 0, err
	}
	if versionBytes == nil {
		return 0, nil
	}
	version := new(storage.Version)
	err = proto.Unmarshal(versionBytes, version)
	if err != nil {
		return 0, err
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
