package protoutils

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
)

func TestErrorOnNilMarshal(t *testing.T) {
	_, err := protocompat.Marshal(nil)
	assert.Equal(t, protocompat.ErrNil, err)

	_, err = protocompat.Marshal((*storage.Image)(nil))
	assert.Equal(t, protocompat.ErrNil, err)

	_, err = protocompat.Marshal(&storage.Image{})
	assert.NoError(t, err)
}

func TestErrorOnNilUnmarshal(t *testing.T) {
	err := protocompat.Unmarshal(nil, (*storage.Image)(nil))
	assert.Equal(t, protocompat.ErrNil, err)

	var img *storage.Image
	err = protocompat.Unmarshal(nil, img)
	assert.Equal(t, protocompat.ErrNil, err)

	img = &storage.Image{}
	err = protocompat.Unmarshal([]byte{}, img)
	assert.NoError(t, err)
}
