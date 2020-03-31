package db

import "github.com/gogo/protobuf/proto"

// Crud provides a simple crud layer on top of a DB supporting proto messages
type Crud interface {
	// Read functions
	Count() (int, error)
	Exists(id string) (bool, error)
	GetKeys() ([]string, error)
	Get(id string) (proto.Message, bool, error)
	GetMany(ids []string) (msgs []proto.Message, indices []int, err error)
	Walk(func(msg proto.Message) error) error

	// Modifying functions
	Upsert(kv proto.Message) error
	UpsertMany(msgs []proto.Message) error

	Delete(id string) error
	DeleteMany(ids []string) error

	// Helper functions
	AckKeysIndexed(keys ...string) error
	GetKeysToIndex() ([]string, error)
}
