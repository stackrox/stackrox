package crud

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/dackbox"
)

type upserterImpl struct {
	keyFunc ProtoKeyFunction

	partials []PartialUpserter
}

// UpsertIn saves the input object and adds a reference to it from the input parentKey if one is passed in.
func (uc upserterImpl) UpsertIn(parentKey []byte, msg proto.Message, dackTxn *dackbox.Transaction) error {
	// Generate key.
	key := uc.keyFunc(msg)

	// Upsert any partial objects, this may alter the input msg object.
	for _, partial := range uc.partials {
		var err error
		if msg, err = partial.UpsertPartialIn(key, msg, dackTxn); err != nil {
			return err
		}
	}

	// If a parent key is set, add the generated key to the parent's child list.
	if parentKey != nil {
		if err := dackTxn.Graph().AddRefs(parentKey, key); err != nil {
			return err
		}
	}

	// Marshal an upsert the base object.
	toWrite, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	err = dackTxn.BadgerTxn().Set(key, toWrite)
	if err != nil {
		return err
	}
	return nil
}

// partialUpserterImpl configures how to write part of an object.
type partialUpserterImpl struct {
	splitFunc ProtoSplitFunction

	upserter Upserter
}

// UpsertIn splits the input object and stores partial by-products using the configured upserter.
func (uc partialUpserterImpl) UpsertPartialIn(parentKey []byte, msg proto.Message, dackTxn *dackbox.Transaction) (proto.Message, error) {
	newBase, partialValues := uc.splitFunc(msg)
	for _, partial := range partialValues {
		if err := uc.upserter.UpsertIn(parentKey, partial, dackTxn); err != nil {
			return nil, err
		}
	}
	return newBase, nil
}
