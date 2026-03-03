package m221tom222

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_221_to_m_222_backfill_image_cve_infos_from_image_cves_v2/schema"
	"github.com/stackrox/rox/migrator/types"
	pkgCve "github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

var (
	log             = logging.LoggerForModule()
	readBatchSize   = 10000
	upsertBatchSize = 5000
)

// CVEWithComponent holds joined data from image_cves_v2 and image_component_v2
type CVEWithComponent struct {
	schema.ImageCvesV2
	ComponentName string `gorm:"column:component_name"`
}

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())
	db := database.GormDB

	// Ensure image_cve_infos table exists
	pgutils.CreateTableFromModel(ctx, db, schema.CreateTableImageCveInfosStmt)

	// Build aggregation map: key = composite ID (cve#package#datasource), value = earliest timestamp
	aggregationMap, err := buildAggregationMap(ctx, database)
	if err != nil {
		return errors.Wrap(err, "failed to build aggregation map")
	}

	log.Infof("Aggregated %d unique (cve, package, datasource) combinations", len(aggregationMap))

	// Upsert to image_cve_infos
	if err := upsertImageCVEInfos(ctx, database, aggregationMap); err != nil {
		return errors.Wrap(err, "failed to upsert image CVE infos")
	}

	return nil
}

func buildAggregationMap(ctx context.Context, database *types.Databases) (map[string]*time.Time, error) {
	db := database.GormDB.WithContext(ctx)
	aggregationMap := make(map[string]*time.Time)

	offset := 0
	totalProcessed := 0

	for {
		var batch []CVEWithComponent

		// Fetch image_cves_v2 with JOINed component data
		result := db.Table("image_cves_v2 as cve").
			Select("cve.*, comp.name as component_name").
			Joins("JOIN image_component_v2 as comp ON cve.componentid = comp.id").
			Limit(readBatchSize).
			Offset(offset).
			Scan(&batch)

		if result.Error != nil {
			return nil, errors.Wrapf(result.Error, "failed to fetch batch at offset %d", offset)
		}

		if len(batch) == 0 {
			break
		}

		// Process each CVE in the batch
		for _, cveWithComp := range batch {
			// Deserialize to extract datasource
			cve, err := schema.ConvertImageCVEV2ToProto(&cveWithComp.ImageCvesV2)
			if err != nil {
				log.Warnf("Failed to deserialize CVE %s: %v (skipping)", cveWithComp.CveBaseInfoCve, err)
				continue
			}

			// Build composite ID using ImageCVEInfoID()
			datasource := cve.GetDatasource()
			id := pkgCve.ImageCVEInfoID(
				cve.GetCveBaseInfo().GetCve(),
				cveWithComp.ComponentName,
				datasource,
			)

			// Track MIN timestamp for each unique ID
			timestamp := cveWithComp.CveBaseInfoCreatedAt
			if timestamp != nil {
				if existing, ok := aggregationMap[id]; !ok || timestamp.Before(*existing) {
					aggregationMap[id] = timestamp
				}
			}
		}

		totalProcessed += len(batch)
		log.Infof("Processed %d image_cves_v2 records (batch %d-%d)", totalProcessed, offset, offset+len(batch))

		offset += readBatchSize
	}

	return aggregationMap, nil
}

func upsertImageCVEInfos(ctx context.Context, database *types.Databases, aggregationMap map[string]*time.Time) error {
	db := database.GormDB.WithContext(ctx)

	// Build ImageCVEInfo records
	infos := make([]*schema.ImageCveInfos, 0, len(aggregationMap))

	for id, timestamp := range aggregationMap {
		// Extract CVE from composite ID (first part)
		parts := pgSearch.IDToParts(id)
		cveName := parts[0]

		// Create protobuf object
		proto := &storage.ImageCVEInfo{
			Id:                    id,
			Cve:                   cveName, // Populate indexed cve column
			FirstSystemOccurrence: protocompat.ConvertTimeToTimestampOrNil(timestamp),
			// FixAvailableTimestamp intentionally left nil (user decision: leave for runtime enrichment)
		}

		// Convert to GORM model
		model, err := schema.ConvertImageCVEInfoFromProto(proto)
		if err != nil {
			return errors.Wrapf(err, "failed to convert proto for ID %s", id)
		}

		// Override timestamps from aggregation (since ConvertImageCVEInfoFromProto sets them to nil)
		model.FirstSystemOccurrence = timestamp
		// model.FixAvailableTimestamp remains nil

		infos = append(infos, model)
	}

	// Upsert in batches using smart timestamp merging
	totalUpserted := 0
	for i := 0; i < len(infos); i += upsertBatchSize {
		end := i + upsertBatchSize
		if end > len(infos) {
			end = len(infos)
		}
		batch := infos[i:end]

		// Fetch existing records to perform smart timestamp merging
		ids := make([]string, len(batch))
		for j, info := range batch {
			ids[j] = info.ID
		}

		var existing []schema.ImageCveInfos
		if err := db.Table("image_cve_infos").Where("id IN ?", ids).Find(&existing).Error; err != nil {
			return errors.Wrapf(err, "failed to fetch existing records for batch %d-%d", i, end)
		}

		// Create map of existing records
		existingMap := make(map[string]*schema.ImageCveInfos)
		for idx := range existing {
			existingMap[existing[idx].ID] = &existing[idx]
		}

		// Perform smart timestamp merging: preserve earlier timestamps
		for _, newInfo := range batch {
			if oldInfo, exists := existingMap[newInfo.ID]; exists {
				// Merge timestamps: use earlier values
				if newInfo.FirstSystemOccurrence == nil {
					newInfo.FirstSystemOccurrence = oldInfo.FirstSystemOccurrence
				} else if oldInfo.FirstSystemOccurrence != nil && oldInfo.FirstSystemOccurrence.Before(*newInfo.FirstSystemOccurrence) {
					newInfo.FirstSystemOccurrence = oldInfo.FirstSystemOccurrence
				}

				// FixAvailableTimestamp: preserve old if new is nil
				if newInfo.FixAvailableTimestamp == nil {
					newInfo.FixAvailableTimestamp = oldInfo.FixAvailableTimestamp
				} else if oldInfo.FixAvailableTimestamp != nil && oldInfo.FixAvailableTimestamp.Before(*newInfo.FixAvailableTimestamp) {
					newInfo.FixAvailableTimestamp = oldInfo.FixAvailableTimestamp
				}
			}
		}

		// Upsert batch using ON CONFLICT DO UPDATE
		if err := db.Table("image_cve_infos").Save(batch).Error; err != nil {
			return errors.Wrapf(err, "failed to upsert batch %d-%d", i, end)
		}

		totalUpserted += len(batch)
		log.Infof("Upserted %d/%d image_cve_infos records", totalUpserted, len(infos))
	}

	return nil
}
