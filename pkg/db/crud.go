package db

import (
	"github.com/stackrox/rox/pkg/protocompat"
)

// Crud provides a simple crud layer on top of a DB supporting proto messages
//
//go:generate mockgen-wrapper
type Crud interface {
	// Read functions
	Count() (int, error)
	Exists(id string) (bool, error)
	GetKeys() ([]string, error)
	Get(id string) (protocompat.Message, bool, error)
	GetMany(ids []string) (msgs []protocompat.Message, indices []int, err error)
	Walk(func(msg protocompat.Message) error) error
	WalkAllWithID(func(id []byte, msg protocompat.Message) error) error

	// Modifying functions
	Upsert(kv protocompat.Message) error
	UpsertMany(msgs []protocompat.Message) error
	UpsertWithID(id string, msg protocompat.Message) error
	UpsertManyWithIDs(ids []string, msgs []protocompat.Message) error

	Delete(id string) error
	DeleteMany(ids []string) error
}
