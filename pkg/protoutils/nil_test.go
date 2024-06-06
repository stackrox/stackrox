package protoutils

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
)

var null = (*storage.Image)(nil)

func TestErrorOnNilMarshal(t *testing.T) {
	_, err := protocompat.Marshal(null)
	assert.Equal(t, proto.ErrNil, err)

	var msg *storage.Image
	_, err = protocompat.Marshal(msg)
	assert.Equal(t, proto.ErrNil, err)

	_, err = protocompat.Marshal(&storage.Image{})
	assert.NoError(t, err)
}

func TestErrorOnNilUnmarshal(t *testing.T) {
	err := protocompat.Unmarshal([]byte{}, null)
	assert.NoError(t, err)

	err = protocompat.Unmarshal(nil, null)
	assert.Equal(t, proto.ErrNil, err)

	err = protocompat.Unmarshal(nil, null)
	assert.Equal(t, proto.ErrNil, err)

	img := &storage.Image{}
	err = protocompat.Unmarshal([]byte{}, img)
	assert.NoError(t, err)
}
