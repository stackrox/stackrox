package crud

import (
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/transitional/protocompat/proto"
)

// Upserter provides the ability to upsert as part of a dackbox transaction.
type Upserter interface {
	UpsertIn(parentKey []byte, msg proto.Message, dackTxn *dackbox.Transaction) error
}

// NewUpserter creates a new instance of an Upserter.
func NewUpserter(opts ...UpserterOption) Upserter {
	uc := &upserterImpl{}
	for _, opt := range opts {
		opt(uc)
	}
	return uc
}
