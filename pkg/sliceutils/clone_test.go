package sliceutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test2DSliceClone(t *testing.T) {
	a := assert.New(t)

	var slice2D [][]byte
	a.Nil(ShallowClone2DSlice(slice2D))

	slice2D = make([][]byte, 0)
	a.NotNil(ShallowClone2DSlice(slice2D))
	a.Equal(ShallowClone2DSlice(slice2D), [][]byte{})

	slice1 := []byte{'a', 'b'}
	slice2 := []byte{'c', 'd'}
	slice2D = [][]byte{slice1, slice2}
	cloned := ShallowClone2DSlice(slice2D)
	a.Len(slice2D, 2)
	a.Equal(cloned[0], slice1)
	a.Equal(cloned[1], slice2)

	slice1[0] = 'f'
	a.NotEqual(cloned[0], slice1)
}
