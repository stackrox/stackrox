package dedupingqueue

import (
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/suite"
)

type uniQueueSuite struct {
	suite.Suite
}

func TestUniQueue(t *testing.T) {
	suite.Run(t, new(uniQueueSuite))
}

func (s *uniQueueSuite) TestPushPull() {
	items := []*testItem{{value: 1}, {value: 2}, {value: 1}, {value: 3}}
	expectedItems := []*testItem{{value: 1}, {value: 2}, {value: 3}}
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

func (s *uniQueueSuite) TestMergeableItemsMergeOnDuplicate() {
	q := NewDedupingQueue[string]()
	q.Push(&mergeableItem{id: "a", flag1: false, flag2: true})
	q.Push(&mergeableItem{id: "a", flag1: true, flag2: false})

	s.Assert().Equal(1, q.queue.Len(), "duplicate key should result in one item")

	stopSignal := concurrency.NewSignal()
	defer stopSignal.Signal()
	item := q.PullBlocking(&stopSignal)
	merged, ok := item.(*mergeableItem)
	s.Require().True(ok)
	s.Assert().True(merged.flag1, "flag1 should be sticky-true after merge")
	s.Assert().True(merged.flag2, "flag2 should be sticky-true after merge")
}

func (s *uniQueueSuite) TestMergeableThreeWayMerge() {
	q := NewDedupingQueue[string]()
	q.Push(&mergeableItem{id: "a", flag1: false, flag2: false})
	q.Push(&mergeableItem{id: "a", flag1: true, flag2: false})
	q.Push(&mergeableItem{id: "a", flag1: false, flag2: true})

	s.Assert().Equal(1, q.queue.Len())

	stopSignal := concurrency.NewSignal()
	defer stopSignal.Signal()
	item := q.PullBlocking(&stopSignal)
	merged, ok := item.(*mergeableItem)
	s.Require().True(ok)
	s.Assert().True(merged.flag1, "flag1 should accumulate across three pushes")
	s.Assert().True(merged.flag2, "flag2 should accumulate across three pushes")
}

func (s *uniQueueSuite) TestNonMergeableItemsReplaceOnDuplicate() {
	q := NewDedupingQueue[string]()
	q.Push(&testItem{value: 1, payload: "first"})
	q.Push(&testItem{value: 1, payload: "second"})

	s.Assert().Equal(1, q.queue.Len(), "duplicate key should result in one item")

	stopSignal := concurrency.NewSignal()
	defer stopSignal.Signal()
	item := q.PullBlocking(&stopSignal)
	pulled, ok := item.(*testItem)
	s.Require().True(ok)
	s.Assert().Equal("second", pulled.payload, "new item should replace the old one")
}

func (s *uniQueueSuite) TestMergeablePreservesQueueOrder() {
	q := NewDedupingQueue[string]()
	q.Push(&mergeableItem{id: "a", flag1: true, flag2: false})
	q.Push(&mergeableItem{id: "b", flag1: false, flag2: true})
	q.Push(&mergeableItem{id: "a", flag1: false, flag2: true})

	s.Assert().Equal(2, q.queue.Len(), "should have two items after merge")

	stopSignal := concurrency.NewSignal()
	defer stopSignal.Signal()

	firstItem := q.PullBlocking(&stopSignal)
	first, ok := firstItem.(*mergeableItem)
	s.Require().True(ok)
	s.Assert().Equal("a", first.id, "merged item should keep its original position")
	s.Assert().True(first.flag1)
	s.Assert().True(first.flag2)

	secondItem := q.PullBlocking(&stopSignal)
	second, ok := secondItem.(*mergeableItem)
	s.Require().True(ok)
	s.Assert().Equal("b", second.id)
}

type testItem struct {
	value   int
	payload string
}

func (i *testItem) GetDedupeKey() string {
	return fmt.Sprintf("%d", i.value)
}

type mergeableItem struct {
	id    string
	flag1 bool
	flag2 bool
}

func (i *mergeableItem) GetDedupeKey() string {
	return i.id
}

func (i *mergeableItem) MergeFrom(old Item[string]) {
	oldItem, ok := old.(*mergeableItem)
	if !ok {
		return
	}
	i.flag1 = i.flag1 || oldItem.flag1
	i.flag2 = i.flag2 || oldItem.flag2
}

type itemWithNoKeyFunction struct {
	val int
}

func (u *itemWithNoKeyFunction) GetDedupeKey() string {
	return ""
}
