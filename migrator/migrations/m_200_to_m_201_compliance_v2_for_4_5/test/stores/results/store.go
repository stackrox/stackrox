package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/generated/storage"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_200_to_m_201_compliance_v2_for_4_5/test/schema"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac/resources"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

var (
	log            = logging.LoggerForModule()
	schema         = oldSchema.ComplianceOperatorCheckResultV2Schema
	targetResource = resources.Compliance
)

type storeType = storage.ComplianceOperatorCheckResultV2

// Store is the interface to interact with the storage for storage.ComplianceOperatorCheckResultV2
type Store interface {
	Upsert(ctx context.Context, obj *storeType) error
	UpsertMany(ctx context.Context, objs []*storeType) error

	Get(ctx context.Context, id string) (*storeType, bool, error)
	GetMany(ctx context.Context, identifiers []string) ([]*storeType, []int, error)
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
	return pgSearch.NewGloballyScopedGenericStore[storeType, *storeType](
		db,
		schema,
		pkGetter,
		insertIntoComplianceOperatorCheckResultV2,
		copyFromComplianceOperatorCheckResultV2,
		metricsSetAcquireDBConnDuration,
		metricsSetPostgresOperationDurationTime,
		targetResource,
	)
}

// region Helper functions

func pkGetter(obj *storeType) string {
	return obj.GetId()
}

func metricsSetPostgresOperationDurationTime(_ time.Time, _ ops.Op) {}

func metricsSetAcquireDBConnDuration(_ time.Time, _ ops.Op) {}

func insertIntoComplianceOperatorCheckResultV2(batch *pgx.Batch, obj *storage.ComplianceOperatorCheckResultV2) error {

	serialized, marshalErr := obj.MarshalVT()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		obj.GetId(),
		obj.GetCheckId(),
		obj.GetCheckName(),
		pgutils.NilOrUUID(obj.GetClusterId()),
		obj.GetStatus(),
		obj.GetSeverity(),
		protocompat.NilOrTime(obj.GetCreatedTime()),
		obj.GetScanName(),
		obj.GetScanConfigName(),
		serialized,
	}

	finalStr := "INSERT INTO compliance_operator_check_result_v2 (Id, CheckId, CheckName, ClusterId, Status, Severity, CreatedTime, ScanName, ScanConfigName, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, CheckId = EXCLUDED.CheckId, CheckName = EXCLUDED.CheckName, ClusterId = EXCLUDED.ClusterId, Status = EXCLUDED.Status, Severity = EXCLUDED.Severity, CreatedTime = EXCLUDED.CreatedTime, ScanName = EXCLUDED.ScanName, ScanConfigName = EXCLUDED.ScanConfigName, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

func copyFromComplianceOperatorCheckResultV2(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, objs ...*storage.ComplianceOperatorCheckResultV2) error {
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
		"checkid",
		"checkname",
		"clusterid",
		"status",
		"severity",
		"createdtime",
		"scanname",
		"scanconfigname",
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
			obj.GetCheckId(),
			obj.GetCheckName(),
			pgutils.NilOrUUID(obj.GetClusterId()),
			obj.GetStatus(),
			obj.GetSeverity(),
			protocompat.NilOrTime(obj.GetCreatedTime()),
			obj.GetScanName(),
			obj.GetScanConfigName(),
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

			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"compliance_operator_check_result_v2"}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return nil
}

// endregion Helper functions
