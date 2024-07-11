package queue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQueue(t *testing.T) {
	t.Parallel()
	q := NewQueue[*string]()

	// 1. Adding a new item to the queue.
	item := "first-item"
	q.Push(&item)

	// 2. Using pull should retrieve the previously added item.
	queueItem := q.Pull()
	assert.Equal(t, item, *queueItem)

	// 3. Add an item after 500ms of waiting. Meanwhile, call pull blocking. It should wait until an item is added
	// and afterward return it.
	time.AfterFunc(500*time.Millisecond, func() {
		item := "second-item"
		q.Push(&item)
	})

	assert.Eventually(t, func() bool {
		queueItem := q.PullBlocking(context.Background())
		return "second-item" == *queueItem
	}, 1*time.Second, 100*time.Millisecond)

	// 4. Another pull should now return an empty value.
	queueItem = q.Pull()
	assert.Nil(t, queueItem)
}
