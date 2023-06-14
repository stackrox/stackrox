package db

import "github.com/gogo/protobuf/proto"

// Crud provides a simple crud layer on top of a DB supporting proto messages
//
//go:generate mockgen-wrapper
type Crud interface {
	// Read functions
	Count() (int, error)
	Exists(id string) (bool, error)
	GetKeys() ([]string, error)
	Get(id string) (proto.Message, bool, error)
	GetMany(ids []string) (msgs []proto.Message, indices []int, err error)
	Walk(func(msg proto.Message) error) error
	WalkAllWithID(func(id []byte, msg proto.Message) error) error

	// Modifying functions
	Upsert(kv proto.Message) error
	UpsertMany(msgs []proto.Message) error
	UpsertWithID(id string, msg proto.Message) error
	UpsertManyWithIDs(ids []string, msgs []proto.Message) error

	Delete(id string) error
	DeleteMany(ids []string) error
}
