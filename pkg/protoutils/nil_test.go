package protoutils

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
)

func TestErrorOnNilMarshal(t *testing.T) {
	_, err := protocompat.ProtoMarshal(nil)
	assert.Equal(t, proto.ErrNil, err)

	_, err = protocompat.Marshal((*storage.Image)(nil))
	assert.Equal(t, proto.ErrNil, err)

	var img *storage.Image
	var msg protocompat.Message = img
	_, err = protocompat.ProtoMarshal(msg)
	assert.Equal(t, proto.ErrNil, err)

	_, err = protocompat.Marshal(&storage.Image{})
	assert.NoError(t, err)
}

func TestErrorOnNilUnmarshal(t *testing.T) {
	err := protocompat.Unmarshal(nil, nil)
	assert.Equal(t, proto.ErrNil, err)

	err = protocompat.Unmarshal(nil, (*storage.Image)(nil))
	assert.Equal(t, proto.ErrNil, err)

	var img *storage.Image
	err = protocompat.Unmarshal(nil, img)
	assert.Equal(t, proto.ErrNil, err)

	img = &storage.Image{}
	err = protocompat.Unmarshal([]byte{}, img)
	assert.NoError(t, err)
}
