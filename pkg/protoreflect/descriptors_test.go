package protoreflect

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestGetMessageDescriptor(t *testing.T) {
	descriptor, err := GetMessageDescriptor(&storage.Node{})
	assert.NoError(t, err)
	assert.Equal(t, "Node", descriptor.GetName())
}

func TestGetEnumDescriptor(t *testing.T) {
	descriptor, err := GetEnumDescriptor(new(storage.Access))
	assert.NoError(t, err)
	assert.Equal(t, "Access", descriptor.GetName())
}
