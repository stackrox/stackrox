package protoutils

import (
	"github.com/stackrox/rox/pkg/protocompat"
	"google.golang.org/protobuf/encoding/protojson"
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
	marshaler := &protojson.MarshalOptions{
		Indent:          "  ",
		EmitUnpopulated: true,
	}
	if w.Message == nil {
		return "<nil>"
	}
	s, _ := marshaler.Marshal(w.Message)
	return string(s)
}
