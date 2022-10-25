package protoutils

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/pkg/transitional/protocompat/proto"
)

// NewWrapper takes in a proto.Message and overrides the String method with jsonpb.Marshal
func NewWrapper(msg proto.Message) *Wrapper {
	return &Wrapper{
		Message: msg,
	}
}

// Wrapper wraps a proto.Message and overrides the String method with jsonpb
type Wrapper struct {
	proto.Message
}

func (w *Wrapper) String() string {
	marshaler := &jsonpb.Marshaler{
		Indent:       "  ",
		EmitDefaults: true,
	}
	if w.Message == nil {
		return "<nil>"
	}
	s, _ := marshaler.MarshalToString(w.Message)
	return s
}
