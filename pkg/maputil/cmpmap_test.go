package maputil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaximap_Get(t *testing.T) {
	m := NewMaxMap[string, int]()
	v, ok := m.Get("a")
	assert.Equal(t, 0, v)
	assert.False(t, ok)

	m.Add("a", 1)
	v, ok = m.Get("a")
	assert.Equal(t, 1, v)
	assert.True(t, ok)
}

func TestMaximap_Add(t *testing.T) {
	m := NewMaxMap[string, int]()
	m.Add("a", 1)

	m.Add("a", 5)
	assert.Equal(t, 5, m.data["a"])

	m.Add("a", 4)
	assert.Equal(t, 5, m.data["a"])
	m.Add("b", 3)
	assert.Equal(t, 3, m.data["b"])
}

func TestMaximap_Reset(t *testing.T) {
	m := NewMaxMap[string, int]()
	m.Add("a", 1)
	prev := m.Reset()
	assert.Equal(t, 1, prev["a"])
	assert.Nil(t, m.data)
}
