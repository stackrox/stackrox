package proto

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/bolthelper/crud/generic"
)

type messageCrudImpl struct {
	genericCrud generic.Crud
	keyFunc     func(message proto.Message) []byte
}

// Read reads and returns a single proto message from bolt.
func (crud *messageCrudImpl) Read(id string) (msg proto.Message, err error) {
	x, err := crud.genericCrud.Read([]byte(id))
	if err != nil {
		return nil, err
	}
	if x == nil {
		return nil, nil
	}
	return x.(proto.Message), nil
}

func idsToKeyPaths(ids []string) []generic.KeyPath {
	paths := make([]generic.KeyPath, len(ids))
	for i, id := range ids {
		paths[i] = generic.KeyPath{generic.Key(id)}
	}
	return paths
}

// ReadBatch reads and returns a list of proto messages for a list of ids in the same order.
func (crud *messageCrudImpl) ReadBatch(ids []string) ([]proto.Message, []int, error) {
	results, missingIndices, err := crud.genericCrud.ReadBatch(idsToKeyPaths(ids)...)
	if err != nil {
		return nil, nil, err
	}
	msgResults := make([]proto.Message, len(results))
	for i, res := range results {
		msgResults[i] = res.(proto.Message)
	}
	return msgResults, missingIndices, nil
}

// ReadAll returns all of the proto messages stored in the bucket.
func (crud *messageCrudImpl) ReadAll() ([]proto.Message, error) {
	results, err := crud.genericCrud.ReadAll(0)
	if err != nil {
		return nil, err
	}
	msgResults := make([]proto.Message, len(results))
	for i, res := range results {
		msgResults[i] = res.Value.(proto.Message)
	}
	return msgResults, nil
}

func (crud *messageCrudImpl) Count() (int, error) {
	return crud.genericCrud.CountLeaves(0)
}

// Create creates a new entry in bolt for the input message.
// Returns an error if an entry with a matching id exists.
func (crud *messageCrudImpl) Create(msg proto.Message) error {
	return crud.genericCrud.Create(msg)
}

// Create creates new entries in bolt for the input messages.
// Returns an error if any entry with a matching id already exists.
func (crud *messageCrudImpl) CreateBatch(msgs []proto.Message) error {
	entries := make([]generic.Entry, len(msgs))
	for i, msg := range msgs {
		entries[i] = generic.Entry{Value: msg}
	}
	return crud.genericCrud.CreateBatch(entries)
}

// Update updates a new entry in bolt for the input message.
// Returns an error an entry with the same id does not already exist.
func (crud *messageCrudImpl) Update(msg proto.Message) (uint64, uint64, error) {
	return crud.genericCrud.Update(msg)
}

// Update updates the entries in bolt for the input messages.
// Returns an error if any input message does not have an existing entry.
func (crud *messageCrudImpl) UpdateBatch(msgs []proto.Message) (uint64, uint64, error) {
	entries := make([]generic.Entry, len(msgs))
	for i, msg := range msgs {
		entries[i] = generic.Entry{Value: msg}
	}
	return crud.genericCrud.UpdateBatch(entries)
}

// Upsert upserts the input message into bolt whether or not an entry with the same id already exists.
func (crud *messageCrudImpl) Upsert(msg proto.Message) (uint64, uint64, error) {
	return crud.genericCrud.Upsert(msg)
}

// Upsert upserts the input messages into bolt whether or not entries with the same ids already exist.
func (crud *messageCrudImpl) UpsertBatch(msgs []proto.Message) (uint64, uint64, error) {
	entries := make([]generic.Entry, len(msgs))
	for i, msg := range msgs {
		entries[i] = generic.Entry{Value: msg}
	}
	return crud.genericCrud.UpsertBatch(entries)
}

// Delete delete the input message in bolt whether or not an entry with the same id exists.
func (crud *messageCrudImpl) Delete(id string) (uint64, uint64, error) {
	return crud.genericCrud.Delete(generic.Key(id))
}

// DeleteBatch deletes the messages associated with all of the input ids in bolt.
func (crud *messageCrudImpl) DeleteBatch(ids []string) (uint64, uint64, error) {
	return crud.genericCrud.DeleteBatch(idsToKeyPaths(ids)...)
}

func (crud *messageCrudImpl) KeyFunc(message proto.Message) []byte {
	return crud.keyFunc(message)
}
