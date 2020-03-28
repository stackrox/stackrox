package m29to30

import (
	"bytes"
	"encoding/binary"

	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

const (
	legacyFieldSep = "\x00"
)

var (
	legacyGroupsBucketName = []byte("groups")
	newGroupsBucketName    = []byte("groups2")
	migration              = types.Migration{
		StartingSeqNum: 30,
		VersionAfter:   storage.Version{SeqNum: 31},
		Run:            updateGroupKeyFormat,
	}
)

func updateGroupKeyFormat(boltDB *bolt.DB, _ *badger.DB) error {
	return boltDB.Update(func(tx *bolt.Tx) error {
		oldBucket := tx.Bucket(legacyGroupsBucketName)
		if oldBucket == nil {
			return nil
		}

		newBucket, err := tx.CreateBucket(newGroupsBucketName)
		if err != nil {
			return errors.Wrap(err, "could not create new groups bucket")
		}

		err = oldBucket.ForEach(func(k, v []byte) error {
			newK := reformatKey(k)
			return newBucket.Put(newK, v)
		})
		if err != nil {
			return err
		}

		return tx.DeleteBucket(legacyGroupsBucketName)
	})
}

func reformatKey(oldKey []byte) []byte {
	parts := bytes.Split(oldKey, []byte(legacyFieldSep))

	// Be very forgiving when parsing keys, even if they seem malformed.
	for len(parts) < 3 {
		parts = append(parts, nil)
	}

	// If there are more than 3 parts, that means there is a null byte in one of the properties. It can't be the auth
	// provider ID, so we assume that it is the "key" part (this is the only instance we've ever observed in the wild).
	if len(parts) > 3 {
		fixedParts := [][]byte{
			parts[0],
			bytes.Join(parts[1:len(parts)-1], []byte(legacyFieldSep)),
			parts[len(parts)-1],
		}
		parts = fixedParts
	}

	var varIntBuf [binary.MaxVarintLen64]byte

	var newKey []byte
	for _, part := range parts {
		l := binary.PutUvarint(varIntBuf[:], uint64(len(part)))
		newKey = append(newKey, varIntBuf[:l]...)
		newKey = append(newKey, part...)
	}

	return newKey
}

func init() {
	migrations.MustRegisterMigration(migration)
}
