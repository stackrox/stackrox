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

func TestAdd(t *testing.T) {
	q := NewUniQueue[int](6)
	assert.NoError(t, q.Start())
	numsToPush := []int{1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 3}
	expected := []int{0, 1, 2, 3}
	pushC := q.PushC()
	// We push the first item that would be put in the frontChannel
	pushC <- 0
	require.True(t, waitWithRetry(func() bool {
		return len(q.PopC()) > 0
	}))
	// The rest of the items can be pushed
	for _, n := range numsToPush {
		pushC <- n
	}
	assert.True(t, waitWithRetry(func() bool {
		return q.Size() == 3
	}))
	var queueContent []int
	for i := 0; i < len(expected); i++ {
		n := <-q.PopC()
		queueContent = append(queueContent, n)
	}

	assert.Equal(t, expected, queueContent)
	assert.Len(t, q.frontChannel, 0)
	q.Stop()
}

func TestCallStartTwice(t *testing.T) {
	q := NewUniQueue[int](5)
	assert.NoError(t, q.Start())
	assert.Error(t, q.Start())
}

func TestStop(t *testing.T) {
	q := NewUniQueue[int](5)
	assert.NoError(t, q.Start())
	q.Stop()
	_, ok := <-q.backChannel
	assert.False(t, ok)
	_, ok = <-q.queueChannel
	assert.False(t, ok)
	_, ok = <-q.frontChannel
	assert.False(t, ok)
	assert.Nil(t, q.inQueue)
	assert.True(t, q.stopper.Client().Stopped().IsDone())
}

func TestStartAfterStop(t *testing.T) {
	q := NewUniQueue[int](5)
	assert.NoError(t, q.Start())
	q.Stop()
	assert.NoError(t, q.Start())
}

func TestPushCPanicsIfStartIsNotCalled(t *testing.T) {
	q := NewUniQueue[int](5)
	callPushC := func() {
		q.PushC()
	}
	assert.Panics(t, callPushC)
}

func TestPopCPanicsIfStartIsNotCalled(t *testing.T) {
	q := NewUniQueue[int](5)
	callPopC := func() {
		q.PopC()
	}
	assert.Panics(t, callPopC)
}

func TestStartFromDifferentRoutines(t *testing.T) {
	q := NewUniQueue[int](5)
	go func() {
		utils.IgnoreError(q.Start)
	}()
	go func() {
		utils.IgnoreError(q.Start)
	}()
}

func TestStopFromDifferentRoutines(t *testing.T) {
	q := NewUniQueue[int](5)
	assert.NoError(t, q.Start())
	go func() {
		q.Stop()
	}()
	go func() {
		q.Stop()
	}()
}

func TestAddFromDifferentRoutines(t *testing.T) {
	q := NewUniQueue[int](5)
	assert.NoError(t, q.Start())
	numsToPush := []int{1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2}
	expected := []int{0, 1, 2, 3}
	pushC := q.PushC()
	// We push the first item that would be put in the frontChannel
	pushC <- 0
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
	pushC <- 3
	assert.True(t, waitWithRetry(func() bool {
		return q.Size() == 3
	}))
	var queueContent []int
	for i := 0; i < len(expected); i++ {
		n := <-q.PopC()
		queueContent = append(queueContent, n)
	}
	assert.Equal(t, expected, queueContent)
	assert.Len(t, q.frontChannel, 0)
	q.Stop()
}

func TestReadFromDifferentRoutines(t *testing.T) {
	q := NewUniQueue[int](5)
	assert.NoError(t, q.Start())
	numsToPush := []int{1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 3}
	expected := map[int]struct{}{
		0: {},
		1: {},
		2: {},
		3: {},
	}
	pushC := q.PushC()
	// We push the first item that would be put in the frontChannel
	pushC <- 0
	require.True(t, waitWithRetry(func() bool {
		return len(q.PopC()) > 0
	}))
	// The rest of the items can be pushed
	for _, n := range numsToPush {
		pushC <- n
	}
	assert.True(t, waitWithRetry(func() bool {
		return q.Size() == 3
	}))
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
				if numReads.Load() == int32(len(expected)) {
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
				if numReads.Load() == int32(len(expected)) {
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
	assert.Equal(t, len(poppedContentMap), len(expected))
	assert.Equal(t, expected, poppedContentMap)
	assert.Len(t, q.frontChannel, 0)
	q.Stop()
}

func waitWithRetry(fn func() bool) bool {
	timeout := time.After(3 * time.Second)
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
