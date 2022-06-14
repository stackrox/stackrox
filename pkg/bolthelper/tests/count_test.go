package tests

import (
	"testing"

	. "github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestCountLeavesRecursive(t *testing.T) {
	suite.Run(t, new(countTestSuite))
}

type countTestSuite struct {
	suite.Suite

	db            *bolt.DB
	testBucketRef BucketRef
}

func (s *countTestSuite) SetupSuite() {
	db := testutils.DBForSuite(s)

	testBucket := []byte("testBucket")
	RegisterBucketOrPanic(db, testBucket)

	s.db, s.testBucketRef = db, TopLevelRef(db, []byte(testBucket))
	s.NoError(s.testBucketRef.Update(func(b *bolt.Bucket) error {
		if err := b.Put([]byte("foo"), []byte("bar")); err != nil {
			return err
		}
		if err := b.Put([]byte("baz"), []byte("qux")); err != nil {
			return err
		}
		nested1, err := b.CreateBucket([]byte("nested1"))
		if err != nil {
			return err
		}
		if err := nested1.Put([]byte("a"), []byte("x")); err != nil {
			return err
		}
		if err := nested1.Put([]byte("b"), []byte("y")); err != nil {
			return err
		}
		if err := nested1.Put([]byte("c"), []byte("z")); err != nil {
			return err
		}
		nested2, err := nested1.CreateBucket([]byte("nested2"))
		if err != nil {
			return err
		}
		if err := nested2.Put([]byte("d"), []byte("w")); err != nil {
			return err
		}
		return nil
	}))
}

func (s *countTestSuite) count(maxDepth int) int {
	var c int
	s.NoError(s.testBucketRef.View(func(b *bolt.Bucket) error {
		return CountLeavesRecursive(b, maxDepth, &c)
	}))
	return c
}

func (s *countTestSuite) TestWithDepth0() {
	s.Equal(2, s.count(0))
}

func (s *countTestSuite) TestWithDepth1() {
	s.Equal(5, s.count(1))
}

func (s *countTestSuite) TestWithDepth2() {
	s.Equal(6, s.count(2))
}

func (s *countTestSuite) TestWithDepthInf() {
	s.Equal(6, s.count(-1))
}

func (s *countTestSuite) TearDownSuite() {
	testutils.TearDownDB(s.db)
}
