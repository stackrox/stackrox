package expiringcache

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/expiringcache/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
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
		pair3,
	}

	mockCtrl := gomock.NewController(t)
	clock := mocks.NewMockClock(mockCtrl)

	ec := NewExpiringCacheWithClock(clock, 10*time.Second)

	// Insert all values one second apart.
	addTime := time.Time{}
	for _, p := range pairs {
		clock.EXPECT().Now().Return(addTime)
		ec.Add(p.key, p.value)

		addTime = addTime.Add(time.Second)
	}

	// Check that at time 0 all values are accessable.
	getTime := time.Time{}
	for _, p := range pairs {
		clock.EXPECT().Now().Return(getTime)
		assert.Equal(t, p.value, ec.Get(p.key).(string))
	}

	// Move forward 11 seconds, and the first element should get pruned but the rest should be available.
	getTime = getTime.Add(11 * time.Second)

	// First is gone.
	clock.EXPECT().Now().Return(getTime)
	assert.Nil(t, ec.Get(pairs[0].key))

	// Other two are still available.
	clock.EXPECT().Now().Return(getTime)
	currentValues := ec.GetAll()
	assert.Equal(t, 2, len(currentValues))
	for i, value := range currentValues {
		assert.Equal(t, pairs[i+1].value, value.(string))
	}

	// Move forward another second, and the second element should drop off.
	getTime = getTime.Add(1 * time.Second)

	// First and second elements are gone.
	clock.EXPECT().Now().Return(getTime)
	assert.Nil(t, ec.Get(pairs[0].key))

	clock.EXPECT().Now().Return(getTime)
	assert.Nil(t, ec.Get(pairs[1].key))

	// Third element is still there.
	clock.EXPECT().Now().Return(getTime)
	currentValues = ec.GetAll()
	assert.Equal(t, 1, len(currentValues))
	for i, value := range currentValues {
		assert.Equal(t, pairs[i+2].value, value.(string))
	}

	mockCtrl.Finish()
}
