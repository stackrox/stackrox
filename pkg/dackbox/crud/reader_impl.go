package crud

import (
	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox"
)

type readerImpl struct {
	allocFunc ProtoAllocFunction

	partials []PartialReader
}

// ExistsIn returns whether a data for a given key exists in a given transaction.
func (rc readerImpl) ExistsIn(key []byte, dackTxn *dackbox.Transaction) (bool, error) {
	_, err := dackTxn.BadgerTxn().Get(key)
	if err == badger.ErrKeyNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// CountIn returns the number of objects in the transaction with the given prefix.
func (rc readerImpl) CountIn(prefix []byte, dackTxn *dackbox.Transaction) (int, error) {
	return badgerhelper.BucketKeyCount(dackTxn.BadgerTxn(), prefix)
}

var foreachOptions = badgerhelper.ForEachOptions{
	IteratorOptions: &badger.IteratorOptions{
		PrefetchValues: true,
		PrefetchSize:   4,
	},
}

// ReadAllIn returns all objects with the given prefix in the given transaction.
func (rc readerImpl) ReadAllIn(prefix []byte, dackTxn *dackbox.Transaction) ([]proto.Message, error) {
	var ret []proto.Message
	err := badgerhelper.BucketForEach(dackTxn.BadgerTxn(), prefix, foreachOptions, func(k, v []byte) error {
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

// ReadIn returns the object saved under the given key in the given transaction.
func (rc readerImpl) ReadIn(key []byte, dackTxn *dackbox.Transaction) (proto.Message, error) {
	// Read the top level object from the DB.
	item, err := dackTxn.BadgerTxn().Get(key)
	if err != badger.ErrKeyNotFound && err != nil {
		return nil, err
	} else if err == badger.ErrKeyNotFound {
		return nil, nil
	}

	result := rc.allocFunc()
	if err = item.Value(func(val []byte) error {
		return proto.Unmarshal(val, result)
	}); err != nil {
		return nil, err
	}
	return rc.readInPartials(key, result, dackTxn)
}

// ReadIn returns the object saved under the given key in the given transaction.
func (rc readerImpl) readInPartials(key []byte, msg proto.Message, dackTxn *dackbox.Transaction) (proto.Message, error) {
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
func (rp partialReaderImpl) ReadPartialIn(key []byte, mergeTo proto.Message, dackTxn *dackbox.Transaction) (proto.Message, error) {
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
