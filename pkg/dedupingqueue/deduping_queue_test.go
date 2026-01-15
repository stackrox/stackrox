package dedupingqueue

import (
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stretchr/testify/suite"
)

type uniQueueSuite struct {
	suite.Suite
}

func TestUniQueue(t *testing.T) {
	suite.Run(t, new(uniQueueSuite))
}

func (s *uniQueueSuite) TestPushPull() {
	items := []*testItem{{1}, {2}, {1}, {3}}
	expectedItems := []*testItem{{1}, {2}, {3}}
	q := NewDedupingQueue[string]()
	for _, i := range items {
		q.Push(i)
	}
	stopSignal := concurrency.NewSignal()
	defer stopSignal.Signal()
	for _, expectedItem := range expectedItems {
		i := q.PullBlocking(&stopSignal)
		ii, ok := i.(*testItem)
		if !ok {
			s.T().Fatalf("item %v in the queue should be of type testItem", i)
		}
		s.Assert().Equal(ii.value, expectedItem.value)
	}
}

func (s *uniQueueSuite) TestPushItemsWithUndefinedKey() {
	// If the item as an implemented `GetDedupeKey`, all items must be pushed to the queue
	items := []*itemWithNoKeyFunction{{val: 0}, {val: 0}}
	q := NewDedupingQueue[string]()
	for _, i := range items {
		q.Push(i)
	}
	s.Assert().Equal(q.queue.Len(), len(items), "should have len %d", len(items))
}

func (s *uniQueueSuite) TestPullFromEmpty() {
	q := NewDedupingQueue[string]()
	// Pulling from an empty queue should not block
	// This should never happen as `pull` should only be called from `PullBlocking`
	s.Never(func() bool {
		i := q.pull()
		return i != nil
	}, 10*time.Millisecond, time.Millisecond)
}

func (s *uniQueueSuite) TestPullBlocking() {
	q := NewDedupingQueue[string]()
	stopSignal := concurrency.NewSignal()
	time.AfterFunc(200*time.Millisecond, func() {
		stopSignal.Signal()
	})
	s.Eventually(func() bool {
		item := q.PullBlocking(&stopSignal)
		return item == nil
	}, time.Second, 100*time.Millisecond, "an nil value should be returned after the stop signal is triggered")
}

type testItem struct {
	value int
}

func (i *testItem) GetDedupeKey() string {
	return fmt.Sprintf("%d", i.value)
}

type itemWithNoKeyFunction struct {
	val int
}

func (u *itemWithNoKeyFunction) GetDedupeKey() string {
	return ""
}

func (s *uniQueueSuite) TestMaxSize_DropNewItemsWhenFull() {
	// When queue is full and a new item with a new dedupe key arrives, it should be dropped
	maxSize := 3
	q := NewDedupingQueue[string](WithMaxSize[string](maxSize))

	// Fill the queue to max capacity
	q.Push(&testItem{value: 1})
	q.Push(&testItem{value: 2})
	q.Push(&testItem{value: 3})
	s.Assert().Equal(maxSize, q.queue.Len(), "queue should be at max capacity")

	// Try to push a new item with a new dedupe key
	q.Push(&testItem{value: 4})
	s.Assert().Equal(maxSize, q.queue.Len(), "queue size should remain at max capacity")

	// Verify the queue still contains only the original items
	stopSignal := concurrency.NewSignal()
	defer stopSignal.Signal()

	expectedValues := []int{1, 2, 3}
	for _, expectedValue := range expectedValues {
		item := q.PullBlocking(&stopSignal)
		ti, ok := item.(*testItem)
		s.Require().True(ok, "item should be testItem")
		s.Assert().Equal(expectedValue, ti.value)
	}
}

