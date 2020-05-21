package dackbox

import (
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
)

func TestDackBox(t *testing.T) {
	suite.Run(t, new(DackBoxTestSuite))
}

type DackBoxTestSuite struct {
	suite.Suite

	dir string
	db  *gorocksdb.DB
	sdb *DackBox
}

func (s *DackBoxTestSuite) SetupTest() {
	var err error
	s.db, s.dir, err = rocksdb.NewTemp("reference")
	if err != nil {
		s.FailNowf("failed to create DB: %+v", err.Error())
	}
	s.sdb, err = NewRocksDBDackBox(s.db, nil, []byte{}, []byte{}, []byte{})
	if err != nil {
		s.FailNowf("failed to create counter: %+v", err.Error())
	}
}

func (s *DackBoxTestSuite) TearDownTest() {
	s.db.Close()
	_ = os.RemoveAll(s.dir)
}

func (s *DackBoxTestSuite) TestRaceAddConfig1() {
	view1 := s.sdb.NewTransaction()
	defer view1.Discard()

	err := view1.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey2")})
	s.NoError(err, "set should have succeeded")
	err = view1.Commit()
	s.NoError(err, "commit should have succeeded")

	view2 := s.sdb.NewTransaction()
	defer view2.Discard()

	err = view2.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey3")})
	s.NoError(err, "set should have succeeded")
	err = view2.Commit()
	s.NoError(err, "commit should have succeeded")

	view3 := s.sdb.NewTransaction()
	defer view3.Discard()

	err = view3.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})
	s.NoError(err, "set should have succeeded")
	err = view3.Commit()
	s.NoError(err, "commit should have succeeded")

	view4 := s.sdb.NewReadOnlyTransaction()
	defer view4.Discard()
	s.Equal([][]byte{[]byte("toKey1"), []byte("toKey4")}, view4.Graph().GetRefsFrom([]byte("fromKey")))
}

func (s *DackBoxTestSuite) TestRaceAddConfig2() {
	view1 := s.sdb.NewTransaction()
	defer view1.Discard()

	err := view1.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey2")})
	s.NoError(err, "set should have succeeded")

	view2 := s.sdb.NewTransaction()
	defer view2.Discard()

	err = view2.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey3")})
	s.NoError(err, "set should have succeeded")

	view3 := s.sdb.NewTransaction()
	defer view3.Discard()

	err = view3.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})
	s.NoError(err, "set should have succeeded")

	err = view1.Commit()
	s.NoError(err, "commit should have succeeded")

	err = view2.Commit()
	s.NoError(err, "commit should have succeeded")

	err = view3.Commit()
	s.NoError(err, "commit should have succeeded")

	view4 := s.sdb.NewReadOnlyTransaction()
	defer view4.Discard()
	s.Equal([][]byte{[]byte("toKey1"), []byte("toKey4")}, view4.Graph().GetRefsFrom([]byte("fromKey")))
}

func (s *DackBoxTestSuite) TestRaceAddConfig3() {
	view1 := s.sdb.NewTransaction()
	defer view1.Discard()

	view2 := s.sdb.NewTransaction()
	defer view2.Discard()

	view3 := s.sdb.NewTransaction()
	defer view3.Discard()

	err := view1.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey2")})
	s.NoError(err, "set should have succeeded")

	err = view2.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey3")})
	s.NoError(err, "set should have succeeded")

	err = view3.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})
	s.NoError(err, "set should have succeeded")

	err = view1.Commit()
	s.NoError(err, "commit should have succeeded")

	err = view2.Commit()
	s.NoError(err, "commit should have succeeded")

	err = view3.Commit()
	s.NoError(err, "commit should have succeeded")

	view4 := s.sdb.NewReadOnlyTransaction()
	defer view4.Discard()
	s.Equal([][]byte{[]byte("toKey1"), []byte("toKey4")}, view4.Graph().GetRefsFrom([]byte("fromKey")))
}

