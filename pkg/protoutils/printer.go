package protoutils

import (
	"github.com/stackrox/rox/pkg/transitional/protocompat/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

// NewWrapper takes in a proto.Message and overrides the String method with protojson.Format
func NewWrapper(msg proto.Message) *Wrapper {
	return &Wrapper{
		Message: msg,
	}
}

// Wrapper wraps a proto.Message and overrides the String method with protojson
type Wrapper struct {
	proto.Message
}

func (w *Wrapper) String() string {
	marshaler := &protojson.MarshalOptions{
		Indent:          "  ",
		EmitUnpopulated: true,
	}
	if w.Message == nil {
		return "<nil>"
	}
	return marshaler.Format(w.Message)
}
