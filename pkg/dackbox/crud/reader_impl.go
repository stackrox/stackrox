package crud

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/sliceutils"
)

type readerImpl struct {
	allocFunc ProtoAllocFunction
}

// ExistsIn returns whether a data for a given key exists in a given transaction.
func (rc *readerImpl) ExistsIn(key []byte, dackTxn *dackbox.Transaction) (bool, error) {
	_, exists, err := dackTxn.Get(key)
	return exists, err
}

// CountIn returns the number of objects in the transaction with the given prefix.
func (rc *readerImpl) CountIn(prefix []byte, dackTxn *dackbox.Transaction) (int, error) {
	return dackTxn.BucketKeyCount(prefix)
}

// ReadAllIn returns all objects with the given prefix in the given transaction.
func (rc *readerImpl) ReadAllIn(prefix []byte, dackTxn *dackbox.Transaction) ([]proto.Message, error) {
	var ret []proto.Message
	err := dackTxn.BucketForEach(prefix, false, func(k, v []byte) error {
		// Read in the base data to the result.
		msg := rc.allocFunc()
		err := proto.Unmarshal(v, msg)
		if err != nil {
			return err
		}

		// Add the read result with the partial data added to the list of results.
		ret = append(ret, msg)
		return nil
	})
	return ret, err
}

// ReadAllIn returns all objects with the given prefix in the given transaction.
func (rc *readerImpl) ReadKeysIn(prefix []byte, dackTxn *dackbox.Transaction) ([][]byte, error) {
	var ret [][]byte
	err := dackTxn.BucketKeyForEach(prefix, false, func(k []byte) error {
		ret = append(ret, sliceutils.ShallowClone(k))
		return nil
	})
	return ret, err
}

// ReadIn returns the object saved under the given key in the given transaction.
func (rc *readerImpl) ReadIn(key []byte, dackTxn *dackbox.Transaction) (proto.Message, error) {
	// Read the top level object from the DB.
	value, exists, err := dackTxn.Get(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	result := rc.allocFunc()
	if err := proto.Unmarshal(value, result); err != nil {
		return nil, err
	}
	return result, nil
}
