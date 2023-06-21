package uniqueue

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

var (
	queueSize      = 5
	frontValue     = 0
	backValue      = 3
	numsToPush     = []int{1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2}
	expectedSlice  = []int{frontValue, 1, 2, backValue}
	expectedMap    = map[int]struct{}{frontValue: {}, 1: {}, 2: {}, backValue: {}}
	defaultTimeout = 1 * time.Second
)

func TestAdd(t *testing.T) {
	q := NewUniQueue[int](queueSize)
	assert.NoError(t, q.Start())
	pushElement(t, q)
	var queueContent []int
	for i := 0; i < len(expectedSlice); i++ {
		n := <-q.PopC()
		queueContent = append(queueContent, n)
	}

	assert.Equal(t, expectedSlice, queueContent)
	assert.Len(t, q.frontChannel, 0)
	q.Stop()
}

func TestCallStartTwice(t *testing.T) {
	q := NewUniQueue[int](queueSize)
	assert.NoError(t, q.Start())
	assert.Error(t, q.Start())
}

func TestStop(t *testing.T) {
	q := NewUniQueue[int](queueSize)
	assert.NoError(t, q.Start())
	q.Stop()
	_, ok := <-q.backChannel
	assert.False(t, ok, "The backChannel should be closed after stopping")
	_, ok = <-q.queueChannel
	assert.False(t, ok, "The queueChannel should be closed after stopping")
	_, ok = <-q.frontChannel
	assert.False(t, ok, "The frontChannel should be closed after stopping")
	assert.Nil(t, q.inQueue, "The inQueue map should be empty after stopping")
	assert.True(t, q.stopper.Client().Stopped().IsDone(), "The stopper should be stopped")
}

func TestStartAfterStop(t *testing.T) {
	q := NewUniQueue[int](queueSize)
	assert.NoError(t, q.Start())
	q.Stop()
	assert.NoError(t, q.Start())
}

func TestPushCPanicsIfStartIsNotCalled(t *testing.T) {
	q := NewUniQueue[int](queueSize)
	callPushC := func() {
		q.PushC()
	}
	assert.Panics(t, callPushC, "PushC should panic if it's called before Start")
}

func TestPopCPanicsIfStartIsNotCalled(t *testing.T) {
	q := NewUniQueue[int](queueSize)
	callPopC := func() {
		q.PopC()
	}
	assert.Panics(t, callPopC, "PopC should panic if it's called before Start")
}

func TestStartFromDifferentRoutines(_ *testing.T) {
	// This test should fail if we have data races at start
	q := NewUniQueue[int](queueSize)
	go func() {
		utils.IgnoreError(q.Start)
	}()
	go func() {
		utils.IgnoreError(q.Start)
	}()
}

func TestStopFromDifferentRoutines(t *testing.T) {
	// This test should fail if we have data races at start
	q := NewUniQueue[int](queueSize)
	assert.NoError(t, q.Start())
	go func() {
		q.Stop()
	}()
	go func() {
		q.Stop()
	}()
}

func TestAddFromDifferentRoutines(t *testing.T) {
	q := NewUniQueue[int](queueSize)
	assert.NoError(t, q.Start())
	pushC := q.PushC()
	// We push the first item that would be put in the frontChannel
	pushC <- frontValue
	require.True(t, waitWithRetry(func() bool {
		return len(q.PopC()) > 0
	}))
	// The rest of the items can be pushed
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		for _, n := range numsToPush {
			q.PushC() <- n
		}
	}()
	go func() {
		defer wg.Done()
		for _, n := range numsToPush {
			q.PushC() <- n
		}
	}()
	wg.Wait()
	// We push a different value to make sure we read all elements
	pushC <- backValue
	require.True(t, waitWithRetry(func() bool {
		return q.Size() == len(expectedSlice)
	}))
	var queueContent []int
	for i := 0; i < len(expectedSlice); i++ {
		n := <-q.PopC()
		queueContent = append(queueContent, n)
	}
	assert.Equal(t, expectedSlice, queueContent)
	assert.Len(t, q.frontChannel, 0)
	q.Stop()
}

func TestReadFromDifferentRoutines(t *testing.T) {
	q := NewUniQueue[int](queueSize)
	assert.NoError(t, q.Start())
	pushElement(t, q)
	assert.True(t, waitWithRetry(func() bool {
		return len(q.backChannel) == 0
	}))
	wg := &sync.WaitGroup{}
	ctx, cancelFn := context.WithCancel(context.Background())
	wg.Add(2)
	numReads := atomic.Int32{}
	var poppedContent1 []int
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case n, ok := <-q.PopC():
				assert.True(t, ok)
				poppedContent1 = append(poppedContent1, n)
				numReads.Add(1)
				if numReads.Load() == int32(len(expectedMap)) {
					cancelFn()
					return
				}
			}
		}
	}()
	var poppedContent2 []int
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case n, ok := <-q.PopC():
				assert.True(t, ok)
				poppedContent2 = append(poppedContent2, n)
				numReads.Add(1)
				if numReads.Load() == int32(len(expectedMap)) {
					cancelFn()
					return
				}
			}
		}
	}()
	wg.Wait()
	poppedContentMap := make(map[int]struct{})
	for _, el := range poppedContent1 {
		poppedContentMap[el] = struct{}{}
	}
	for _, el := range poppedContent2 {
		poppedContentMap[el] = struct{}{}
	}
	assert.Equal(t, len(poppedContentMap), len(expectedMap))
	assert.Equal(t, expectedMap, poppedContentMap)
	assert.Len(t, q.frontChannel, 0)
	q.Stop()
}

func pushElement(t *testing.T, q *UniQueue[int]) {
	pushC := q.PushC()
	// We push the frontValue item that would be put in the frontChannel
	pushC <- frontValue
	require.True(t, waitWithRetry(func() bool {
		return len(q.PopC()) > 0
	}))
	// The rest of the items can be pushed
	for _, n := range numsToPush {
		pushC <- n
	}
	// We push a different value to make sure we read all elements
	pushC <- backValue
	require.True(t, waitWithRetry(func() bool {
		return q.Size() == len(expectedSlice)
	}))
}

func waitWithRetry(fn func() bool) bool {
	timeout := time.After(defaultTimeout)
	for {
		select {
		case <-timeout:
			return false
		default:
			if fn() {
				return true
			}
		}
	}
}
