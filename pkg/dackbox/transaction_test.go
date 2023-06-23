package dackbox

import (
	"testing"

	"github.com/stackrox/rox/pkg/concurrency/sortedkeys"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestDackBoxTransaction(t *testing.T) {
	suite.Run(t, new(DackBoxTransactionTestSuite))
}

type DackBoxTransactionTestSuite struct {
	suite.Suite

	db  *rocksdb.RocksDB
	sdb *DackBox
}

func (s *DackBoxTransactionTestSuite) SetupTest() {
	var err error
	s.db, err = rocksdb.NewTemp("reference")
	if err != nil {
		s.FailNowf("failed to create DB: %+v", err.Error())
	}
	s.sdb, err = NewRocksDBDackBox(s.db, nil, []byte{}, []byte{}, []byte{})
	if err != nil {
		s.FailNowf("failed to create counter: %+v", err.Error())
	}
}

func (s *DackBoxTransactionTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.db)
}

func (s *DackBoxTransactionTestSuite) TestRefView() {
	// Start with all three keys pointing to the same two keys.
	firstGraph, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer firstGraph.Discard()

	firstGraph.Graph().SetRefs([]byte("f1"), sortedkeys.SortedKeys{[]byte("t1"), []byte("t2")})
	firstGraph.Graph().SetRefs([]byte("f2"), sortedkeys.SortedKeys{[]byte("t1"), []byte("t2")})
	firstGraph.Graph().SetRefs([]byte("f3"), sortedkeys.SortedKeys{[]byte("t1"), []byte("t2")})

	s.Equal([][]byte{[]byte("t1"), []byte("t2")}, firstGraph.Graph().GetRefsFrom([]byte("f1")))
	s.Equal([][]byte{[]byte("t1"), []byte("t2")}, firstGraph.Graph().GetRefsFrom([]byte("f2")))
	s.Equal([][]byte{[]byte("t1"), []byte("t2")}, firstGraph.Graph().GetRefsFrom([]byte("f3")))

	s.Equal([][]byte{[]byte("f1"), []byte("f2"), []byte("f3")}, firstGraph.Graph().GetRefsTo([]byte("t1")))
	s.Equal([][]byte{[]byte("f1"), []byte("f2"), []byte("f3")}, firstGraph.Graph().GetRefsTo([]byte("t2")))

	// Commit the view so that all future views will see it's updates.
	err = firstGraph.Commit()
	s.NoError(err)

	// Change one key to point to two new keys, and update the other with a no-op change.
	secondGraph, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer secondGraph.Discard()

	secondGraph.Graph().SetRefs([]byte("f2"), sortedkeys.SortedKeys{[]byte("t3"), []byte("t4")})
	secondGraph.Graph().SetRefs([]byte("f3"), sortedkeys.SortedKeys{[]byte("t1"), []byte("t2")})

	s.Equal([][]byte{[]byte("t1"), []byte("t2")}, secondGraph.Graph().GetRefsFrom([]byte("f1")))
	s.Equal([][]byte{[]byte("t3"), []byte("t4")}, secondGraph.Graph().GetRefsFrom([]byte("f2")))
	s.Equal([][]byte{[]byte("t1"), []byte("t2")}, secondGraph.Graph().GetRefsFrom([]byte("f3")))

	s.Equal([][]byte{[]byte("f1"), []byte("f3")}, secondGraph.Graph().GetRefsTo([]byte("t1")))
	s.Equal([][]byte{[]byte("f1"), []byte("f3")}, secondGraph.Graph().GetRefsTo([]byte("t2")))
	s.Equal([][]byte{[]byte("f2")}, secondGraph.Graph().GetRefsTo([]byte("t3")))
	s.Equal([][]byte{[]byte("f2")}, secondGraph.Graph().GetRefsTo([]byte("t4")))

	// Create a third view before commit to check that we don't see any of the second views changes.
	thirdGraph, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer thirdGraph.Discard()

	// Delete a ref in the second view after three has been created.
	secondGraph.Graph().DeleteRefsFrom([]byte("f3"))

	// Commit the second view.
	err = secondGraph.Commit()
	s.NoError(err)

	// Create a third view before commit to check that we don't see any of the second views changes.
	forthGraph, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer forthGraph.Discard()

	// Check that the third view sees only the first views changes.
	s.Equal([][]byte{[]byte("t1"), []byte("t2")}, thirdGraph.Graph().GetRefsFrom([]byte("f1")))
	s.Equal([][]byte{[]byte("t1"), []byte("t2")}, thirdGraph.Graph().GetRefsFrom([]byte("f2")))
	s.Equal([][]byte{[]byte("t1"), []byte("t2")}, thirdGraph.Graph().GetRefsFrom([]byte("f3")))

	s.Equal([][]byte{[]byte("f1"), []byte("f2"), []byte("f3")}, thirdGraph.Graph().GetRefsTo([]byte("t1")))
	s.Equal([][]byte{[]byte("f1"), []byte("f2"), []byte("f3")}, thirdGraph.Graph().GetRefsTo([]byte("t2")))
	err = thirdGraph.Commit()
	s.NoError(err)

	// Check that forth view sees the second views changes.
	s.Equal([][]byte{[]byte("t1"), []byte("t2")}, forthGraph.Graph().GetRefsFrom([]byte("f1")))
	s.Equal([][]byte{[]byte("t3"), []byte("t4")}, forthGraph.Graph().GetRefsFrom([]byte("f2")))
	s.Equal([][]uint8(nil), forthGraph.Graph().GetRefsFrom([]byte("f3")))

	s.Equal([][]byte{[]byte("f1")}, forthGraph.Graph().GetRefsTo([]byte("t1")))
	s.Equal([][]byte{[]byte("f1")}, forthGraph.Graph().GetRefsTo([]byte("t2")))
	s.Equal([][]byte{[]byte("f2")}, forthGraph.Graph().GetRefsTo([]byte("t3")))
	s.Equal([][]byte{[]byte("f2")}, forthGraph.Graph().GetRefsTo([]byte("t4")))
}
