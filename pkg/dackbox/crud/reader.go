package crud

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/dackbox"
)

// Reader provides the ability to read data as part of a dackbox transaction.
type Reader interface {
	ExistsIn(key []byte, dackTxn *dackbox.Transaction) (bool, error)
	CountIn(prefix []byte, dackTxn *dackbox.Transaction) (int, error)
	ReadAllIn(prefix []byte, dackTxn *dackbox.Transaction) ([]proto.Message, error)
	ReadIn(key []byte, dackTxn *dackbox.Transaction) (proto.Message, error)
}

// NewReader creates a new instance of a Reader.
func NewReader(opts ...ReaderOption) Reader {
	rc := &readerImpl{}
	for _, opt := range opts {
		opt(rc)
	}
	return rc
}

// PartialReader represents reading in part of a messages data from a separate reader.
type PartialReader interface {
	ReadPartialIn(key []byte, msg proto.Message, dackTxn *dackbox.Transaction) (proto.Message, error)
}

// NewPartialReader creates a new instance of a PartialReader.
func NewPartialReader(opts ...PartialReaderOption) PartialReader {
	rc := &partialReaderImpl{}
	for _, opt := range opts {
		opt(rc)
	}
	return rc
}
