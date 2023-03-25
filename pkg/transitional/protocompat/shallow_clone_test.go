package protocompat

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestShallowClone(t *testing.T) {
	msg := &storage.Image{
		Id: "foo",
		Name: &storage.ImageName{
			FullName: "bar",
		},
	}

	clone := ShallowClone(msg)
	assert.NotSame(t, msg, clone)
	assert.Equal(t, msg, clone)

	clone.Id = "baz"
	assert.Equal(t, "foo", msg.GetId())

	clone.Name.FullName = "qux"
	assert.Equal(t, "qux", msg.GetName().GetFullName())
}
