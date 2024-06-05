package protocompat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmpty(t *testing.T) {
	refEmpty := &Empty{}

	assert.True(t, Equal(refEmpty, ProtoEmpty()))
}
