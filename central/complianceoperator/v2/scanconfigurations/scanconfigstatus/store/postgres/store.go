// Code generated by pg-bindings generator. DO NOT EDIT.

package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"gorm.io/gorm"
)

const (
	baseTable = "compliance_operator_cluster_scan_config_statuses"
	storeName = "ComplianceOperatorClusterScanConfigStatus"
)

var (
	log            = logging.LoggerForModule()
	schema         = pkgSchema.ComplianceOperatorClusterScanConfigStatusesSchema
	targetResource = resources.Compliance
)

type (
	storeType = storage.ComplianceOperatorClusterScanConfigStatus
	callback  = func(obj *storeType) error
)

// Store is the interface to interact with the storage for storage.ComplianceOperatorClusterScanConfigStatus
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
	// Deprecated: use GetByQueryFn instead
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storeType, error)
	GetByQueryFn(ctx context.Context, query *v1.Query, fn callback) error
	GetMany(ctx context.Context, identifiers []string) ([]*storeType, []int, error)
	GetIDs(ctx context.Context) ([]string, error)

	Walk(ctx context.Context, fn callback) error
	WalkByQuery(ctx context.Context, query *v1.Query, fn callback) error
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
	return pgSearch.NewGenericStore[storeType, *storeType](
		db,
		schema,
		pkGetter,
		insertIntoComplianceOperatorClusterScanConfigStatuses,
		copyFromComplianceOperatorClusterScanConfigStatuses,
		metricsSetAcquireDBConnDuration,
		metricsSetPostgresOperationDurationTime,
		isUpsertAllowed,
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
func isUpsertAllowed(ctx context.Context, objs ...*storeType) error {
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_WRITE_ACCESS).Resource(targetResource)
	if scopeChecker.IsAllowed() {
		return nil
	}
	var deniedIDs []string
	for _, obj := range objs {
		subScopeChecker := scopeChecker.ClusterID(obj.GetClusterId())
		if !subScopeChecker.IsAllowed() {
			deniedIDs = append(deniedIDs, obj.GetId())
		}
	}
	if len(deniedIDs) != 0 {
		return errors.Wrapf(sac.ErrResourceAccessDenied, "modifying complianceOperatorClusterScanConfigStatuss with IDs [%s] was denied", strings.Join(deniedIDs, ", "))
	}
	return nil
}

func insertIntoComplianceOperatorClusterScanConfigStatuses(batch *pgx.Batch, obj *storage.ComplianceOperatorClusterScanConfigStatus) error {

	serialized, marshalErr := obj.MarshalVT()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		pgutils.NilOrUUID(obj.GetId()),
		pgutils.NilOrUUID(obj.GetClusterId()),
		pgutils.NilOrUUID(obj.GetScanConfigId()),
		protocompat.NilOrTime(obj.GetLastUpdatedTime()),
		serialized,
	}

	finalStr := "INSERT INTO compliance_operator_cluster_scan_config_statuses (Id, ClusterId, ScanConfigId, LastUpdatedTime, serialized) VALUES($1, $2, $3, $4, $5) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, ClusterId = EXCLUDED.ClusterId, ScanConfigId = EXCLUDED.ScanConfigId, LastUpdatedTime = EXCLUDED.LastUpdatedTime, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

func copyFromComplianceOperatorClusterScanConfigStatuses(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, objs ...*storage.ComplianceOperatorClusterScanConfigStatus) error {
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
		"clusterid",
		"scanconfigid",
		"lastupdatedtime",
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
			pgutils.NilOrUUID(obj.GetId()),
			pgutils.NilOrUUID(obj.GetClusterId()),
			pgutils.NilOrUUID(obj.GetScanConfigId()),
			protocompat.NilOrTime(obj.GetLastUpdatedTime()),
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

			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"compliance_operator_cluster_scan_config_statuses"}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

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
	dropTableComplianceOperatorClusterScanConfigStatuses(ctx, db)
}

func dropTableComplianceOperatorClusterScanConfigStatuses(ctx context.Context, db postgres.DB) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS compliance_operator_cluster_scan_config_statuses CASCADE")

}

// endregion Used for testing
