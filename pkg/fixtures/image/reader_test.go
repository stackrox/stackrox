package image

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReader(t *testing.T) {
	images, err := GetTestImages(t)
	assert.NoError(t, err)
	assert.Len(t, images, 5)
}
