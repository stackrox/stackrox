package jsonpb

import (
	"io"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Unmarshal allows unmarshaling from a reader in protojson format.
func Unmarshal(r io.Reader, m proto.Message) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(data, m)
}
