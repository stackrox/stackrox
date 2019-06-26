package expiringcache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMappedQueue(t *testing.T) {
	key1 := "k1"
	value1 := "v1"

	key2 := "k2"
	value2 := "v2"

	key3 := "k3"
	value3 := "v3"

	mq := newMappedQueue(2)

	// Add k/v 1
	mq.push(key1, value1)

	// From should hold k/v 1
	actualKey, actualValue := mq.front()
	assert.Equal(t, actualKey.(string), key1)
	assert.Equal(t, actualValue.(string), value1)

	// Add k/v 2
	mq.push(key2, value2)

	// From should hold k/v 1
	actualKey, actualValue = mq.front()
	assert.Equal(t, actualKey.(string), key1)
	assert.Equal(t, actualValue.(string), value1)

	// Add k/v 3
	mq.push(key3, value3)

	// From should hold k/v 2 since 1 was pushed out by max size
	actualKey, actualValue = mq.front()
	assert.Equal(t, actualKey.(string), key2)
	assert.Equal(t, actualValue.(string), value2)

	// Should be able to fetch k3
	actualValue = mq.get(key3)
	assert.Equal(t, actualValue.(string), value3)

	// getAllValues should return 3 and 2.
	actualValues := mq.getAllValues()
	assert.Equal(t, len(actualValues), 2)

	// Remove 2
	mq.remove(key2)
	actualValues = mq.getAllValues()
	assert.Equal(t, len(actualValues), 1)
	assert.Equal(t, actualValues[0].(string), value3)

	// removeAll
	mq.removeAll()
	actualValues = mq.getAllValues()
	assert.Equal(t, len(actualValues), 0)
}
