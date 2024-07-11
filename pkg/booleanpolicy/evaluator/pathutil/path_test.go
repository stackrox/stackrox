package pathutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testObj struct {
	A string
	B nestedObj
	C []nestedSliceObj
	//lint:ignore U1000 This is actually used through reflect.
	unexported string
}

type nestedObj struct {
	Val string
}

type nestedPtrObj struct {
	Val string
}

type nestedSliceObj struct {
	Val string
	Ptr *nestedPtrObj
}

func TestTraverse(t *testing.T) {
	obj := testObj{
		A: "A",
		B: nestedObj{
			Val: "B",
		},
		C: []nestedSliceObj{
			{Val: "C0", Ptr: &nestedPtrObj{Val: "Ptr0"}},
			{Val: "C1"},
		},
	}
	for _, testCase := range []struct {
		path          *Path
		expectedValue interface{}
		expectedErr   bool
	}{
		{
			path:        PathFromSteps(t, "NOTTHERR"),
			expectedErr: true,
		},
		{
			path:        PathFromSteps(t, "unexported"),
			expectedErr: true,
		},
		{
			path:          PathFromSteps(t, "A"),
			expectedValue: "A",
		},
		{
			path:        PathFromSteps(t, "B", "Blah"),
			expectedErr: true,
		},
		{
			path:          PathFromSteps(t, "B", "Val"),
			expectedValue: "B",
		},
		{
			path:          PathFromSteps(t, "B", "Val"),
			expectedValue: "B",
		},
		{
			path:          PathFromSteps(t, "C", 0, "Val"),
			expectedValue: "C0",
		},
		{
			path:          PathFromSteps(t, "C", 0, "Ptr", "Val"),
			expectedValue: "Ptr0",
		},
		{
			path:          PathFromSteps(t, "C", 1, "Val"),
			expectedValue: "C1",
		},
		// Pointer is nil
		{
			path:        PathFromSteps(t, "C", 1, "Ptr", "Val"),
			expectedErr: true,
		},
		// Index out of bounds
		{
			path:        PathFromSteps(t, "C", 2, "Val"),
			expectedErr: true,
		},
	} {
		c := testCase
		t.Run(fmt.Sprintf("%+v", c), func(t *testing.T) {
			got, err := RetrieveValueAtPath(obj, c.path)
			if c.expectedErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.expectedValue, got)
		})
	}
}
