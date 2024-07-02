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

func TestErrorOnNilUnmarshal(t *testing.T) {
	err := Unmarshal([]byte{}, null)
	assert.NoError(t, err)

	err = Unmarshal(nil, null)
	assert.Equal(t, proto.ErrNil, err)

	err = Unmarshal(nil, null)
	assert.Equal(t, proto.ErrNil, err)

	img := &storage.Image{}
	err = Unmarshal([]byte{}, img)
	assert.NoError(t, err)
}