func (s *DackBoxTestSuite) TestRaceAddConfig4() {
	view1 := s.sdb.NewTransaction()
	defer view1.Discard()

	view2 := s.sdb.NewTransaction()
	defer view2.Discard()

	view3 := s.sdb.NewTransaction()
	defer view3.Discard()

	err := view1.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey2")})
	s.NoError(err, "set should have succeeded")
	err = view1.Commit()
	s.NoError(err, "commit should have succeeded")

	err = view2.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey3")})
	s.NoError(err, "set should have succeeded")
	err = view2.Commit()
	s.NoError(err, "commit should have succeeded")

	err = view3.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})
	s.NoError(err, "set should have succeeded")
	err = view3.Commit()
	s.NoError(err, "commit should have succeeded")

	view4 := s.sdb.NewReadOnlyTransaction()
	defer view4.Discard()
	s.Equal([][]byte{[]byte("toKey1"), []byte("toKey4")}, view4.Graph().GetRefsFrom([]byte("fromKey")))
}

func (s *DackBoxTestSuite) TestRaceAddConfig5() {
	view1 := s.sdb.NewTransaction()
	defer view1.Discard()

	view2 := s.sdb.NewTransaction()
	defer view2.Discard()

	view3 := s.sdb.NewTransaction()
	defer view3.Discard()

	err := view3.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})
	s.NoError(err, "set should have succeeded")

	err = view2.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey3")})
	s.NoError(err, "set should have succeeded")

	err = view1.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})
	s.NoError(err, "set should have succeeded")

	err = view1.Commit()
	s.NoError(err, "commit should have succeeded")
	err = view2.Commit()
	s.NoError(err, "commit should have succeeded")
	err = view3.Commit()
	s.NoError(err, "commit should have succeeded")

	view4 := s.sdb.NewReadOnlyTransaction()
	defer view4.Discard()
	s.Equal([][]byte{[]byte("toKey1"), []byte("toKey4")}, view4.Graph().GetRefsFrom([]byte("fromKey")))
}

func (s *DackBoxTestSuite) TestRaceAddDeleteToConfig5() {
	view1 := s.sdb.NewTransaction()
	defer view1.Discard()

	view2 := s.sdb.NewTransaction()
	defer view2.Discard()

	view3 := s.sdb.NewTransaction()
	defer view3.Discard()

	err := view3.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})
	s.NoError(err, "set should have succeeded")
	err = view3.Graph().DeleteRefsTo([]byte("toKey1"))
	s.NoError(err, "set should have succeeded")

	err = view2.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey3")})
	s.NoError(err, "set should have succeeded")

	err = view1.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})
	s.NoError(err, "set should have succeeded")

	err = view1.Commit()
	s.NoError(err, "commit should have succeeded")
	err = view2.Commit()
	s.NoError(err, "commit should have succeeded")
	err = view3.Commit()
	s.NoError(err, "commit should have succeeded")

	view4 := s.sdb.NewReadOnlyTransaction()
	defer view4.Discard()
	s.Equal([][]byte{[]byte("toKey4")}, view4.Graph().GetRefsFrom([]byte("fromKey")))
}

func (s *DackBoxTestSuite) TestRaceAddDeleteFromConfig5() {
	view1 := s.sdb.NewTransaction()
	defer view1.Discard()

	view2 := s.sdb.NewTransaction()
	defer view2.Discard()

	view3 := s.sdb.NewTransaction()
	defer view3.Discard()

	err := view3.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})
	s.NoError(err, "set should have succeeded")
	err = view3.Graph().DeleteRefsFrom([]byte("fromKey"))
	s.NoError(err, "set should have succeeded")

	err = view2.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey1"), []byte("toKey3")})
	s.NoError(err, "set should have succeeded")

	err = view1.Graph().SetRefs([]byte("fromKey"), [][]byte{[]byte("toKey4"), []byte("toKey1")})
	s.NoError(err, "set should have succeeded")

	err = view1.Commit()
	s.NoError(err, "commit should have succeeded")
	err = view2.Commit()
	s.NoError(err, "commit should have succeeded")
	err = view3.Commit()
	s.NoError(err, "commit should have succeeded")

	view4 := s.sdb.NewReadOnlyTransaction()
	defer view4.Discard()
	s.Equal([][]byte(nil), view4.Graph().GetRefsFrom([]byte("fromKey")))
}
