//go:build sql_integration

package largeobject

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"io"
	"testing"

	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type GormUtilsTestSuite struct {
	suite.Suite

	db     *pgtest.TestPostgres
	ctx    context.Context
	gormDB *gorm.DB
}

func TestLargeObjects(t *testing.T) {
	suite.Run(t, new(GormUtilsTestSuite))
}

func (s *GormUtilsTestSuite) SetupTest() {
	s.db = pgtest.ForT(s.T())
	s.ctx = context.Background()
	s.gormDB = s.db.GetGormDB(s.T()).WithContext(s.ctx)
}

func (s *GormUtilsTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *GormUtilsTestSuite) TestUpsertGet() {
	// Write a long file
	randomData := make([]byte, 90000)
	_, err := rand.Read(randomData)
	s.NoError(err)

	reader := bytes.NewBuffer(randomData)
	tx := s.gormDB.Begin(&sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	los := LargeObjects{tx}
	oid, err := los.Create()
	s.Require().NoError(err)
	err = los.Upsert(oid, reader)
	s.Require().NoError(err)
	s.Require().NoError(tx.Commit().Error)

	// Read it back and verify
	tx = s.gormDB.Begin(&sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	los = LargeObjects{tx}
	writer := bytes.NewBuffer([]byte{})
	s.Require().NoError(los.Get(oid, writer))
	s.Require().NoError(tx.Commit().Error)

	// Overwrite it
	s.Require().Equal(randomData, writer.Bytes())
	reader = bytes.NewBuffer([]byte("hi"))
	tx = s.gormDB.Begin(&sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	los = LargeObjects{tx}
	err = los.Upsert(oid, reader)
	s.Require().NoError(err)
	s.Require().NoError(tx.Commit().Error)

	// Read it back and verify
	tx = s.gormDB.Begin(&sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	los = LargeObjects{tx}
	writer = bytes.NewBuffer([]byte{})
	writer.Reset()
	s.Require().NoError(los.Get(oid, writer))
	s.Require().Equal([]byte("hi"), writer.Bytes())
	s.Require().NoError(tx.Commit().Error)
}

func (s *GormUtilsTestSuite) TestLargeObjectSingleTransaction() {
	tx := s.gormDB.Begin()
	s.Require().NoError(tx.Error)

	los := &LargeObjects{tx}

	id, err := los.Create()
	s.Require().NoError(err)

	obj, err := los.Open(id, ModeWrite|ModeRead)
	s.Require().NoError(err)

	n, err := obj.Write([]byte("testing"))
	s.Require().NoError(err)
	s.Require().Equal(7, n, "Expected n to be 7, got %d", n)

	pos, err := obj.Seek(1, 0)
	s.Require().NoError(err)
	s.Require().Equal(int64(1), pos)

	res := make([]byte, 6)
	n, err = obj.Read(res)
	s.Require().NoError(err)
	s.Require().Equal("esting", string(res))
	s.Require().Equal(6, n)

	n, err = obj.Read(res)
	s.Require().Equal(err, io.EOF)
	s.Require().Zero(n)

	pos, err = obj.Tell()
	s.Require().NoError(err)
	s.Require().EqualValues(7, pos)

	_, err = obj.Truncate(1)
	s.Require().NoError(err)

	pos, err = obj.Seek(-1, 2)
	s.Require().NoError(err)
	s.Require().Zero(pos)

	res = make([]byte, 2)
	n, err = obj.Read(res)
	s.Require().Equal(io.EOF, err)
	s.Require().Equal(1, n)
	s.Require().EqualValues('t', res[0])

	err = obj.Close()
	s.Require().NoError(err)

	err = los.Unlink(id)
	s.Require().NoError(err)

	_, err = los.Open(id, ModeRead)
	s.Require().Contains(err.Error(), "does not exist (SQLSTATE 42704)")
}

func (s *GormUtilsTestSuite) TestLargeObjectMultipleTransactions() {
	tx := s.gormDB.Begin()
	s.Require().NoError(tx.Error)
	los := &LargeObjects{tx}

	id, err := los.Create()
	s.Require().NoError(err)
	obj, err := los.Open(id, ModeWrite|ModeRead)
	s.Require().NoError(err)

	n, err := obj.Write([]byte("testing"))
	s.Require().NoError(err)
	s.Require().Equal(7, n, "Expected n to be 7, got %d", n)

	// Commit the first transaction
	s.Require().NoError(tx.Commit().Error)

	// IMPORTANT: Use the same connection for another query
	query := `select n from generate_series(1,10) n`
	rows, err := s.gormDB.Raw(query).Rows()
	s.Require().NoError(err)
	s.Require().NoError(rows.Err())
	s.NoError(rows.Close())

	// Start a new transaction
	tx2 := s.gormDB.Begin()
	s.Require().NoError(tx.Error)
	los2 := &LargeObjects{tx2}

	// Reopen the large object in the new transaction
	obj2, err := los2.Open(id, ModeWrite|ModeRead)
	s.Require().NoError(err)

	pos, err := obj2.Seek(1, 0)
	s.Require().NoError(err)
	s.Require().EqualValues(1, pos)

	res := make([]byte, 6)
	n, err = obj2.Read(res)
	s.Require().NoError(err)
	s.Require().Equal("esting", string(res))
	s.Require().Equal(6, n)

	n, err = obj2.Read(res)
	s.Require().Equal(err, io.EOF)
	s.Require().Zero(n)

	pos, err = obj2.Tell()
	s.Require().NoError(err)
	s.Require().EqualValues(7, pos)

	_, err = obj2.Truncate(1)
	s.Require().NoError(err)

	pos, err = obj2.Seek(-1, 2)
	s.Require().NoError(err)
	s.Require().Zero(pos)

	res = make([]byte, 2)
	n, err = obj2.Read(res)
	s.Require().Equal(io.EOF, err)
	s.Require().Equal(1, n)
	s.Require().EqualValues('t', res[0])

	err = obj2.Close()
	s.Require().NoError(err)

	err = los2.Unlink(id)
	s.Require().NoError(err)

	_, err = los2.Open(id, ModeRead)
	s.Require().Contains(err.Error(), "does not exist (SQLSTATE 42704)")
}
