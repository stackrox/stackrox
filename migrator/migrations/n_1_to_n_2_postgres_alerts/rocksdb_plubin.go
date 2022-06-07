package n1ton2

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
)

// Walk iterates over all of the objects in the store and applies the closure
func (b *storeImpl) Walk(_ context.Context, fn func(obj *storage.Alert) error) error {
	return b.crud.Walk(func(msg proto.Message) error {
		return fn(msg.(*storage.Alert))
	})
}
