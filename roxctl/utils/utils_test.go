package utils

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_wordsAndDelimeters(t *testing.T) {
	cases := map[string][]string{
		"\nText with \nmultiple\n lines.": {"\n", "Text", " ", "with", " ", "\n", "multiple", "\n", " ", "lines."},
		"":                                {},
		" ":                               {" "},
		" \n ":                            {" ", "\n", " "},
		" \n\t":                           {" ", "\n", "\t"},
		"word":                            {"word"},
	}
	for text, c := range cases {
		sc := bufio.NewScanner(strings.NewReader(text))
		result := []string{}
		sc.Split(wordsAndDelimeters)
		for sc.Scan() {
			result = append(result, sc.Text())
		}
		assert.Equal(t, c, result)
	}
}

func Test_pop(t *testing.T) {
	cases := []struct {
		arr           indents
		expectedValue int
		expectedArr   []int
	}{
		{[]int{1, 2, 3}, 1, []int{2, 3}},
		{[]int{1}, 1, []int{1}},
		{[]int{}, 0, []int{}},
	}
	for _, c := range cases {
		value := c.arr.pop()
		assert.Equal(t, c.expectedValue, value)
		assert.Equal(t, c.expectedArr, []int(c.arr))
	}
}
