package deduper

import (
	"github.com/gogo/protobuf/proto"
)

func serializeDeterministic(msg proto.Message) ([]byte, error) {
	var storage []byte
	if msgWithSize, ok := msg.(interface{ Size() int }); ok {
		storage = make([]byte, 0, msgWithSize.Size())
	}
	buf := proto.NewBuffer(storage)
	buf.SetDeterministic(true)
	if err := buf.Marshal(msg); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
