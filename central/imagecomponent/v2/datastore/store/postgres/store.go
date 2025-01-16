<<<<<<< HEAD
<<<<<<< HEAD
>>>>>>> de967dabd7 (X-Smart-Squash: Squashed 13 commits:)
=======
=======
>>>>>>> 8cca5a6f70 (X-Smart-Squash: Squashed 4 commits:)
>>>>>>> 65c1047c5e (X-Smart-Squash: Squashed 4 commits:)
=======
>>>>>>> 367b120c6d (X-Smart-Squash: Squashed 27 commits:)
package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
<<<<<<< HEAD
=======
	"github.com/stackrox/rox/pkg/protocompat"
>>>>>>> 9a652b9c15 (X-Smart-Squash: Squashed 10 commits:)
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"gorm.io/gorm"
)

const (
	baseTable = "image_component_v2"
	storeName = "ImageComponentV2"
)

var (
	log            = logging.LoggerForModule()
	schema         = pkgSchema.ImageComponentV2Schema
	targetResource = resources.Image
)

type storeType = storage.ImageComponentV2

// Store is the interface to interact with the storage for storage.ImageComponentV2
type Store interface {
	Upsert(ctx context.Context, obj *storeType) error
	UpsertMany(ctx context.Context, objs []*storeType) error
	Delete(ctx context.Context, id string) error
	DeleteByQuery(ctx context.Context, q *v1.Query) ([]string, error)
	DeleteMany(ctx context.Context, identifiers []string) error
	PruneMany(ctx context.Context, identifiers []string) error

	Count(ctx context.Context, q *v1.Query) (int, error)
	Exists(ctx context.Context, id string) (bool, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)

	Get(ctx context.Context, id string) (*storeType, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storeType, error)
	GetMany(ctx context.Context, identifiers []string) ([]*storeType, []int, error)
	GetIDs(ctx context.Context) ([]string, error)

	Walk(ctx context.Context, fn func(obj *storeType) error) error
	WalkByQuery(ctx context.Context, query *v1.Query, fn func(obj *storeType) error) error
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
	return pgSearch.NewGenericStore[storeType, *storeType](
		db,
		schema,
		pkGetter,
		insertIntoImageComponentV2,
		copyFromImageComponentV2,
		metricsSetAcquireDBConnDuration,
		metricsSetPostgresOperationDurationTime,
		pgSearch.GloballyScopedUpsertChecker[storeType, *storeType](targetResource),
		targetResource,
	)
}

// region Helper functions

func pkGetter(obj *storeType) string {
	return obj.GetId()
}

func metricsSetPostgresOperationDurationTime(start time.Time, op ops.Op) {
	metrics.SetPostgresOperationDurationTime(start, op, storeName)
}

func metricsSetAcquireDBConnDuration(start time.Time, op ops.Op) {
	metrics.SetAcquireDBConnDuration(start, op, storeName)
}

func insertIntoImageComponentV2(batch *pgx.Batch, obj *storage.ImageComponentV2) error {

	serialized, marshalErr := obj.MarshalVT()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		obj.GetId(),
		obj.GetName(),
		obj.GetVersion(),
		obj.GetPriority(),
		obj.GetSource(),
		obj.GetRiskScore(),
		obj.GetTopCvss(),
		obj.GetOperatingSystem(),
		obj.GetImageId(),
		obj.GetLocation(),
		serialized,
	}

	finalStr := "INSERT INTO image_component_v2 (Id, Name, Version, Priority, Source, RiskScore, TopCvss, OperatingSystem, ImageId, Location, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, Name = EXCLUDED.Name, Version = EXCLUDED.Version, Priority = EXCLUDED.Priority, Source = EXCLUDED.Source, RiskScore = EXCLUDED.RiskScore, TopCvss = EXCLUDED.TopCvss, OperatingSystem = EXCLUDED.OperatingSystem, ImageId = EXCLUDED.ImageId, Location = EXCLUDED.Location, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> 90f2c751ff (X-Smart-Squash: Squashed 10 commits:)
=======
>>>>>>> 399dafd1bb (X-Smart-Squash: Squashed 10 commits:)
=======
	var query string

	for childIndex, child := range obj.GetCves() {
		if err := insertIntoImageComponentV2Cves(batch, child, obj.GetId(), childIndex); err != nil {
			return err
		}
	}

	query = "delete from image_component_v2_cves where image_component_v2_Id = $1 AND idx >= $2"
	batch.Queue(query, obj.GetId(), len(obj.GetCves()))
	return nil
}

