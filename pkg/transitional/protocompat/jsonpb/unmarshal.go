package jsonpb

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"io"
)

func Unmarshal(r io.Reader, m proto.Message) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(data, m)
}
