package protocompat

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

var null = (*storage.Image)(nil)

func TestErrorOnNilMarshal(t *testing.T) {
	_, err := Marshal(null)
	assert.Equal(t, proto.ErrNil, err)

	var msg *storage.Image
	_, err = Marshal(msg)
	assert.Equal(t, proto.ErrNil, err)

	_, err = Marshal(&storage.Image{})
	assert.NoError(t, err)
}
