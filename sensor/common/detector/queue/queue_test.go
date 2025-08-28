package queue

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/suite"
)

type queueSuite struct {
	suite.Suite
}

func TestQueue(t *testing.T) {
	suite.Run(t, new(queueSuite))
}

func (s *queueSuite) createAndStartQueue(stopper concurrency.Stopper, size int) *Queue[*string] {
	q := NewQueue[*string](stopper, "queue", size, nil, nil)
	q.Start()
	return q
}

func (s *queueSuite) TestPauseAndResume() {
	s.T().Setenv(features.SensorCapturesIntermediateEvents.EnvVar(), "true")
	cases := map[string][]func(*Queue[*string], concurrency.Stopper){
		"Pause":                              {s.push, s.pause, s.noPull},
		"Pause, resume":                      {s.push, s.pause, s.push, s.resume, s.pull, s.pull},
		"Pause, stop":                        {s.push, s.pause, s.noPull, s.stopPull},
		"2 push, pull, pause, pull":          {s.push, s.push, s.resume, s.pull, s.pause, s.pull, s.stopPull},
		"2 Push, pull, pause, push, no pull": {s.push, s.push, s.resume, s.pull, s.pause, s.pull, s.push, s.noPull, s.stopPull},
		"Block until push":                   {s.resume, s.pushPullBlocking},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			testStopper := concurrency.NewStopper()
			queueStopper := concurrency.NewStopper()
			q := s.createAndStartQueue(queueStopper, 0)
			for _, fn := range tc {
				fn(q, testStopper)
			}
			testStopper.Client().Stop()
			select {
			case <-time.After(2 * time.Second):
				s.Fail("timeout waiting for the queue to be unblocked")
			case <-testStopper.Flow().StopRequested():
				queueStopper.Client().Stop()
				return
			}
		})
	}
}

func (s *queueSuite) push(q *Queue[*string], _ concurrency.Stopper) {
	old := q.queue.Len()
	item := "item"
	q.Push(&item)
	s.Assert().Equal(old+1, q.queue.Len())
}

func (s *queueSuite) pause(q *Queue[*string], _ concurrency.Stopper) {
	q.Pause()
}

func (s *queueSuite) noPull(q *Queue[*string], stopper concurrency.Stopper) {
	ch := make(chan *string)
	go func() {
		defer close(ch)
		select {
		case item := <-q.Pull():
			ch <- item
		case <-stopper.Flow().StopRequested():
		}
	}()
	select {
	case <-time.After(500 * time.Millisecond):
		return
	case item := <-ch:
		s.Failf("should not pull from the queue", "%s was pulled", *item)
	}
}

func (s *queueSuite) resume(q *Queue[*string], _ concurrency.Stopper) {
	q.Resume()
}

func (s *queueSuite) pull(q *Queue[*string], stopper concurrency.Stopper) {
	ch := make(chan *string)
	go func() {
		defer close(ch)
		select {
		case item := <-q.Pull():
			ch <- item
		case <-stopper.Flow().StopRequested():
		}
	}()
	select {
	case <-time.After(500 * time.Millisecond):
		s.Fail("timeout waiting to pull from the queue")
	case item := <-ch:
		s.Assert().Equal("item", *item)
	}
}

func (s *queueSuite) stopPull(q *Queue[*string], _ concurrency.Stopper) {
	time.AfterFunc(500*time.Millisecond, func() {
		q.stopper.Client().Stop()
	})
	s.Assert().Eventually(func() bool {
		item := <-q.Pull()
		return nil == item
	}, 1*time.Second, 100*time.Millisecond)
}

func (s *queueSuite) pushPullBlocking(q *Queue[*string], _ concurrency.Stopper) {
	time.AfterFunc(500*time.Millisecond, func() {
		item := "item"
		q.Push(&item)
	})
	s.Assert().Eventually(func() bool {
		item := <-q.Pull()
		return "item" == *item
	}, 1*time.Second, 100*time.Millisecond)
}
