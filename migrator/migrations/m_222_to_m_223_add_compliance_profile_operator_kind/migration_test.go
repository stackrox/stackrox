//go:build sql_integration

package m222tom223

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_222_to_m_223_add_compliance_profile_operator_kind/test/schema"
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
}

func (s *migrationTestSuite) TestMigration() {
	origBatchSize := batchSize
	batchSize = 2 // Force multiple batches
	defer func() { batchSize = origBatchSize }()

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	clusterID := uuid.NewV4().String()

	profiles := []*storage.ComplianceOperatorProfileV2{
		{
			Id:           uuid.NewV4().String(),
			ProfileId:    "profile-1",
			Name:         "ocp4-cis",
			ProductType:  "Platform",
			Standard:     "CIS",
			ClusterId:    clusterID,
			ProfileRefId: uuid.NewV4().String(),
			OperatorKind: storage.ComplianceOperatorProfileV2_PROFILE,
		},
		{
			Id:           uuid.NewV4().String(),
			ProfileId:    "profile-2",
			Name:         "ocp4-cis-tailored",
			ProductType:  "Platform",
			Standard:     "CIS",
			ClusterId:    clusterID,
			ProfileRefId: uuid.NewV4().String(),
			OperatorKind: storage.ComplianceOperatorProfileV2_TAILORED_PROFILE,
		},
		{
			Id:           uuid.NewV4().String(),
			ProfileId:    "profile-3",
			Name:         "ocp4-moderate",
			ProductType:  "Platform",
			Standard:     "NIST-800-53",
			ClusterId:    clusterID,
			ProfileRefId: uuid.NewV4().String(),
			OperatorKind: storage.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED,
		},
	}

	// Create the old schema (without OperatorKind column).
	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, oldSchema.CreateTableComplianceOperatorProfileV2Stmt)

	// Insert profiles using the old schema. The serialized blob contains OperatorKind,
	// but the column does not exist yet.
	for _, profile := range profiles {
		converted, err := oldSchema.ConvertProfileFromProto(profile)
		s.Require().NoError(err)
		s.Require().NoError(dbs.GormDB.Create(converted).Error)
	}

	// Run the migration (adds column + backfills from serialized blob).
	s.Require().NoError(migration.Run(dbs))

	// Verify each profile's OperatorKind column matches the value from the serialized blob.
	for _, profile := range profiles {
		var operatorKind int32
		err := s.db.DB.QueryRow(s.ctx,
			"SELECT operatorkind FROM compliance_operator_profile_v2 WHERE id = $1", profile.GetId(),
		).Scan(&operatorKind)
		s.Require().NoError(err)
		s.Require().Equal(int32(profile.GetOperatorKind()), operatorKind, "OperatorKind mismatch for profile %s", profile.GetName())
	}

	// Run migration again to verify idempotency.
	s.Require().NoError(migration.Run(dbs))

	// Verify values are unchanged after second run.
	for _, profile := range profiles {
		var operatorKind int32
		err := s.db.DB.QueryRow(s.ctx,
			"SELECT operatorkind FROM compliance_operator_profile_v2 WHERE id = $1", profile.GetId(),
		).Scan(&operatorKind)
		s.Require().NoError(err)
		s.Require().Equal(int32(profile.GetOperatorKind()), operatorKind, "OperatorKind mismatch after idempotent run for profile %s", profile.GetName())
	}
}