func insertIntoImageComponentV2Cves(batch *pgx.Batch, obj *storage.ImageCVEV2, imageComponentV2ID string, idx int) error {

	values := []interface{}{
		// parent primary keys start
		imageComponentV2ID,
		idx,
		obj.GetId(),
		obj.GetImageId(),
		obj.GetCveBaseInfo().GetCve(),
		protocompat.NilOrTime(obj.GetCveBaseInfo().GetPublishedOn()),
		protocompat.NilOrTime(obj.GetCveBaseInfo().GetCreatedAt()),
<<<<<<< HEAD
<<<<<<< HEAD
		obj.GetCveBaseInfo().GetEpss().GetEpssProbability(),
=======
<<<<<<< HEAD
<<<<<<< HEAD
		obj.GetCveBaseInfo().GetEpss().GetEpssProbability(),
=======
>>>>>>> baec3e8b51 (X-Smart-Squash: Squashed 9 commits:)
=======
		obj.GetCveBaseInfo().GetEpss().GetEpssProbability(),
>>>>>>> 4e3ee609f4 (X-Smart-Squash: Squashed 4 commits:)
>>>>>>> 8cca5a6f70 (X-Smart-Squash: Squashed 4 commits:)
=======
		obj.GetCveBaseInfo().GetEpss().GetEpssProbability(),
>>>>>>> 399dafd1bb (X-Smart-Squash: Squashed 10 commits:)
		obj.GetOperatingSystem(),
		obj.GetCvss(),
		obj.GetSeverity(),
		obj.GetImpactScore(),
		obj.GetNvdcvss(),
		protocompat.NilOrTime(obj.GetFirstImageOccurrence()),
		obj.GetState(),
		obj.GetIsFixable(),
		obj.GetFixedBy(),
	}

<<<<<<< HEAD
<<<<<<< HEAD
	finalStr := "INSERT INTO image_component_v2_cves (image_component_v2_Id, idx, Id, ImageId, CveBaseInfo_Cve, CveBaseInfo_PublishedOn, CveBaseInfo_CreatedAt, CveBaseInfo_Epss_EpssProbability, OperatingSystem, Cvss, Severity, ImpactScore, Nvdcvss, FirstImageOccurrence, State, IsFixable, FixedBy) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17) ON CONFLICT(image_component_v2_Id, idx) DO UPDATE SET image_component_v2_Id = EXCLUDED.image_component_v2_Id, idx = EXCLUDED.idx, Id = EXCLUDED.Id, ImageId = EXCLUDED.ImageId, CveBaseInfo_Cve = EXCLUDED.CveBaseInfo_Cve, CveBaseInfo_PublishedOn = EXCLUDED.CveBaseInfo_PublishedOn, CveBaseInfo_CreatedAt = EXCLUDED.CveBaseInfo_CreatedAt, CveBaseInfo_Epss_EpssProbability = EXCLUDED.CveBaseInfo_Epss_EpssProbability, OperatingSystem = EXCLUDED.OperatingSystem, Cvss = EXCLUDED.Cvss, Severity = EXCLUDED.Severity, ImpactScore = EXCLUDED.ImpactScore, Nvdcvss = EXCLUDED.Nvdcvss, FirstImageOccurrence = EXCLUDED.FirstImageOccurrence, State = EXCLUDED.State, IsFixable = EXCLUDED.IsFixable, FixedBy = EXCLUDED.FixedBy"
=======
<<<<<<< HEAD
<<<<<<< HEAD
	finalStr := "INSERT INTO image_component_v2_cves (image_component_v2_Id, idx, Id, ImageId, CveBaseInfo_Cve, CveBaseInfo_PublishedOn, CveBaseInfo_CreatedAt, CveBaseInfo_Epss_EpssProbability, OperatingSystem, Cvss, Severity, ImpactScore, Nvdcvss, FirstImageOccurrence, State, IsFixable, FixedBy) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17) ON CONFLICT(image_component_v2_Id, idx) DO UPDATE SET image_component_v2_Id = EXCLUDED.image_component_v2_Id, idx = EXCLUDED.idx, Id = EXCLUDED.Id, ImageId = EXCLUDED.ImageId, CveBaseInfo_Cve = EXCLUDED.CveBaseInfo_Cve, CveBaseInfo_PublishedOn = EXCLUDED.CveBaseInfo_PublishedOn, CveBaseInfo_CreatedAt = EXCLUDED.CveBaseInfo_CreatedAt, CveBaseInfo_Epss_EpssProbability = EXCLUDED.CveBaseInfo_Epss_EpssProbability, OperatingSystem = EXCLUDED.OperatingSystem, Cvss = EXCLUDED.Cvss, Severity = EXCLUDED.Severity, ImpactScore = EXCLUDED.ImpactScore, Nvdcvss = EXCLUDED.Nvdcvss, FirstImageOccurrence = EXCLUDED.FirstImageOccurrence, State = EXCLUDED.State, IsFixable = EXCLUDED.IsFixable, FixedBy = EXCLUDED.FixedBy"
=======
	finalStr := "INSERT INTO image_component_v2_cves (image_component_v2_Id, idx, Id, ImageId, CveBaseInfo_Cve, CveBaseInfo_PublishedOn, CveBaseInfo_CreatedAt, OperatingSystem, Cvss, Severity, ImpactScore, Nvdcvss, FirstImageOccurrence, State, IsFixable, FixedBy) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16) ON CONFLICT(image_component_v2_Id, idx) DO UPDATE SET image_component_v2_Id = EXCLUDED.image_component_v2_Id, idx = EXCLUDED.idx, Id = EXCLUDED.Id, ImageId = EXCLUDED.ImageId, CveBaseInfo_Cve = EXCLUDED.CveBaseInfo_Cve, CveBaseInfo_PublishedOn = EXCLUDED.CveBaseInfo_PublishedOn, CveBaseInfo_CreatedAt = EXCLUDED.CveBaseInfo_CreatedAt, OperatingSystem = EXCLUDED.OperatingSystem, Cvss = EXCLUDED.Cvss, Severity = EXCLUDED.Severity, ImpactScore = EXCLUDED.ImpactScore, Nvdcvss = EXCLUDED.Nvdcvss, FirstImageOccurrence = EXCLUDED.FirstImageOccurrence, State = EXCLUDED.State, IsFixable = EXCLUDED.IsFixable, FixedBy = EXCLUDED.FixedBy"
>>>>>>> baec3e8b51 (X-Smart-Squash: Squashed 9 commits:)
=======
	finalStr := "INSERT INTO image_component_v2_cves (image_component_v2_Id, idx, Id, ImageId, CveBaseInfo_Cve, CveBaseInfo_PublishedOn, CveBaseInfo_CreatedAt, CveBaseInfo_Epss_EpssProbability, OperatingSystem, Cvss, Severity, ImpactScore, Nvdcvss, FirstImageOccurrence, State, IsFixable, FixedBy) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17) ON CONFLICT(image_component_v2_Id, idx) DO UPDATE SET image_component_v2_Id = EXCLUDED.image_component_v2_Id, idx = EXCLUDED.idx, Id = EXCLUDED.Id, ImageId = EXCLUDED.ImageId, CveBaseInfo_Cve = EXCLUDED.CveBaseInfo_Cve, CveBaseInfo_PublishedOn = EXCLUDED.CveBaseInfo_PublishedOn, CveBaseInfo_CreatedAt = EXCLUDED.CveBaseInfo_CreatedAt, CveBaseInfo_Epss_EpssProbability = EXCLUDED.CveBaseInfo_Epss_EpssProbability, OperatingSystem = EXCLUDED.OperatingSystem, Cvss = EXCLUDED.Cvss, Severity = EXCLUDED.Severity, ImpactScore = EXCLUDED.ImpactScore, Nvdcvss = EXCLUDED.Nvdcvss, FirstImageOccurrence = EXCLUDED.FirstImageOccurrence, State = EXCLUDED.State, IsFixable = EXCLUDED.IsFixable, FixedBy = EXCLUDED.FixedBy"
>>>>>>> 4e3ee609f4 (X-Smart-Squash: Squashed 4 commits:)
>>>>>>> 8cca5a6f70 (X-Smart-Squash: Squashed 4 commits:)
	batch.Queue(finalStr, values...)

>>>>>>> 9a652b9c15 (X-Smart-Squash: Squashed 10 commits:)
<<<<<<< HEAD
=======
>>>>>>> 8ad8e67206 (X-Smart-Squash: Squashed 16 commits:)
=======
>>>>>>> 90f2c751ff (X-Smart-Squash: Squashed 10 commits:)
=======
>>>>>>> 367b120c6d (X-Smart-Squash: Squashed 27 commits:)
=======
	finalStr := "INSERT INTO image_component_v2_cves (image_component_v2_Id, idx, Id, ImageId, CveBaseInfo_Cve, CveBaseInfo_PublishedOn, CveBaseInfo_CreatedAt, CveBaseInfo_Epss_EpssProbability, OperatingSystem, Cvss, Severity, ImpactScore, Nvdcvss, FirstImageOccurrence, State, IsFixable, FixedBy) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17) ON CONFLICT(image_component_v2_Id, idx) DO UPDATE SET image_component_v2_Id = EXCLUDED.image_component_v2_Id, idx = EXCLUDED.idx, Id = EXCLUDED.Id, ImageId = EXCLUDED.ImageId, CveBaseInfo_Cve = EXCLUDED.CveBaseInfo_Cve, CveBaseInfo_PublishedOn = EXCLUDED.CveBaseInfo_PublishedOn, CveBaseInfo_CreatedAt = EXCLUDED.CveBaseInfo_CreatedAt, CveBaseInfo_Epss_EpssProbability = EXCLUDED.CveBaseInfo_Epss_EpssProbability, OperatingSystem = EXCLUDED.OperatingSystem, Cvss = EXCLUDED.Cvss, Severity = EXCLUDED.Severity, ImpactScore = EXCLUDED.ImpactScore, Nvdcvss = EXCLUDED.Nvdcvss, FirstImageOccurrence = EXCLUDED.FirstImageOccurrence, State = EXCLUDED.State, IsFixable = EXCLUDED.IsFixable, FixedBy = EXCLUDED.FixedBy"
	batch.Queue(finalStr, values...)

>>>>>>> 9a652b9c15 (X-Smart-Squash: Squashed 10 commits:)
>>>>>>> 399dafd1bb (X-Smart-Squash: Squashed 10 commits:)
	return nil
}

