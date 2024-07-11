package sliceutils

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

type randomTestStruct struct {
	blah string
}

func testMap[T, U any](t *testing.T, desc string, slice []T, mapFunc func(T) U, expectedOut []U) {
	t.Run(desc, func(t *testing.T) {
		out := Map(slice, mapFunc)
		require.Equal(t, expectedOut, out)
	})
}

func TestMap(t *testing.T) {
	testMap(t,
		"simple func on string",
		[]string{"1", "2"},
		func(s string) string {
			return s + s
		},
		[]string{"11", "22"},
	)
	testMap(t,
		"func of int to string array",
		[]int{1, 2},
		func(val int) []string {
			out := make([]string, 0, val)
			for i := 0; i < val; i++ {
				out = append(out, strconv.Itoa(i))
			}
			return out
		},
		[][]string{{"0"}, {"0", "1"}},
	)
	testMap(t,
		"extract element from struct",
		[]randomTestStruct{{"1"}, {"2"}},
		func(s randomTestStruct) string {
			return s.blah
		},
		[]string{"1", "2"},
	)
}
