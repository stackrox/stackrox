package m194tom195

import (
	"context"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_194_to_m_195_vuln_request_global_scope/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"gorm.io/gorm/clause"
)

var (
	batchSize = 2000
	log       = logging.LoggerForModule()
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())
	pgutils.CreateTableFromModel(ctx, database.GormDB, schema.CreateTableVulnerabilityRequestsStmt)

	now := timestamp.TimestampNow()
	return createVulnRequests(ctx, database, now)
}

func createVulnRequests(ctx context.Context, database *types.Databases, now *timestamp.Timestamp) error {
	db := database.GormDB.WithContext(ctx).Table(schema.VulnerabilityRequestsTableName)
	query := database.GormDB.WithContext(ctx).Table(schema.VulnerabilityRequestsTableName).Select("serialized")
	rows, err := query.Rows()
	if err != nil {
		return errors.Wrapf(err, "failed to query table %s", schema.VulnerabilityRequestsTableName)
	}
	defer func() { _ = rows.Close() }()

	var convertedObjs []*schema.VulnerabilityRequests
	var count int
	for rows.Next() {
		var obj schema.VulnerabilityRequests
		if err = query.ScanRows(rows, &obj); err != nil {
			return errors.Wrap(err, "failed to scan image_cves table rows")
		}
		proto, err := schema.ConvertVulnerabilityRequestToProto(&obj)
		if err != nil {
			return errors.Wrapf(err, "failed to convert %+v to proto", proto)
		}

		// Update the representation of global scope per the new way.
		if proto.GetScope().GetGlobalScope() == nil {
			continue
		}
		proto.Scope = &storage.VulnerabilityRequest_Scope{
			Info: &storage.VulnerabilityRequest_Scope_ImageScope{
				ImageScope: &storage.VulnerabilityRequest_Scope_Image{
					Registry: ".*",
					Remote:   ".*",
					Tag:      ".*",
				},
			},
		}

		converted, err := schema.ConvertVulnerabilityRequestFromProto(proto)
		if err != nil {
			return errors.Wrapf(err, "failed to convert from proto %+v", proto)
		}
		convertedObjs = append(convertedObjs, converted)
		count++

		if len(convertedObjs) == batchSize {
			if err = db.
				Clauses(clause.OnConflict{UpdateAll: true}).
				Model(schema.CreateTableVulnerabilityRequestsStmt.GormModel).
				Create(&convertedObjs).Error; err != nil {
				return errors.Wrapf(err, "failed to upsert converted %d objects after %d upserted", len(convertedObjs), count-len(convertedObjs))
			}
			convertedObjs = convertedObjs[:0]
		}
	}
	if rows.Err() != nil {
		return errors.Wrapf(rows.Err(), "failed to get rows for %s", schema.VulnerabilityRequestsTableName)
	}

	if len(convertedObjs) > 0 {
		if err = db.
			Clauses(clause.OnConflict{UpdateAll: true}).
			Model(schema.CreateTableVulnerabilityRequestsStmt.GormModel).
			Create(&convertedObjs).Error; err != nil {
			return errors.Wrapf(err, "failed to upsert last %d objects", len(convertedObjs))
		}
	}
	log.Infof("Updated %d global scope vulnerability exceptions", count)
	return nil
}
