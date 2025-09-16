package protocompat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestEmpty(t *testing.T) {
	refEmpty := &Empty{}

	assert.True(t, proto.Equal(refEmpty, ProtoEmpty()))
}
