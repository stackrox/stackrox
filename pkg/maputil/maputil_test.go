package maputil

import (
	"strconv"
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
)

func TestShallowClone(t *testing.T) {
	a := assert.New(t)

	m := map[string]string{
		"a": "1",
		"b": "2",
	}

	cloned := ShallowClone(m)
	cloned["c"] = "3"
	delete(cloned, "a")

	a.Equal(map[string]string{
		"a": "1",
		"b": "2",
	}, m)
	a.Equal(map[string]string{
		"b": "2",
		"c": "3",
	}, cloned)
}

func TestFastRMap(t *testing.T) {
	a := assert.New(t)

	m := NewFastRMap[string, string]()
	m.Set("a", "1")

	a.Equal(map[string]string{"a": "1"}, m.GetMap())
	got, exists := m.Get("a")
	a.True(exists)
	a.Equal(got, "1")

	got, exists = m.Get("b")
	a.False(exists)
	a.Equal(got, "")

	m.Delete("a")
	a.Equal(map[string]string{}, m.GetMap())
}

func TestFastRMapThreadSafe(t *testing.T) {
	a := assert.New(t)
	var wg sync.WaitGroup

	m := NewFastRMap[string, string]()

	numThreads := 1000
	if buildinfo.RaceEnabled {
		numThreads = 100 // race detector chokes on this if we have too many goroutines
	}

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			m.Set("a", "1")
			m.Set(strconv.Itoa(index), "2")
			_ = m.GetMap()
			m.Delete("a")
			got, exists := m.Get(strconv.Itoa(index))
			a.True(exists)
			a.Equal(got, "2")
		}(i)
	}
	wg.Wait()
	currentMap := m.GetMap()
	a.Len(currentMap, numThreads)
	expectedKeys := make([]string, 0, numThreads)
	for i := 0; i < numThreads; i++ {
		expectedKeys = append(expectedKeys, strconv.Itoa(i))
	}
	mapKeys := make([]string, 0, numThreads)
	for k := range currentMap {
		mapKeys = append(mapKeys, k)
	}

	assert.ElementsMatch(t, mapKeys, expectedKeys)
}
