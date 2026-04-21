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

	ec := NewExpiringCacheWithClock[string, string](clock, 10*time.Second)

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
		v, ok := ec.Get(p.key)
		assert.True(t, ok)
		assert.Equal(t, p.value, v)
	}

	// Move forward 11 seconds, and the first element should get pruned but the rest should be available.
	getTime = getTime.Add(11 * time.Second)

	// First is gone.
	clock.EXPECT().Now().Return(getTime)
	v, ok := ec.Get(pairs[0].key)
	assert.Empty(t, v)
	assert.False(t, ok)

	// Other two are still available.
	clock.EXPECT().Now().Return(getTime)
	currentValues := ec.GetAll()
	assert.Equal(t, 2, len(currentValues))
	for i, value := range currentValues {
		assert.Equal(t, pairs[i+1].value, value)
	}

	// Move forward another second, and the second element should drop off.
	getTime = getTime.Add(1 * time.Second)

	// First and second elements are gone.
	clock.EXPECT().Now().Return(getTime)
	v, ok = ec.Get(pairs[0].key)
	assert.Empty(t, v)
	assert.False(t, ok)

	clock.EXPECT().Now().Return(getTime)
	v, ok = ec.Get(pairs[1].key)
	assert.Empty(t, v)
	assert.False(t, ok)

	// Third element is still there.
	clock.EXPECT().Now().Return(getTime)
	currentValues = ec.GetAll()
	assert.Equal(t, 1, len(currentValues))
	for i, value := range currentValues {
		assert.Equal(t, pairs[i+2].value, value)
	}

	mockCtrl.Finish()
}

func TestTouch(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	clock := mocks.NewMockClock(mockCtrl)

	ec := NewExpiringCacheWithClock[string, string](clock, 10*time.Second)

	t0 := time.Time{}

	// Add key1 at t=0. (addNoLock calls Now() once)
	clock.EXPECT().Now().Return(t0)
	ec.Add("key1", "value1")

	// Touch non-existent key returns false. (cleanNoLock calls Now() once, no addNoLock)
	clock.EXPECT().Now().Return(t0)
	assert.False(t, ec.Touch("no-such-key"))

	// Touch existing key returns true. (cleanNoLock + addNoLock = 2 Now() calls)
	clock.EXPECT().Now().Return(t0).Times(2)
	assert.True(t, ec.Touch("key1"))

	// Verify value is preserved.
	clock.EXPECT().Now().Return(t0)
	v, ok := ec.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", v)

	// Advance to t=9s — Touch to reset TTL. (2 Now() calls)
	t9 := t0.Add(9 * time.Second)
	clock.EXPECT().Now().Return(t9).Times(2)
	assert.True(t, ec.Touch("key1"))

	// At t=15s (>10s from original add, but <10s from touch at t=9s) the
	// entry should still be alive because Touch reset the TTL.
	t15 := t0.Add(15 * time.Second)
	clock.EXPECT().Now().Return(t15)
	v, ok = ec.Get("key1")
	assert.True(t, ok, "entry should survive past original TTL after Touch")
	assert.Equal(t, "value1", v)

	// At t=20s (>10s from touch at t=9s) the entry should be expired.
	t20 := t0.Add(20 * time.Second)
	clock.EXPECT().Now().Return(t20)
	v, ok = ec.Get("key1")
	assert.False(t, ok, "entry should expire after TTL from last Touch")
	assert.Empty(t, v)

	// Touch on expired key returns false. (1 Now() call, no addNoLock)
	clock.EXPECT().Now().Return(t20)
	assert.False(t, ec.Touch("key1"))

	mockCtrl.Finish()
}
