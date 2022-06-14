package crud

import (
	"github.com/stackrox/stackrox/pkg/dackbox"
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
