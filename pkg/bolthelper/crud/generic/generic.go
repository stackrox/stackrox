package generic

import (
	"github.com/stackrox/rox/pkg/bolthelper"
)

// Key is an alias for a byte slice.
type Key = []byte

// KeyPath is an alias for a slice of Keys.
type KeyPath = []Key

// Entry references an entry in the DB.
type Entry struct {
	Nesting KeyPath
	Value   interface{}
}

// Crud provides a simple crud layer on top of bolt DB supporting messages of any type stored in a nested structure.
type Crud interface {
	Read(firstKey Key, restKeys ...Key) (interface{}, error)
	ReadBatch(keyPaths ...KeyPath) ([]interface{}, []int, error)
	ReadAll(maxDepth int, keyPathPrefix ...Key) ([]Entry, error)
	CountLeaves(maxDepth int, keyPathPrefix ...Key) (int, error)

	Create(x interface{}, nesting ...Key) error
	CreateBatch(entries []Entry, nestingPrefix ...Key) error
	Update(x interface{}, nesting ...Key) (uint64, uint64, error)
	UpdateBatch(entries []Entry, nestingPrefix ...Key) (uint64, uint64, error)
	Upsert(x interface{}, nesting ...Key) (uint64, uint64, error)
	UpsertBatch(entries []Entry, nestingPrefix ...Key) (uint64, uint64, error)

	Delete(firstKey Key, restKeys ...Key) (uint64, uint64, error)
	DeleteBatch(keyPaths ...KeyPath) (uint64, uint64, error)
}

// DeserializeFunc is the function converting a byte slice into an element (or returning an error).
type DeserializeFunc func(k, v []byte) (interface{}, error)

// SerializeFunc is the function converting an element into a byte slice (or returning an error).
type SerializeFunc func(x interface{}) (k []byte, v []byte, err error)

// NewCrud returns a new Crud instance for the given bucket reference.
func NewCrud(bucketRef bolthelper.BucketRef,
	deserializeFunc DeserializeFunc,
	serializeFunc SerializeFunc) Crud {
	return &crudImpl{
		bucketRef:       bucketRef,
		deserializeFunc: deserializeFunc,
		serializeFunc:   serializeFunc,
	}
}
