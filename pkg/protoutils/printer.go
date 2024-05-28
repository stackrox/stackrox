package protoutils

import (
	"github.com/stackrox/rox/pkg/protocompat"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/protoadapt"
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
		Indent:            "  ",
		EmitDefaultValues: true,
	}
	if w.Message == nil {
		return "<nil>"
	}
	m2 := protoadapt.MessageV2Of(w.Message)
	s, _ := marshaler.Marshal(m2)
	return string(s)
}
