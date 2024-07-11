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

	mq := newMappedQueue()

	// Add k/v 1
	mq.push(key1, value1)

	// Front should hold k/v 1
	actualKey, actualValue := mq.front()
	assert.Equal(t, key1, actualKey.(string))
	assert.Equal(t, value1, actualValue.(string))

	// overwrite key1 with value2
	mq.push(key1, value2)
	actualKey, actualValue = mq.front()
	assert.Equal(t, key1, actualKey.(string))
	assert.Equal(t, value2, actualValue.(string))

	// reset back to value1
	mq.push(key1, value1)

	// Add k/v 2
	mq.push(key2, value2)

	// Front should hold k/v 1
	actualKey, actualValue = mq.front()
	assert.Equal(t, key1, actualKey.(string))
	assert.Equal(t, value1, actualValue.(string))

	// Add k/v 3
	mq.push(key3, value3)

	// Should be able to fetch k3
	actualValue = mq.get(key3)
	assert.Equal(t, value3, actualValue.(string))

	// getAllValues should return 1,2,3
	actualValues := mq.getAllValues()
	assert.Equal(t, 3, len(actualValues))

	// Should remove k/v 1
	mq.pop()
	actualValue = mq.get(key1)
	assert.Nil(t, actualValue)

	actualValues = mq.getAllValues()
	assert.Equal(t, 2, len(actualValues))

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
