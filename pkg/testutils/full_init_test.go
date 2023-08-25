package testutils

import (
	"testing"

	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nestedStruct struct {
	x [3]int
	y []*string
	z map[string]int

	w *testStruct
}

type nestedStruct2 struct {
	x uint
}

type nestedStruct3 struct {
	x complex64
}

type testStruct struct {
	x int
	y string
	z bool

	v *float32
	w []*nestedStruct

	nestedStruct2
	*nestedStruct3
}

func TestFullInit(t *testing.T) {
	var s testStruct
	require.NoError(t, FullInit(&s, SimpleInitializer(), nil))

	expected := testStruct{
		x: 1,
		y: uuid.NewDummy().String(),
		z: true,
		v: &[]float32{1.0}[0],
		w: []*nestedStruct{
			{
				x: [3]int{1, 1, 1},
				y: []*string{
					&[]string{uuid.NewDummy().String()}[0],
				},
				z: map[string]int{
					uuid.NewDummy().String(): 1,
				},
				w: nil,
			},
		},
		nestedStruct2: nestedStruct2{
			x: 1,
		},
		nestedStruct3: &nestedStruct3{
			x: 1.0i,
		},
	}

	assert.Equal(t, expected, s)
}
