//go:build sql_integration

package m222tom223

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/m_222_to_m_223_remove_component_risk_records/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
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
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), frozenSchema.CreateTableRisksStmt)
}

func (s *migrationTestSuite) TestMigration() {
	db := s.db.DB

	// Insert risks of various subject types using raw SQL.
	deploymentID := uuid.NewV4().String()
	imageID := uuid.NewV4().String()
	nodeID := uuid.NewV4().String()
	imageComponentID1 := uuid.NewV4().String()
	imageComponentID2 := uuid.NewV4().String()
	nodeComponentID := uuid.NewV4().String()

	insertStmt := "INSERT INTO risks (id, subject_type, score, serialized) VALUES ($1, $2, $3, $4)"
	emptyBytes := []byte{}

	for _, tc := range []struct {
		id          string
		subjectType storage.RiskSubjectType
	}{
		{deploymentID, storage.RiskSubjectType_DEPLOYMENT},
		{imageID, storage.RiskSubjectType_IMAGE},
		{nodeID, storage.RiskSubjectType_NODE},
		{imageComponentID1, storage.RiskSubjectType_IMAGE_COMPONENT},
		{imageComponentID2, storage.RiskSubjectType_IMAGE_COMPONENT},
		{nodeComponentID, storage.RiskSubjectType_NODE_COMPONENT},
	} {
		_, err := db.Exec(s.ctx, insertStmt, tc.id, tc.subjectType, 1.0, emptyBytes)
		s.Require().NoError(err)
	}

	// Verify all 6 risks are present.
	var count int
	err := db.QueryRow(s.ctx, "SELECT COUNT(*) FROM risks").Scan(&count)
	s.Require().NoError(err)
	s.Equal(6, count)

	// Run migration.
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: db,
		DBCtx:      s.ctx,
	}
	s.Require().NoError(migration.Run(dbs))

	// Verify only non-component risks remain.
	err = db.QueryRow(s.ctx, "SELECT COUNT(*) FROM risks").Scan(&count)
	s.Require().NoError(err)
	s.Equal(3, count)

	// Verify the remaining risks are the expected types.
	for _, id := range []string{deploymentID, imageID, nodeID} {
		var exists bool
		err := db.QueryRow(s.ctx, "SELECT EXISTS(SELECT 1 FROM risks WHERE id = $1)", id).Scan(&exists)
		s.Require().NoError(err)
		s.True(exists, "risk %s should still exist", id)
	}

	// Verify component risks are gone.
	for _, id := range []string{imageComponentID1, imageComponentID2, nodeComponentID} {
		var exists bool
		err := db.QueryRow(s.ctx, "SELECT EXISTS(SELECT 1 FROM risks WHERE id = $1)", id).Scan(&exists)
		s.Require().NoError(err)
		s.False(exists, "risk %s should have been removed", id)
	}
}

func (s *migrationTestSuite) TestMigrationLargeDataset() {
	db := s.db.DB

	// Clean table from prior test.
	_, err := db.Exec(s.ctx, "DELETE FROM risks")
	s.Require().NoError(err)

	const (
		batchSize           = 1000
		imageComponentCount = 500_000
		nodeComponentCount  = 100_000
		deploymentCount     = 10_000
		imageCount          = 10_000
		nodeCount           = 1_000
	)

	bulkInsert := func(count int, subjectType storage.RiskSubjectType) {
		for i := 0; i < count; i += batchSize {
			n := batchSize
			if i+n > count {
				n = count - i
			}
			values := make([]string, 0, n)
			args := make([]any, 0, n*4)
			for j := range n {
				offset := j * 4
				values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d)", offset+1, offset+2, offset+3, offset+4))
				args = append(args, uuid.NewV4().String(), subjectType, 1.0, []byte{})
			}
			_, err := db.Exec(s.ctx,
				"INSERT INTO risks (id, subject_type, score, serialized) VALUES "+strings.Join(values, ","),
				args...,
			)
			s.Require().NoError(err)
		}
	}

	s.T().Log("Inserting test data...")
	insertStart := time.Now()

	bulkInsert(imageComponentCount, storage.RiskSubjectType_IMAGE_COMPONENT)
	bulkInsert(nodeComponentCount, storage.RiskSubjectType_NODE_COMPONENT)
	bulkInsert(deploymentCount, storage.RiskSubjectType_DEPLOYMENT)
	bulkInsert(imageCount, storage.RiskSubjectType_IMAGE)
	bulkInsert(nodeCount, storage.RiskSubjectType_NODE)

	s.T().Logf("Inserted %d rows in %v", imageComponentCount+nodeComponentCount+deploymentCount+imageCount+nodeCount, time.Since(insertStart))

	// Verify total row count.
	var totalCount int
	err = db.QueryRow(s.ctx, "SELECT COUNT(*) FROM risks").Scan(&totalCount)
	s.Require().NoError(err)
	s.Equal(imageComponentCount+nodeComponentCount+deploymentCount+imageCount+nodeCount, totalCount)

	// Run migration and measure duration.
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: db,
		DBCtx:      s.ctx,
	}

	migrationStart := time.Now()
	s.Require().NoError(migration.Run(dbs))
	migrationDuration := time.Since(migrationStart)

	s.T().Logf("Migration deleted %d component risk rows in %v", imageComponentCount+nodeComponentCount, migrationDuration)

	// Verify only non-component risks remain.
	var remainingCount int
	err = db.QueryRow(s.ctx, "SELECT COUNT(*) FROM risks").Scan(&remainingCount)
	s.Require().NoError(err)
	s.Equal(deploymentCount+imageCount+nodeCount, remainingCount)

	// Verify no component risks remain.
	var componentCount int
	err = db.QueryRow(s.ctx, "SELECT COUNT(*) FROM risks WHERE subject_type IN ($1, $2)",
		storage.RiskSubjectType_IMAGE_COMPONENT, storage.RiskSubjectType_NODE_COMPONENT).Scan(&componentCount)
	s.Require().NoError(err)
	s.Equal(0, componentCount)
}
