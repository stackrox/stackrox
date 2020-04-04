package sliceutils

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type randomTestStruct struct {
	blah string
}

func TestMap(t *testing.T) {
	testCases := []struct {
		desc        string
		slice       interface{}
		mapFunc     interface{}
		shouldPanic bool
		expectedOut interface{}
	}{
		{
			"simple func on string",
			[]string{"1", "2"},
			func(s string) string {
				return s + s
			},
			false,
			[]string{"11", "22"},
		},
		{
			"same func, but with ptr",
			[]string{"1", "2"},
			func(s *string) string {
				return *s + *s
			},
			false,
			[]string{"11", "22"},
		},
		{
			"func of int to string array",
			[]int{1, 2},
			func(val int) []string {
				out := make([]string, 0, val)
				for i := 0; i < val; i++ {
					out = append(out, strconv.Itoa(i))
				}
				return out
			},
			false,
			[][]string{{"0"}, {"0", "1"}},
		},
		{
			"extract element from struct",
			[]randomTestStruct{{"1"}, {"2"}},
			func(s randomTestStruct) string {
				return s.blah
			},
			false,
			[]string{"1", "2"},
		},
		{
			"extract element from struct with pointer",
			[]randomTestStruct{{"1"}, {"2"}},
			func(s *randomTestStruct) string {
				return s.blah
			},
			false,
			[]string{"1", "2"},
		},
		{
			"extract element from struct with pointer to pointer, should panic",
			[]randomTestStruct{{"1"}, {"2"}},
			func(s **randomTestStruct) string {
				return (*s).blah
			},
			true,
			[]string{"1", "2"},
		},
		{
			"extract element from struct with pointer to pointer",
			[]*randomTestStruct{{"1"}, {"2"}},
			func(s **randomTestStruct) string {
				return (*s).blah
			},
			false,
			[]string{"1", "2"},
		},
	}

	for _, testCase := range testCases {
		c := testCase
		t.Run(c.desc, func(t *testing.T) {
			if c.shouldPanic {
				assert.Panics(t, func() {
					Map(c.slice, c.mapFunc)
				})
				return
			}
			out := Map(c.slice, c.mapFunc)
			require.Equal(t, c.expectedOut, out)
		})
	}
}
