package concurrency

import (
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stretchr/testify/assert"
)

func TestTransparentMutexHappyPath(t *testing.T) {
	lock := TransparentMutex{}
	assert.True(t, lock.MaybeLock())
	assert.False(t, lock.MaybeLock())
	lock.Unlock()
	assert.True(t, lock.MaybeLock())
}

func TestTransparentMutexConcurrently(t *testing.T) {
	lock := TransparentMutex{}

	var successCount int32
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(time.Duration(rand.Int()%10) * time.Millisecond)
			succeeded := lock.MaybeLock()
			if succeeded {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), successCount)
}

func TestTransparentMutexIsResilientToRaces(t *testing.T) {
	lock := TransparentMutex{}

	var successCount int32
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(time.Duration(rand.Int()%100) * time.Millisecond)
			succeeded := lock.MaybeLock()
			if succeeded {
				atomic.AddInt32(&successCount, 1)
				lock.Unlock()
			}
		}()
	}
	wg.Wait()

	// We expect at least one success, but can't be guaranteed more than that.
	// The purpose of this test is really to make sure there are no race conditions.
	assert.True(t, successCount >= int32(1))
}
