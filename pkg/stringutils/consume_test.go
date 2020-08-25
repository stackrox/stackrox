package stringutils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConsumePrefix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input          string
		prefix         string
		expectedOutput string
		expectedResult bool
	}{
		{
			input:          "",
			prefix:         "",
			expectedOutput: "",
			expectedResult: true,
		},
		{
			input:          "foo",
			prefix:         "",
			expectedOutput: "foo",
			expectedResult: true,
		},
		{
			input:          "foobar",
			prefix:         "bar",
			expectedOutput: "foobar",
			expectedResult: false,
		},
		{
			input:          "foobar",
			prefix:         "foo",
			expectedOutput: "bar",
			expectedResult: true,
		},
		{
			input:          "foo",
			prefix:         "foo",
			expectedOutput: "",
			expectedResult: true,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s/%s", c.input, c.prefix), func(t *testing.T) {
			s := c.input
			res := ConsumePrefix(&s, c.prefix)
			assert.Equal(t, c.expectedOutput, s)
			assert.Equal(t, c.expectedResult, res)
		})
	}
}

func TestConsumeSuffix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input          string
		suffix         string
		expectedOutput string
		expectedResult bool
	}{
		{
			input:          "",
			suffix:         "",
			expectedOutput: "",
			expectedResult: true,
		},
		{
			input:          "foo",
			suffix:         "",
			expectedOutput: "foo",
			expectedResult: true,
		},
		{
			input:          "foobar",
			suffix:         "foo",
			expectedOutput: "foobar",
			expectedResult: false,
		},
		{
			input:          "foobar",
			suffix:         "bar",
			expectedOutput: "foo",
			expectedResult: true,
		},
		{
			input:          "foo",
			suffix:         "foo",
			expectedOutput: "",
			expectedResult: true,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s/%s", c.input, c.suffix), func(t *testing.T) {
			s := c.input
			res := ConsumeSuffix(&s, c.suffix)
			assert.Equal(t, c.expectedOutput, s)
			assert.Equal(t, c.expectedResult, res)
		})
	}
}
