//go:build sql_integration

package m220tom221

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type migrationTestSuite struct {
	suite.Suite

	db  *pghelper.TestPostgres
	ctx context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)

	// Create a test table with a serialized bytea column.
	_, err := s.db.DB.Exec(s.ctx, `
		CREATE TABLE IF NOT EXISTS test_migration_bytea_to_jsonb (
			id TEXT PRIMARY KEY,
			serialized bytea
		)
	`)
	s.Require().NoError(err)
}

func (s *migrationTestSuite) TearDownSuite() {
	_, _ = s.db.DB.Exec(s.ctx, `DROP TABLE IF EXISTS test_migration_bytea_to_jsonb`)
}

func (s *migrationTestSuite) TestMigration() {
	// Insert a protobuf-serialized row using the old bytea format.
	testObj := &storage.NetworkGraphConfig{
		HideDefaultExternalSrcs: true,
	}
	serialized, err := proto.Marshal(testObj)
	s.Require().NoError(err)

	_, err = s.db.DB.Exec(s.ctx, `INSERT INTO test_migration_bytea_to_jsonb (id, serialized) VALUES ($1, $2)`,
		"test-id-1", serialized)
	s.Require().NoError(err)

	// Override the table map for testing.
	origMap := tableToProtoName
	tableToProtoName = map[string]protoreflect.FullName{
		"test_migration_bytea_to_jsonb": "storage.NetworkGraphConfig",
	}
	defer func() { tableToProtoName = origMap }()

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	// Verify the data was converted to JSON.
	var jsonData []byte
	err = s.db.DB.QueryRow(s.ctx, `SELECT serialized FROM test_migration_bytea_to_jsonb WHERE id = $1`, "test-id-1").Scan(&jsonData)
	s.Require().NoError(err)

	// Unmarshal the JSON data and verify.
	var result storage.NetworkGraphConfig
	err = protojson.Unmarshal(jsonData, &result)
	s.Require().NoError(err)
	s.True(result.GetHideDefaultExternalSrcs())

	// Verify the column type is now jsonb.
	var colType string
	err = s.db.DB.QueryRow(s.ctx,
		`SELECT data_type FROM information_schema.columns WHERE table_name = 'test_migration_bytea_to_jsonb' AND column_name = 'serialized'`,
	).Scan(&colType)
	s.Require().NoError(err)
	s.Equal("jsonb", colType)
}
