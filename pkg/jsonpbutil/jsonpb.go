package jsonpbutil

import (
	"fmt"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

type MarshalStringWrapper struct {
	marshaler *jsonpb.Marshaler
	data      []string
}

func NewMarshalStringWrapper(marshaler *jsonpb.Marshaler) *MarshalStringWrapper {
	return &MarshalStringWrapper{
		marshaler: marshaler,
	}
}

func (m *MarshalStringWrapper) Marshal(msg proto.Message) error {
	data, err := m.marshaler.MarshalToString(msg)
	if err != nil {
		return err
	}
	m.data = append(m.data, data)
	return nil
}

func (m *MarshalStringWrapper) String() string {
	return fmt.Sprintf("[%s]", strings.Join(m.data, ","))
}
