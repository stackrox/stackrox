package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOneOrMore(t *testing.T) {
	assert.Equal(t, 1, OneOrMore(-2).Count())
	assert.Equal(t, 1, OneOrMore(-1).Count())
	assert.Equal(t, 1, OneOrMore(0).Count())
	assert.Equal(t, 1, OneOrMore(1).Count())
	assert.Equal(t, 2, OneOrMore(2).Count())
}
