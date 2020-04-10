package protoutils

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestErrorOnNilMarshal(t *testing.T) {
	_, err := proto.Marshal(nil)
	assert.Equal(t, proto.ErrNil, err)

	_, err = proto.Marshal((*storage.Image)(nil))
	assert.Equal(t, proto.ErrNil, err)

	var img *storage.Image
	var msg proto.Message = img
	_, err = proto.Marshal(msg)
	assert.Equal(t, proto.ErrNil, err)

	_, err = proto.Marshal(&storage.Image{})
	assert.NoError(t, err)
}

func TestErrorOnNilUnmarshal(t *testing.T) {
	err := proto.Unmarshal(nil, nil)
	assert.Equal(t, proto.ErrNil, err)

	err = proto.Unmarshal(nil, (*storage.Image)(nil))
	assert.Equal(t, proto.ErrNil, err)

	var img *storage.Image
	err = proto.Unmarshal(nil, img)
	assert.Equal(t, proto.ErrNil, err)

	img = &storage.Image{}
	err = proto.Unmarshal([]byte{}, img)
	assert.NoError(t, err)
}
