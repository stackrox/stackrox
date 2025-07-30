//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	postgresSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stretchr/testify/suite"
)

func TestImageMapperDataStoreWithFlattenImageDataEnabled(t *testing.T) {
	suite.Run(t, new(ImageV2DataStoreTestSuite))
}

type ImageV2DataStoreTestSuite struct {
	suite.Suite

	ctx       context.Context
	testDB    *pgtest.TestPostgres
	datastore imageDatastore.DataStore
}

func (s *ImageV2DataStoreTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.testDB = pgtest.ForT(s.T())
}

func (s *ImageV2DataStoreTestSuite) SetupTest() {
	s.datastore = GetTestPostgresDataStore(s.T(), s.testDB.DB)
}

func (s *ImageV2DataStoreTestSuite) TearDownTest() {
	s.truncateTable(postgresSchema.ImagesTableName)
	s.truncateTable(postgresSchema.ImageComponentsTableName)
	s.truncateTable(postgresSchema.ImageCvesTableName)
	s.truncateTable(postgresSchema.ImagesV2TableName)
	s.truncateTable(postgresSchema.ImageComponentV2TableName)
	s.truncateTable(postgresSchema.ImageCvesV2TableName)
}

func (s *ImageV2DataStoreTestSuite) TearDownSuite() {
	s.testDB.Close()
}

func (s *ImageV2DataStoreTestSuite) truncateTable(name string) {
	sql := fmt.Sprintf("TRUNCATE %s CASCADE", name)
	_, err := s.testDB.Exec(s.ctx, sql)
	s.NoError(err)
}
