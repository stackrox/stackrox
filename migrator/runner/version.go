package runner

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
)

var (
	versionBucketName = []byte("version")
	versionKey        = []byte("\x00")
)

func getCurrentVersion(db *bolt.DB) (*storage.Version, error) {
	versionBucket := bolthelpers.TopLevelRef(db, versionBucketName)
	versionBytes, err := bolthelpers.RetrieveElementAtKey(versionBucket, versionKey)
	if err != nil {
		return nil, err
	}
	if versionBytes == nil {
		return nil, nil
	}
	version := new(storage.Version)
	err = proto.Unmarshal(versionBytes, version)
	if err != nil {
		return nil, err
	}
	return version, nil
}
