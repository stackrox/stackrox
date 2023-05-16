package largeobject

import (
	"bytes"
	"context"
	"crypto/rand"
	"testing"

	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stretchr/testify/suite"
)

type gormUtilsTestSuite struct {
	suite.Suite

	db  *pghelper.TestPostgres
	ctx context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(gormUtilsTestSuite))
}

func (s *gormUtilsTestSuite) SetupTest() {
	s.db = pghelper.ForT(s.T(), true)
	s.ctx = context.Background()

}

func (s *gormUtilsTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *gormUtilsTestSuite) TestMigration() {
	randomData := make([]byte, 10000)
	_, err := rand.Read(randomData)
	s.NoError(err)

	reader := bytes.NewBuffer(randomData)
	gormDB := s.db.GetGormDB()
	tx := gormDB.Begin()
	los := LargeObjects{tx}
	oid, err := los.Create()
	s.NoError(err)
	err = los.Upsert(oid, reader)
	s.NoError(err)
	s.NoError(tx.Commit().Error)
}
