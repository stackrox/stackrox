package glob

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPattern(t *testing.T) {
	p := Pattern("*value*")
	assert.True(t, p.Match("value"))
	assert.True(t, p.Match("some value"))
	assert.True(t, p.Match("value here"))
	assert.False(t, p.Match("no match"))
}

func TestPattern_Exact(t *testing.T) {
	p := Pattern("exact")
	assert.True(t, p.Match("exact"))
	assert.False(t, p.Match("not exact"))
}

func TestPattern_Empty(t *testing.T) {
	p := Pattern("")
	assert.True(t, p.Match("anything"))
	assert.True(t, p.Match(""))
}

func TestPattern_Nil(t *testing.T) {
	var p *Pattern
	assert.True(t, p.Match("anything"))
}

func TestPattern_QuestionMark(t *testing.T) {
	p := Pattern("val?e")
	assert.True(t, p.Match("value"))
	assert.True(t, p.Match("valXe"))
	assert.False(t, p.Match("val"))
}

func TestPattern_Compile(t *testing.T) {
	p := Pattern("valid*")
	assert.NoError(t, p.Compile())

	bad := Pattern("[invalid")
	assert.Error(t, bad.Compile())
}