func copyFromImageComponentV2(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, objs ...*storage.ImageComponentV2) error {
	batchSize := pgSearch.MaxBatchSize
	if len(objs) < batchSize {
		batchSize = len(objs)
	}
	inputRows := make([][]interface{}, 0, batchSize)

	// This is a copy so first we must delete the rows and re-add them
	// Which is essentially the desired behaviour of an upsert.
	deletes := make([]string, 0, batchSize)

	copyCols := []string{
		"id",
		"name",
		"version",
		"priority",
		"source",
		"riskscore",
		"topcvss",
		"operatingsystem",
		"imageid",
		"location",
		"serialized",
	}

	for idx, obj := range objs {
		// Todo: ROX-9499 Figure out how to more cleanly template around this issue.
		log.Debugf("This is here for now because there is an issue with pods_TerminatedInstances where the obj "+
			"in the loop is not used as it only consists of the parent ID and the index.  Putting this here as a stop gap "+
			"to simply use the object.  %s", obj)

		serialized, marshalErr := obj.MarshalVT()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			obj.GetName(),
			obj.GetVersion(),
			obj.GetPriority(),
			obj.GetSource(),
			obj.GetRiskScore(),
			obj.GetTopCvss(),
			obj.GetOperatingSystem(),
			obj.GetImageId(),
			obj.GetLocation(),
			serialized,
		})

		// Add the ID to be deleted.
		deletes = append(deletes, obj.GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			if err := s.DeleteMany(ctx, deletes); err != nil {
				return err
			}
			// clear the inserts and vals for the next batch
			deletes = deletes[:0]

			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"image_component_v2"}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> 90f2c751ff (X-Smart-Squash: Squashed 10 commits:)
