package n{{.Migration.MigrateSequence}}ton{{add .Migration.MigrateSequence 1}}
import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
)

func alloc() proto.Message {
	return &{{.Type}}{}
}

func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(*{{.Type}}).GetId())
}

// Walk iterates over all of the objects in the store and applies the closure
func (b *storeImpl) Walk(_ context.Context, fn func(obj *{{.Type}}) error) error {
	return b.crud.Walk(func(msg proto.Message) error {
		return fn(msg.(*{{.Type}}))
	})
}
