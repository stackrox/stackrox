package concurrency

import (
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestKeyedMutexSameKey(t *testing.T) {
	a := assert.New(t)

	km := NewKeyedMutex(2)
	const testKey = "test"

	km.Lock(testKey)

	signal := NewSignal()
	go func() {
		km.Lock(testKey)
		defer km.Unlock(testKey)
		a.True(signal.Signal())
	}()

	a.False(signal.IsDone())
	km.Unlock(testKey)
	a.True(WaitWithTimeout(signal.WaitC(), 5*time.Second))
}

func TestKeyedMutexDifferentKeys(t *testing.T) {
	a := assert.New(t)

	km := NewKeyedMutex(100000)

	var counter uint32
	for i := 0; i < 10; i++ {
		go func(key string) {
			km.Lock(key)
			atomic.AddUint32(&counter, 1)
		}(strconv.FormatInt(int64(i), 10))
	}

	time.Sleep(500 * time.Millisecond)

	// At least 8 of the goroutines should have hit the counter.
	// The probability of multiple collisions is negligible given
	// our poolSize (and this is deterministic unless someone changes
	// the code, so no chance of random flakes).
	counterVal := atomic.LoadUint32(&counter)
	a.True(counterVal >= 8, "%d was smaller than expected", counterVal)
}