func (s *uniQueueSuite) TestMaxSize_ReplaceExistingItemWhenFull() {
	// When queue is full and an item with an existing dedupe key arrives, it should replace the old one
	maxSize := 3
	q := NewDedupingQueue[string](WithMaxSize[string](maxSize))

	// Fill the queue to max capacity
	q.Push(&testItem{value: 1})
	q.Push(&testItem{value: 2})
	q.Push(&testItem{value: 3})
	s.Assert().Equal(maxSize, q.queue.Len(), "queue should be at max capacity")

	// Push an item with the same dedupe key as value 2 (should replace it)
	q.Push(&testItem{value: 2})
	s.Assert().Equal(maxSize, q.queue.Len(), "queue size should remain at max capacity")

	// Verify the queue contains the items (with value 2 moved to its new position before value 3)
	stopSignal := concurrency.NewSignal()
	defer stopSignal.Signal()

	expectedValues := []int{1, 2, 3}
	for _, expectedValue := range expectedValues {
		item := q.PullBlocking(&stopSignal)
		ti, ok := item.(*testItem)
		s.Require().True(ok, "item should be testItem")
		s.Assert().Equal(expectedValue, ti.value)
	}
}

func (s *uniQueueSuite) TestMaxSize_DropItemsWithNoDedupeKeyWhenFull() {
	// When queue is full and an item with no dedupe key arrives, it should be dropped
	maxSize := 2
	q := NewDedupingQueue[string](WithMaxSize[string](maxSize))

	// Fill the queue with items that have no dedupe key
	q.Push(&itemWithNoKeyFunction{val: 1})
	q.Push(&itemWithNoKeyFunction{val: 2})
	s.Assert().Equal(maxSize, q.queue.Len(), "queue should be at max capacity")

	// Try to push another item with no dedupe key
	q.Push(&itemWithNoKeyFunction{val: 3})
	s.Assert().Equal(maxSize, q.queue.Len(), "queue size should remain at max capacity")

	// Verify the queue still contains only the first two items
	stopSignal := concurrency.NewSignal()
	defer stopSignal.Signal()

	expectedValues := []int{1, 2}
	for _, expectedValue := range expectedValues {
		item := q.PullBlocking(&stopSignal)
		nk, ok := item.(*itemWithNoKeyFunction)
		s.Require().True(ok, "item should be itemWithNoKeyFunction")
		s.Assert().Equal(expectedValue, nk.val)
	}
}

func (s *uniQueueSuite) TestMaxSize_MetricsTracking() {
	// Verify that metrics are correctly tracked for adds, dedupes, and drops
	maxSize := 2

	var addCount, dedupeCount, dropCount int
	metricFunc := func(op ops.Op, _ string) {
		switch op {
		case ops.Add:
			addCount++
		case ops.Dedupe:
			dedupeCount++
		case ops.Dropped:
			dropCount++
		}
	}

	q := NewDedupingQueue[string](
		WithMaxSize[string](maxSize),
		WithOperationMetricsFunc[string](metricFunc),
	)

	// Add two items (should count as 2 adds)
	q.Push(&testItem{value: 1})
	q.Push(&testItem{value: 2})
	s.Assert().Equal(2, addCount, "should have 2 adds")
	s.Assert().Equal(0, dedupeCount, "should have 0 dedupes")
	s.Assert().Equal(0, dropCount, "should have 0 drops")

	// Push duplicate of item 1 (should count as 1 dedupe)
	q.Push(&testItem{value: 1})
	s.Assert().Equal(2, addCount, "should still have 2 adds")
	s.Assert().Equal(1, dedupeCount, "should have 1 dedupe")
	s.Assert().Equal(0, dropCount, "should have 0 drops")

	// Try to add a new item when full (should count as 1 drop)
	q.Push(&testItem{value: 3})
	s.Assert().Equal(2, addCount, "should still have 2 adds")
	s.Assert().Equal(1, dedupeCount, "should still have 1 dedupe")
	s.Assert().Equal(1, dropCount, "should have 1 drop")

	// Add item with no dedupe key when full (should count as 1 drop)
	q.Push(&itemWithNoKeyFunction{val: 10})
	s.Assert().Equal(2, addCount, "should still have 2 adds")
	s.Assert().Equal(1, dedupeCount, "should still have 1 dedupe")
	s.Assert().Equal(2, dropCount, "should have 2 drops")
}
