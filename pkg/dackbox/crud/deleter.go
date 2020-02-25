package crud

import (
	"github.com/stackrox/rox/pkg/dackbox"
)

// Deleter provides the ability to delete as part of a dackbox transaction.
type Deleter interface {
	DeleteIn(key []byte, dackTxn *dackbox.Transaction) error
}

// NewDeleter creates a new instance of a deleter.
func NewDeleter(opts ...DeleterOption) Deleter {
	dc := &deleterImpl{}
	for _, opt := range opts {
		opt(dc)
	}
	return dc
}

// PartialDeleter represents deleting connected keys as part of a parent key.
type PartialDeleter interface {
	DeletePartialsIn(keys [][]byte, dackTxn *dackbox.Transaction) error
}

// NewPartialDeleter creates a new instance of a PartialDeleter.
func NewPartialDeleter(opts ...PartialDeleterOption) PartialDeleter {
	uc := &partialDeleterImpl{}
	for _, opt := range opts {
		opt(uc)
	}
	return uc
}
