package sliceutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type myType int

func TestFind(t *testing.T) {
	t.Parallel()

	slice := []myType{1, 3, 7}
	assert.Equal(t, -1, Find(slice, 3))
	assert.Equal(t, 1, Find(slice, myType(3)))
}
