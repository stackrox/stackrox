package sliceutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReversed(t *testing.T) {
	in := []string{"foo", "bar", "baz"}
	out := Reversed(in)
	assert.Equal(t, []string{"baz", "bar", "foo"}, out)
	assert.Equal(t, []string{"foo", "bar", "baz"}, in)
}