=======
>>>>>>> 399dafd1bb (X-Smart-Squash: Squashed 10 commits:)
=======
	for idx, obj := range objs {
		_ = idx // idx may or may not be used depending on how nested we are, so avoid compile-time errors.

		if err := copyFromImageComponentV2Cves(ctx, s, tx, obj.GetId(), obj.GetCves()...); err != nil {
			return err
		}
	}

	return nil
}

func copyFromImageComponentV2Cves(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, imageComponentV2ID string, objs ...*storage.ImageCVEV2) error {
	batchSize := pgSearch.MaxBatchSize
	if len(objs) < batchSize {
		batchSize = len(objs)
	}
	inputRows := make([][]interface{}, 0, batchSize)

	copyCols := []string{
		"image_component_v2_id",
		"idx",
		"id",
		"imageid",
		"cvebaseinfo_cve",
		"cvebaseinfo_publishedon",
		"cvebaseinfo_createdat",
<<<<<<< HEAD
<<<<<<< HEAD
		"cvebaseinfo_epss_epssprobability",
=======
<<<<<<< HEAD
<<<<<<< HEAD
		"cvebaseinfo_epss_epssprobability",
=======
>>>>>>> baec3e8b51 (X-Smart-Squash: Squashed 9 commits:)
=======
		"cvebaseinfo_epss_epssprobability",
>>>>>>> 4e3ee609f4 (X-Smart-Squash: Squashed 4 commits:)
>>>>>>> 8cca5a6f70 (X-Smart-Squash: Squashed 4 commits:)
=======
		"cvebaseinfo_epss_epssprobability",
>>>>>>> 399dafd1bb (X-Smart-Squash: Squashed 10 commits:)
		"operatingsystem",
		"cvss",
		"severity",
		"impactscore",
		"nvdcvss",
		"firstimageoccurrence",
		"state",
		"isfixable",
		"fixedby",
	}

	for idx, obj := range objs {
		// Todo: ROX-9499 Figure out how to more cleanly template around this issue.
		log.Debugf("This is here for now because there is an issue with pods_TerminatedInstances where the obj "+
			"in the loop is not used as it only consists of the parent ID and the index.  Putting this here as a stop gap "+
			"to simply use the object.  %s", obj)

		inputRows = append(inputRows, []interface{}{
			imageComponentV2ID,
			idx,
			obj.GetId(),
			obj.GetImageId(),
			obj.GetCveBaseInfo().GetCve(),
			protocompat.NilOrTime(obj.GetCveBaseInfo().GetPublishedOn()),
			protocompat.NilOrTime(obj.GetCveBaseInfo().GetCreatedAt()),
<<<<<<< HEAD
<<<<<<< HEAD
			obj.GetCveBaseInfo().GetEpss().GetEpssProbability(),
=======
<<<<<<< HEAD
<<<<<<< HEAD
			obj.GetCveBaseInfo().GetEpss().GetEpssProbability(),
=======
>>>>>>> baec3e8b51 (X-Smart-Squash: Squashed 9 commits:)
=======
			obj.GetCveBaseInfo().GetEpss().GetEpssProbability(),
>>>>>>> 4e3ee609f4 (X-Smart-Squash: Squashed 4 commits:)
>>>>>>> 8cca5a6f70 (X-Smart-Squash: Squashed 4 commits:)
=======
			obj.GetCveBaseInfo().GetEpss().GetEpssProbability(),
>>>>>>> 399dafd1bb (X-Smart-Squash: Squashed 10 commits:)
			obj.GetOperatingSystem(),
			obj.GetCvss(),
			obj.GetSeverity(),
			obj.GetImpactScore(),
			obj.GetNvdcvss(),
			protocompat.NilOrTime(obj.GetFirstImageOccurrence()),
			obj.GetState(),
			obj.GetIsFixable(),
			obj.GetFixedBy(),
		})

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"image_component_v2_cves"}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

>>>>>>> 9a652b9c15 (X-Smart-Squash: Squashed 10 commits:)
<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> 8ad8e67206 (X-Smart-Squash: Squashed 16 commits:)
=======
>>>>>>> 90f2c751ff (X-Smart-Squash: Squashed 10 commits:)
=======
>>>>>>> 367b120c6d (X-Smart-Squash: Squashed 27 commits:)
=======
>>>>>>> 399dafd1bb (X-Smart-Squash: Squashed 10 commits:)
	return nil
}

// endregion Helper functions

// region Used for testing

// CreateTableAndNewStore returns a new Store instance for testing.
func CreateTableAndNewStore(ctx context.Context, db postgres.DB, gormDB *gorm.DB) Store {
	pkgSchema.ApplySchemaForTable(ctx, gormDB, baseTable)
	return New(db)
}

// Destroy drops the tables associated with the target object type.
func Destroy(ctx context.Context, db postgres.DB) {
	dropTableImageComponentV2(ctx, db)
}

func dropTableImageComponentV2(ctx context.Context, db postgres.DB) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS image_component_v2 CASCADE")
<<<<<<< HEAD
=======
	dropTableImageComponentV2Cves(ctx, db)

}

func dropTableImageComponentV2Cves(ctx context.Context, db postgres.DB) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS image_component_v2_cves CASCADE")
>>>>>>> 9a652b9c15 (X-Smart-Squash: Squashed 10 commits:)

}

// endregion Used for testing
