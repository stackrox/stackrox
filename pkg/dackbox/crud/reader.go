package crud

import (
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/protocompat"
)

// Reader provides the ability to read data as part of a dackbox transaction.
type Reader interface {
	ExistsIn(key []byte, dackTxn *dackbox.Transaction) (bool, error)
	CountIn(prefix []byte, dackTxn *dackbox.Transaction) (int, error)
	ReadAllIn(prefix []byte, dackTxn *dackbox.Transaction) ([]protocompat.Message, error)
	ReadKeysIn(prefix []byte, dackTxn *dackbox.Transaction) ([][]byte, error)
	ReadIn(key []byte, dackTxn *dackbox.Transaction) (protocompat.Message, error)
}

// NewReader creates a new instance of a Reader.
func NewReader(opts ...ReaderOption) Reader {
	rc := &readerImpl{}
	for _, opt := range opts {
		opt(rc)
	}
	return rc
}
