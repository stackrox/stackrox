package crud

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/dackbox"
)

type readerImpl struct {
	allocFunc ProtoAllocFunction

	partials []PartialReader
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
		msg, err = rc.readInPartials(k, msg, dackTxn)
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
		ret = append(ret, append([]byte{}, k...))
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
	return rc.readInPartials(key, result, dackTxn)
}

// ReadIn returns the object saved under the given key in the given transaction.
func (rc *readerImpl) readInPartials(key []byte, msg proto.Message, dackTxn *dackbox.Transaction) (proto.Message, error) {
	// Merge in any partial objects.
	var err error
	for _, partial := range rc.partials {
		msg, err = partial.ReadPartialIn(key, msg, dackTxn)
		if err != nil {
			return nil, err
		}
	}
	return msg, nil
}

// PartialReadConfig describes how to read part of a higher level object.
type partialReaderImpl struct {
	matchFunc KeyMatchFunction
	mergeFunc ProtoMergeFunction

	reader Reader
}

// ReadIn reads in partial data to a higher level object.
func (rp *partialReaderImpl) ReadPartialIn(key []byte, mergeTo proto.Message, dackTxn *dackbox.Transaction) (proto.Message, error) {
	toKeys := dackTxn.Graph().GetRefsFrom(key)
	partials := make([]proto.Message, 0, len(toKeys))
	for _, key := range toKeys {
		if !rp.matchFunc(key) {
			continue
		}

		partial, err := rp.reader.ReadIn(key, dackTxn)
		if err != nil {
			return nil, err
		}

		partials = append(partials, partial)
	}
	return rp.mergeFunc(mergeTo, partials...), nil
}
