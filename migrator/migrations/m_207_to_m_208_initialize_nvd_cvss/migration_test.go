//go:build sql_integration

package m207tom208

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/generated/storage"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_207_to_m_208_initialize_nvd_cvss/schema/old"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
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
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), oldSchema.CreateTableImageCvesStmt)
}

func (s *migrationTestSuite) TestMigration() {
	batchSize := 20
	inputRows := make([][]interface{}, 0, batchSize)
	cves := []*storage.ImageCVE{
		getTestImageCVE("cve-2023-123", true, false),
		getTestImageCVE("cve-2023-124", true, false),
		getTestImageCVE("cve-2023-125", true, false),
		getTestImageCVE("cve-2023-126", true, true),
		getTestImageCVE("cve-2023-127", true, true),
		getTestImageCVE("cve-2023-128", true, true),
		getTestImageCVE("cve-2023-129", false, false),
		getTestImageCVE("cve-2023-131", false, false),
		getTestImageCVE("cve-2023-132", false, false),
		getTestImageCVE("cve-2023-134", true, false),
		getTestImageCVE("cve-2023-135", true, false),
		getTestImageCVE("cve-2023-136", true, false),
		getTestImageCVE("cve-2023-137", true, false),
		func() *storage.ImageCVE {
			cve := getTestImageCVE("cve-2023-138", true, false)
			cve.SnoozeExpiry = nil
			return cve
		}(),
	}

	copyCols := []string{
		"id",
		"cvebaseinfo_cve",
		"cvebaseinfo_publishedon",
		"cvebaseinfo_createdat",
		"operatingsystem",
		"cvss",
		"severity",
		"impactscore",
		"snoozed",
		"snoozeexpiry",
		"serialized",
	}

	for _, obj := range cves {
		serialized, marshalErr := obj.MarshalVT()
		s.Require().NoError(marshalErr)

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			obj.GetCveBaseInfo().GetCve(),
			protocompat.NilOrTime(obj.GetCveBaseInfo().GetPublishedOn()),
			protocompat.NilOrTime(obj.GetCveBaseInfo().GetCreatedAt()),
			obj.GetOperatingSystem(),
			obj.GetCvss(),
			obj.GetSeverity(),
			obj.GetImpactScore(),
			obj.GetSnoozed(),
			protocompat.NilOrTime(obj.GetSnoozeExpiry()),
			serialized,
		})
	}
	tx, err := s.db.DB.Begin(s.ctx)
	s.Require().NoError(err)

	_, err = tx.CopyFrom(s.ctx, pgx.Identifier{"image_cves"}, copyCols, pgx.CopyFromRows(inputRows))
	s.Require().NoError(err)

	s.Require().NoError(tx.Commit(s.ctx))

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	var n int
	err = s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM image_cves WHERE nvdcvss IS NULL;").Scan(&n)
	s.Require().NoError(err)
	s.Require().Equal(0, n)

	err = s.db.DB.QueryRow(s.ctx, "SELECT COUNT(*) FROM image_cves WHERE nvdcvss IS NOT NULL;").Scan(&n)
	s.Require().NoError(err)
	s.Require().Equal(len(cves), n)
}

func getTestImageCVE(cve string, snoozed, expired bool) *storage.ImageCVE {
	return &storage.ImageCVE{
		Id: cve,
		CveBaseInfo: &storage.CVEInfo{
			Cve: cve,
		},
		Snoozed: snoozed,
		SnoozeExpiry: func() *protocompat.Timestamp {
			now := time.Now()
			if expired {
				now = now.Add(-(7 * 24 * time.Hour))
			} else {
				now = now.Add(7 * 24 * time.Hour)
			}
			return protoconv.ConvertTimeToTimestamp(now)
		}(),
	}
}
