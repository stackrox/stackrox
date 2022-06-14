package dackbox

import (
	"testing"

	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestDackBox(t *testing.T) {
	suite.Run(t, new(DackBoxTestSuite))
}

type DackBoxTestSuite struct {
	suite.Suite

	db  *rocksdb.RocksDB
	sdb *DackBox
}

func (s *DackBoxTestSuite) SetupTest() {
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

func (s *DackBoxTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.db)
}

func (s *DackBoxTestSuite) TestRaceAddConfig1() {
	view1, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view1.Discard()

	view1.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey2")})
	err = view1.Commit()
	s.NoError(err, "commit should have succeeded")

	view2, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view2.Discard()

	view2.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey3")})
	err = view2.Commit()
	s.NoError(err, "commit should have succeeded")

	view3, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view3.Discard()

	view3.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})
	err = view3.Commit()
	s.NoError(err, "commit should have succeeded")

	view4, err := s.sdb.NewReadOnlyTransaction()
	s.NoError(err)
	defer view4.Discard()
	s.Equal([][]byte{[]byte("toKey1"), []byte("toKey4")}, view4.Graph().GetRefsFrom([]byte("fromKey")))
}

func (s *DackBoxTestSuite) TestRaceAddConfig2() {
	view1, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view1.Discard()

	view1.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey2")})

	view2, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view2.Discard()

	view2.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey3")})

	view3, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view3.Discard()

	view3.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})

	err = view1.Commit()
	s.NoError(err, "commit should have succeeded")

	err = view2.Commit()
	s.NoError(err, "commit should have succeeded")

	err = view3.Commit()
	s.NoError(err, "commit should have succeeded")

	view4, err := s.sdb.NewReadOnlyTransaction()
	s.NoError(err)
	defer view4.Discard()
	s.Equal([][]byte{[]byte("toKey1"), []byte("toKey4")}, view4.Graph().GetRefsFrom([]byte("fromKey")))
}

func (s *DackBoxTestSuite) TestRaceAddConfig3() {
	view1, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view1.Discard()

	view2, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view2.Discard()

	view3, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view3.Discard()

	view1.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey2")})

	view2.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey3")})

	view3.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})

	err = view1.Commit()
	s.NoError(err, "commit should have succeeded")

	err = view2.Commit()
	s.NoError(err, "commit should have succeeded")

	err = view3.Commit()
	s.NoError(err, "commit should have succeeded")

	view4, err := s.sdb.NewReadOnlyTransaction()
	s.NoError(err)
	defer view4.Discard()
	s.Equal([][]byte{[]byte("toKey1"), []byte("toKey4")}, view4.Graph().GetRefsFrom([]byte("fromKey")))
}

func (s *DackBoxTestSuite) TestRaceAddConfig4() {
	view1, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view1.Discard()

	view2, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view2.Discard()

	view3, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view3.Discard()

	view1.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey2")})
	err = view1.Commit()
	s.NoError(err, "commit should have succeeded")

	view2.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey3")})
	err = view2.Commit()
	s.NoError(err, "commit should have succeeded")

	view3.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})
	err = view3.Commit()
	s.NoError(err, "commit should have succeeded")

	view4, err := s.sdb.NewReadOnlyTransaction()
	s.NoError(err)
	defer view4.Discard()
	s.Equal([][]byte{[]byte("toKey1"), []byte("toKey4")}, view4.Graph().GetRefsFrom([]byte("fromKey")))
}

func (s *DackBoxTestSuite) TestRaceAddConfig5() {
	view1, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view1.Discard()

	view2, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view2.Discard()

	view3, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view3.Discard()

	view3.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})

	view2.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey3")})

	view1.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})

	err = view1.Commit()
	s.NoError(err, "commit should have succeeded")
	err = view2.Commit()
	s.NoError(err, "commit should have succeeded")
	err = view3.Commit()
	s.NoError(err, "commit should have succeeded")

	view4, err := s.sdb.NewReadOnlyTransaction()
	s.NoError(err)
	defer view4.Discard()
	s.Equal([][]byte{[]byte("toKey1"), []byte("toKey4")}, view4.Graph().GetRefsFrom([]byte("fromKey")))
}

func (s *DackBoxTestSuite) TestRaceAddDeleteToConfig5() {
	view1, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view1.Discard()

	view2, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view2.Discard()

	view3, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view3.Discard()

	view3.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})
	view3.Graph().DeleteRefsTo([]byte("toKey1"))

	view2.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey3")})

	view1.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})

	err = view1.Commit()
	s.NoError(err, "commit should have succeeded")
	err = view2.Commit()
	s.NoError(err, "commit should have succeeded")
	err = view3.Commit()
	s.NoError(err, "commit should have succeeded")

	view4, err := s.sdb.NewReadOnlyTransaction()
	s.NoError(err)
	defer view4.Discard()
	s.Equal([][]byte{[]byte("toKey4")}, view4.Graph().GetRefsFrom([]byte("fromKey")))
}

func (s *DackBoxTestSuite) TestRaceAddDeleteFromConfig5() {
	view1, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view1.Discard()

	view2, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view2.Discard()

	view3, err := s.sdb.NewTransaction()
	s.NoError(err)
	defer view3.Discard()

	view3.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})
	view3.Graph().DeleteRefsFrom([]byte("fromKey"))

	view2.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey3")})

	view1.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})

	err = view1.Commit()
	s.NoError(err, "commit should have succeeded")
	err = view2.Commit()
	s.NoError(err, "commit should have succeeded")
	err = view3.Commit()
	s.NoError(err, "commit should have succeeded")

	view4, err := s.sdb.NewReadOnlyTransaction()
	s.NoError(err)
	defer view4.Discard()
	s.Equal([][]byte(nil), view4.Graph().GetRefsFrom([]byte("fromKey")))
}
