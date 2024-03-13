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
