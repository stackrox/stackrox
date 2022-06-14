package queue

import (
	"testing"
	"time"

	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
)

func TestWaitableQueue(t *testing.T) {
	q := NewWaitableQueue()

	input := [][]byte{
		[]byte("id1"),
		[]byte("id2"),
		[]byte("id3"),
		[]byte("id4"),
	}
	q.Push(input[0], nil)
	q.Push(input[1], nil)
	q.Push(input[2], nil)
	q.Push(input[3], nil)

	var results [][]byte
	for q.Length() > 0 {
		k, _, _ := q.Pop()
		results = append(results, k)
	}
	assert.Equal(t, input, results)

	q.Push(input[0], nil)
	q.Push(input[1], nil)
	q.Push(input[2], nil)
	q.Push(input[3], nil)

	results = [][]byte{}
	for q.Length() > 0 {
		k, _, _ := q.Pop()
		results = append(results, k)
	}

	assert.Equal(t, input, results)
}

func TestWaitableQueueConcurrent(t *testing.T) {
	q := NewWaitableQueue()

	input := [][]byte{
		[]byte("id1"),
		[]byte("id2"),
		[]byte("id3"),
		[]byte("id4"),
	}

	var results [][]byte
	go func() {
		for len(results) < 5 {
			// Wait for q to have values.
			<-q.NotEmpty().Done()

			// Pop the next value.
			k, _, s := q.Pop()
			if k != nil {
				results = append(results, k)
			} else if s != nil {
				s.Signal()
			}
		}
	}()

	// Add values to the queue.
	q.Push(input[0], nil)
	q.Push(input[1], nil)
	q.Push(input[2], nil)
	q.Push(input[3], nil)

	assertableSignal := concurrency.NewSignal()
	q.PushSignal(&assertableSignal)

	// Wait for the thread building the results to be done.
	select {
	case <-time.After(time.Second):
		assert.Fail(t, "assertable never returned")
	case <-assertableSignal.Done():
	}

	assert.Equal(t, input, results)
}
