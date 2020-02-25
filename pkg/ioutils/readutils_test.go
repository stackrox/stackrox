package ioutils

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadAtMost(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input          string
		inputErr       error
		num            int
		expectedOutput string
		expectedErr    error
	}{
		{
			input:          "foobar",
			num:            3,
			expectedOutput: "foo",
		},
		{
			input:          "foobar",
			num:            6,
			expectedOutput: "foobar",
		},
		{
			input:          "foobar",
			num:            9,
			expectedOutput: "foobar",
		},
		{
			input:          "",
			num:            10,
			expectedOutput: "",
		},
		{
			input:          "foo",
			inputErr:       errors.New("bar"),
			num:            3,
			expectedOutput: "foo",
		},
		{
			input:          "foo",
			inputErr:       errors.New("bar"),
			num:            4,
			expectedOutput: "foo",
			expectedErr:    errors.New("bar"),
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			inputReader := io.MultiReader(strings.NewReader(c.input), ErrorReader(c.inputErr))
			out, err := ReadAtMost(inputReader, c.num)
			assert.Equal(t, c.expectedOutput, string(out))
			assert.Equal(t, c.expectedErr, err)
		})
	}
}
