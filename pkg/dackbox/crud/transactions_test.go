package crud

import (
	"os"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/suite"
)

func TestTransactionCounter(t *testing.T) {
	suite.Run(t, new(DackBoxTestSuite))
}

type DackBoxTestSuite struct {
	suite.Suite

	dir string
	db  *badger.DB
	sdb *dackbox.DackBox
}

func (s *DackBoxTestSuite) SetupTest() {
	var err error
	s.db, s.dir, err = badgerhelper.NewTemp("reference", true)
	if err != nil {
		s.FailNowf("failed to create DB: %+v", err.Error())
	}
	s.sdb, err = dackbox.NewDackBox(s.db, []byte{})
	if err != nil {
		s.FailNowf("failed to create counter: %+v", err.Error())
	}
}

func (s *DackBoxTestSuite) TearDownTest() {
	_ = s.db.Close()
	_ = os.RemoveAll(s.dir)
}

func (s *DackBoxTestSuite) TestConcurrentTransactionUpdates() {
	numConcurrentTxnCounters := 100

	allWaiting := sync.WaitGroup{}
	allWaiting.Add(numConcurrentTxnCounters)
	allFinished := sync.WaitGroup{}
	allFinished.Add(numConcurrentTxnCounters)

	counter, err := NewTxnCounter(s.sdb, []byte("counter"))
	s.NoError(err, "initialization should not fail")
	for i := 0; i < numConcurrentTxnCounters; i++ {
		go func() {
			allWaiting.Done()
			allWaiting.Wait()

			err := counter.IncTxnCount()
			s.NoError(err, "incrememnt should not fail")

			allFinished.Done()
		}()
	}

	allFinished.Wait()
	loaded, err := NewTxnCounter(s.sdb, []byte("counter"))
	s.NoError(err, "load should not fail")
	s.Equal(uint64(numConcurrentTxnCounters), loaded.GetTxnCount())
}
