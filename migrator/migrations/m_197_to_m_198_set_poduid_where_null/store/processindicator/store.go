package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	migrationSchema "github.com/stackrox/rox/migrator/migrations/m_197_to_m_198_set_poduid_where_null/schema/process_indicators"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

const (
	baseTable = "process_indicators"
	storeName = "ProcessIndicator"

	// using copyFrom, we may not even want to batch.  It would probably be simpler
	// to deal with failures if we just sent it all.  Something to think about as we
	// proceed and move into more e2e and larger performance testing
	batchSize = 10000
)

var (
	log            = logging.LoggerForModule()
	schema         = migrationSchema.ProcessIndicatorsSchema
	targetResource = resources.DeploymentExtension
)

type storeType = storage.ProcessIndicator

// Store is the interface to interact with the storage for storage.ProcessIndicator
type Store interface {
	Upsert(ctx context.Context, obj *storeType) error
	UpsertMany(ctx context.Context, objs []*storeType) error
	Delete(ctx context.Context, id string) error
	DeleteByQuery(ctx context.Context, q *v1.Query) error
	DeleteMany(ctx context.Context, identifiers []string) error

	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)

	Get(ctx context.Context, id string) (*storeType, bool, error)
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storeType, error)
	GetMany(ctx context.Context, identifiers []string) ([]*storeType, []int, error)
	GetIDs(ctx context.Context) ([]string, error)

	Walk(ctx context.Context, fn func(obj *storeType) error) error
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
	return pgSearch.NewGenericStore[storeType, *storeType](
		db,
		schema,
		pkGetter,
		insertIntoProcessIndicators,
		copyFromProcessIndicators,
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

func metricsSetPostgresOperationDurationTime(_ time.Time, _ ops.Op) {
}

func metricsSetAcquireDBConnDuration(_ time.Time, _ ops.Op) {
}
func isUpsertAllowed(ctx context.Context, objs ...*storeType) error {
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_WRITE_ACCESS).Resource(targetResource)
	if scopeChecker.IsAllowed() {
		return nil
	}
	var deniedIDs []string
	for _, obj := range objs {
		subScopeChecker := scopeChecker.ClusterID(obj.GetClusterId()).Namespace(obj.GetNamespace())
		if !subScopeChecker.IsAllowed() {
			deniedIDs = append(deniedIDs, obj.GetId())
		}
	}
	if len(deniedIDs) != 0 {
		return errors.Wrapf(sac.ErrResourceAccessDenied, "modifying processIndicators with IDs [%s] was denied", strings.Join(deniedIDs, ", "))
	}
	return nil
}

func insertIntoProcessIndicators(batch *pgx.Batch, obj *storage.ProcessIndicator) error {

	serialized, marshalErr := obj.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		pgutils.NilOrUUID(obj.GetId()),
		pgutils.NilOrUUID(obj.GetDeploymentId()),
		obj.GetContainerName(),
		obj.GetPodId(),
		pgutils.NilOrUUID(obj.GetPodUid()),
		obj.GetSignal().GetContainerId(),
		pgutils.NilOrTime(obj.GetSignal().GetTime()),
		obj.GetSignal().GetName(),
		obj.GetSignal().GetArgs(),
		obj.GetSignal().GetExecFilePath(),
		obj.GetSignal().GetUid(),
		pgutils.NilOrUUID(obj.GetClusterId()),
		obj.GetNamespace(),
		serialized,
	}

	finalStr := "INSERT INTO process_indicators (Id, DeploymentId, ContainerName, PodId, PodUid, Signal_ContainerId, Signal_Time, Signal_Name, Signal_Args, Signal_ExecFilePath, Signal_Uid, ClusterId, Namespace, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, DeploymentId = EXCLUDED.DeploymentId, ContainerName = EXCLUDED.ContainerName, PodId = EXCLUDED.PodId, PodUid = EXCLUDED.PodUid, Signal_ContainerId = EXCLUDED.Signal_ContainerId, Signal_Time = EXCLUDED.Signal_Time, Signal_Name = EXCLUDED.Signal_Name, Signal_Args = EXCLUDED.Signal_Args, Signal_ExecFilePath = EXCLUDED.Signal_ExecFilePath, Signal_Uid = EXCLUDED.Signal_Uid, ClusterId = EXCLUDED.ClusterId, Namespace = EXCLUDED.Namespace, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

func copyFromProcessIndicators(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, objs ...*storage.ProcessIndicator) error {
	inputRows := make([][]interface{}, 0, batchSize)

	// This is a copy so first we must delete the rows and re-add them
	// Which is essentially the desired behaviour of an upsert.
	deletes := make([]string, 0, batchSize)

	copyCols := []string{
		"id",
		"deploymentid",
		"containername",
		"podid",
		"poduid",
		"signal_containerid",
		"signal_time",
		"signal_name",
		"signal_args",
		"signal_execfilepath",
		"signal_uid",
		"clusterid",
		"namespace",
		"serialized",
	}

	for idx, obj := range objs {
		// Todo: ROX-9499 Figure out how to more cleanly template around this issue.
		log.Debugf("This is here for now because there is an issue with pods_TerminatedInstances where the obj "+
			"in the loop is not used as it only consists of the parent ID and the index.  Putting this here as a stop gap "+
			"to simply use the object.  %s", obj)

		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			pgutils.NilOrUUID(obj.GetId()),
			pgutils.NilOrUUID(obj.GetDeploymentId()),
			obj.GetContainerName(),
			obj.GetPodId(),
			pgutils.NilOrUUID(obj.GetPodUid()),
			obj.GetSignal().GetContainerId(),
			pgutils.NilOrTime(obj.GetSignal().GetTime()),
			obj.GetSignal().GetName(),
			obj.GetSignal().GetArgs(),
			obj.GetSignal().GetExecFilePath(),
			obj.GetSignal().GetUid(),
			pgutils.NilOrUUID(obj.GetClusterId()),
			obj.GetNamespace(),
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

			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"process_indicators"}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
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

// Destroy drops the tables associated with the target object type.
func Destroy(ctx context.Context, db postgres.DB) {
	dropTableProcessIndicators(ctx, db)
}

func dropTableProcessIndicators(ctx context.Context, db postgres.DB) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS process_indicators CASCADE")

}

// endregion Used for testing
