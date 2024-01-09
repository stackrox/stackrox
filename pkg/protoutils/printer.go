package protoutils

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/pkg/protocompat"
)

// NewWrapper takes in a protocompat.Message and overrides the String method with jsonpb.Marshal
func NewWrapper(msg protocompat.Message) *Wrapper {
	return &Wrapper{
		Message: msg,
	}
}

// Wrapper wraps a protocompat.Message and overrides the String method with jsonpb
type Wrapper struct {
	protocompat.Message
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
