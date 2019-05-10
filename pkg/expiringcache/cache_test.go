package expiringcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type pair struct {
	key, value string
}

func TestExpiringCache(t *testing.T) {
	pair1 := pair{
		key:   "key1",
		value: "value1",
	}
	pair2 := pair{
		key:   "key2",
		value: "value2",
	}
	pair3 := pair{
		key:   "key3",
		value: "value3",
	}
	var pairs = []pair{
		pair1,
		pair2,
	}

	ec := NewExpiringCacheOrPanic(2, 10*time.Second, 0)
	for _, p := range pairs {
		ec.Add(p.key, p.value)
	}
	for _, p := range pairs {
		assert.Equal(t, p.value, ec.Get(p.key))
	}
	values := []string{pair1.value, pair2.value}
	assert.ElementsMatch(t, values, ec.GetAll())

	ec.Add(pair3.key, pair3.value)
	values = []string{pair2.value, pair3.value}
	assert.ElementsMatch(t, values, ec.GetAll())

	assert.False(t, ec.Remove(pair1.key))
	assert.True(t, ec.Remove(pair3.key))

	values = []string{pair2.value}
	assert.ElementsMatch(t, values, ec.GetAll())
}

func TestExpiringCacheExpired(t *testing.T) {
	pair1 := pair{
		key:   "key1",
		value: "value1",
	}
	ec := NewExpiringCacheOrPanic(2, 1*time.Nanosecond, 0)
	ec.Add(pair1.key, pair1.value)
	assert.Nil(t, ec.Get(pair1.key))
}
