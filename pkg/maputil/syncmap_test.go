package maputil

import (
	"testing"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
)

func TestSyncMapStoreLoad(t *testing.T) {
	m := NewSyncMap[string, int]()
	m.Store("a", 1)
	m.Store("a", 2)
	v, ok := m.Load("a")
	assert.True(t, ok)
	assert.Equal(t, 2, v)
}

func TestSyncMap(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(3)
	m := NewSyncMap[string, int]()
	go func() {
		m.Store("a", 1)
		wg.Done()
	}()
	go func() {
		_, _ = m.Load("a")
		wg.Done()
	}()
	go func() {
		m.Access(func(m *map[string]int) { (*m)["b"] = 2 })
		wg.Done()
	}()
	wg.Wait()
	v, ok := m.Load("a")
	assert.True(t, ok)
	assert.Equal(t, 1, v)
	v, ok = m.Load("b")
	assert.True(t, ok)
	assert.Equal(t, 2, v)
}

func TestAccess(t *testing.T) {
	m := NewCmpMap[string, int](nil)
	m.Store("a", 1)
	m.Store("a", 2)
	m.RAccess(func(m map[string]int) {
		assert.Equal(t, 2, m["a"])
	})
	v, ok := m.Load("a")
	assert.True(t, ok)
	assert.Equal(t, 2, v)

	m.Access(func(m *map[string]int) {
		*m = map[string]int{"c": 3}
	})
	_, ok = m.Load("a")
	assert.False(t, ok)
	v, ok = m.Load("c")
	assert.True(t, ok)
	assert.Equal(t, 3, v)
}
