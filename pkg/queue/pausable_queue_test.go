package queue

import (
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/suite"
)

type pausableQueueSuite struct {
	suite.Suite
}

func TestPausableQueue(t *testing.T) {
	suite.Run(t, new(pausableQueueSuite))
}

func (s *pausableQueueSuite) createAndStartPausableQueue(opts ...PausableQueueOption[*string]) *pausableQueueImpl[*string] {
	q := NewPausableQueue[*string](opts...)
	q.Resume()
	ret, ok := q.(*pausableQueueImpl[*string])
	s.Require().True(ok)
	return ret
}

func (s *pausableQueueSuite) TestPush() {
	cases := map[string]struct {
		options       []PausableQueueOption[*string]
		items         []string
		expectedItems []string
	}{
		"no aggregators": {
			items:         []string{"item-1", "item-2"},
			expectedItems: []string{"item-1", "item-2"},
		},
		"with one aggregator": {
			options:       []PausableQueueOption[*string]{WithAggregator[*string](aggregateIfPrefixItem)},
			items:         []string{"item-1", "item-2", "no-item"},
			expectedItems: []string{"item-1", "no-item"},
		},
		"with two aggregators": {
			options: []PausableQueueOption[*string]{
				WithAggregator[*string](aggregateIfPrefixItem),
				WithAggregator[*string](aggregateIfEqual),
			},
			items:         []string{"item-1", "item-2", "no-item", "no-item"},
			expectedItems: []string{"item-1", "no-item"},
		},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			q := s.createAndStartPausableQueue(tc.options...)
			for _, it := range tc.items {
				val := it
				q.Push(&val)
			}
			s.Require().Equal(q.internalQueue.queue.Len(), len(tc.expectedItems))
			i := 0
			for e := q.internalQueue.queue.Front(); e != nil; e = e.Next() {
				val, ok := e.Value.(*string)
				s.Require().True(ok)
				s.Assert().Equal(tc.expectedItems[i], *val)
				i++
			}
		})
	}
}

func (s *pausableQueueSuite) TestPauseAndResume() {
	cases := map[string][]func(*pausableQueueImpl[*string], concurrency.Stopper){
		"Pause":            {s.push, s.pause, s.noPull, s.pullBlockingCall},
		"Pause, resume":    {s.push, s.pause, s.push, s.noPull, s.resume, s.pull, s.pullBlocking},
		"Pause, stop":      {s.push, s.pause, s.noPull, s.stopPullBlocking},
		"Block until push": {s.pushPullBlocking},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			stopper := concurrency.NewStopper()
			q := s.createAndStartPausableQueue()
			for _, fn := range tc {
				fn(q, stopper)
			}
			stopper.Client().Stop()
			select {
			case <-time.After(2 * time.Second):
				s.Fail("timeout waiting for the queue to be unblocked")
			case <-stopper.Flow().StopRequested():
				return
			}
		})
	}
}

func (s *pausableQueueSuite) push(q *pausableQueueImpl[*string], _ concurrency.Stopper) {
	old := q.internalQueue.queue.Len()
	item := "item"
	q.Push(&item)
	s.Assert().Equal(old+1, q.internalQueue.queue.Len())
}

func (s *pausableQueueSuite) pause(q *pausableQueueImpl[*string], _ concurrency.Stopper) {
	q.Pause()
}

func (s *pausableQueueSuite) noPull(q *pausableQueueImpl[*string], _ concurrency.Stopper) {
	item := q.Pull()
	s.Assert().Nil(item)
}

func (s *pausableQueueSuite) pullBlockingCall(q *pausableQueueImpl[*string], stopper concurrency.Stopper) {
	ch := make(chan *string)
	go func() {
		defer close(ch)
		ch <- q.PullBlocking(stopper.LowLevel().GetStopRequestSignal())
	}()
	select {
	case <-time.After(500 * time.Millisecond):
		return
	case <-ch:
		s.Fail("PullBlocking should block unless the queue is stopped but returned")
	}
}

func (s *pausableQueueSuite) resume(q *pausableQueueImpl[*string], _ concurrency.Stopper) {
	q.Resume()
}

func (s *pausableQueueSuite) pull(q *pausableQueueImpl[*string], _ concurrency.Stopper) {
	item := q.Pull()
	s.Assert().Equal("item", *item)
}

func (s *pausableQueueSuite) pullBlocking(q *pausableQueueImpl[*string], stopper concurrency.Stopper) {
	item := q.PullBlocking(stopper.LowLevel().GetStopRequestSignal())
	s.Assert().Equal("item", *item)
}

func (s *pausableQueueSuite) stopPullBlocking(q *pausableQueueImpl[*string], stopper concurrency.Stopper) {
	time.AfterFunc(500*time.Millisecond, func() {
		stopper.Client().Stop()
	})
	s.Assert().Eventually(func() bool {
		item := q.PullBlocking(stopper.LowLevel().GetStopRequestSignal())
		return nil == item
	}, 1*time.Second, 100*time.Millisecond)
}

func (s *pausableQueueSuite) pushPullBlocking(q *pausableQueueImpl[*string], stopper concurrency.Stopper) {
	time.AfterFunc(500*time.Millisecond, func() {
		item := "item"
		q.Push(&item)
	})
	s.Assert().Eventually(func() bool {
		item := q.PullBlocking(stopper.LowLevel().GetStopRequestSignal())
		return "item" == *item
	}, 1*time.Second, 100*time.Millisecond)
}

func aggregateIfEqual(x, y *string) (*string, bool) {
	var ret *string
	if *x == *y {
		return x, true
	}
	return ret, false
}

func aggregateIfPrefixItem(x, y *string) (*string, bool) {
	var ret *string
	if strings.HasPrefix(*x, "item") && strings.HasPrefix(*y, "item") {
		ret = x
		return ret, true
	}
	return ret, false
}
