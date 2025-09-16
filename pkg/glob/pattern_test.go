package glob

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPattern(t *testing.T) {
	p := Pattern("*value*")

	_, cached := globCache.Load(p)
	assert.False(t, cached)

	assert.True(t, p.Match("value"))

	_, cached = globCache.Load(p)
	assert.True(t, cached)

	assert.True(t, p.Match("some value"))
}
