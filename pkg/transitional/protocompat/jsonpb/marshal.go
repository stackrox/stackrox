package jsonpb

import (
	"io"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Marshal allows marshaling to a writer in protojson format.
func Marshal(w io.Writer, m proto.Message) error {
	bytes, err := protojson.Marshal(m)
	if err != nil {
		return err
	}
	if _, err := w.Write(bytes); err != nil {
		return err
	}
	return nil
}
