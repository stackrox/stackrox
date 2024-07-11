package maputil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmpMax(t *testing.T) {
	m := NewMaxMap[string, int]()

	m.Store("a", -1)
	v, ok := m.Load("a")
	assert.Equal(t, -1, v)
	assert.True(t, ok)

	m.Store("a", 5)
	v, _ = m.Load("a")
	assert.Equal(t, 5, v)

	m.Store("a", 4)
	v, _ = m.Load("a")
	assert.Equal(t, 5, v)
	m.Store("b", 3)
	v, _ = m.Load("b")
	assert.Equal(t, 3, v)
}
