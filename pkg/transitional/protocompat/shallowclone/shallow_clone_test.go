package shallowclone

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestShallowClone(t *testing.T) {
	img := &storage.Image{
		Id: "foo",
		Name: &storage.ImageName{
			FullName: "docker.io/library/nginx:1.23",
		},
	}

	imgShallowClone := UnsafeShallowClone(img)
	assert.NotSame(t, img, imgShallowClone)
	assert.Same(t, img.GetName(), imgShallowClone.GetName())

	imgShallowClone.Id = "bar"
	assert.NotEqual(t, img.GetId(), imgShallowClone.GetId())

	imgShallowClone.Name.FullName = "docker.io/library/nginx:1.24"
	assert.Equal(t, img.GetName().GetFullName(), imgShallowClone.GetName().GetFullName())
}
