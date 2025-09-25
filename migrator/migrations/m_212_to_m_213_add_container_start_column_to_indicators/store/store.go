package postgres

import (
	"context"
	"slices"

	"github.com/jackc/pgx/v5"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pgSearch "github.com/stackrox/rox/migrator/migrations/m_212_to_m_213_add_container_start_column_to_indicators/generic"
	pkgSchema "github.com/stackrox/rox/migrator/migrations/m_212_to_m_213_add_container_start_column_to_indicators/schema"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac/resources"
)

const (
	storeName = "ProcessIndicator"
)

var (
	log            = logging.LoggerForModule()
	schema         = pkgSchema.ProcessIndicatorsSchema
	targetResource = resources.DeploymentExtension
)

type (
	storeType = storage.ProcessIndicator
	callback  = func(obj *storeType) error
)

// Store is the interface to interact with the storage for storage.ProcessIndicator
type Store interface {
	Upsert(ctx context.Context, obj *storeType) error
	UpsertMany(ctx context.Context, objs []*storeType) error
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, identifiers []string) error

	GetByQueryFn(ctx context.Context, query *v1.Query, fn callback) error
	GetByQuery(ctx context.Context, q *v1.Query) ([]*storage.ProcessIndicator, error)
	WalkByQuery(ctx context.Context, query *v1.Query, fn callback) error
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
	return pgSearch.NewGenericStore[storeType, *storeType](
		db,
		schema,
		pkGetter,
		insertIntoProcessIndicators,
		copyFromProcessIndicators,
		nil,
		nil,
		isUpsertAllowed,
		targetResource,
		nil,
		nil,
	)
}

// region Helper functions

func pkGetter(obj *storeType) string {
	return obj.GetId()
}

func isUpsertAllowed(_ context.Context, _ ...*storeType) error {
	return nil
}

func insertIntoProcessIndicators(batch *pgx.Batch, obj *storage.ProcessIndicator) error {

	serialized, marshalErr := obj.MarshalVT()
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
		protocompat.NilOrTime(obj.GetSignal().GetTime()),
		obj.GetSignal().GetName(),
		obj.GetSignal().GetArgs(),
		obj.GetSignal().GetExecFilePath(),
		obj.GetSignal().GetUid(),
		pgutils.NilOrUUID(obj.GetClusterId()),
		obj.GetNamespace(),
		protocompat.NilOrTime(obj.GetContainerStartTime()),
		serialized,
	}

	finalStr := "INSERT INTO process_indicators (Id, DeploymentId, ContainerName, PodId, PodUid, Signal_ContainerId, Signal_Time, Signal_Name, Signal_Args, Signal_ExecFilePath, Signal_Uid, ClusterId, Namespace, ContainerStartTime, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, DeploymentId = EXCLUDED.DeploymentId, ContainerName = EXCLUDED.ContainerName, PodId = EXCLUDED.PodId, PodUid = EXCLUDED.PodUid, Signal_ContainerId = EXCLUDED.Signal_ContainerId, Signal_Time = EXCLUDED.Signal_Time, Signal_Name = EXCLUDED.Signal_Name, Signal_Args = EXCLUDED.Signal_Args, Signal_ExecFilePath = EXCLUDED.Signal_ExecFilePath, Signal_Uid = EXCLUDED.Signal_Uid, ClusterId = EXCLUDED.ClusterId, Namespace = EXCLUDED.Namespace, ContainerStartTime = EXCLUDED.ContainerStartTime, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

func copyFromProcessIndicators(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, objs ...*storage.ProcessIndicator) error {
	if len(objs) == 0 {
		return nil
	}
	batchSize := min(len(objs), pgSearch.MaxBatchSize)
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
		"containerstarttime",
		"serialized",
	}

	for objBatch := range slices.Chunk(objs, batchSize) {
		for _, obj := range objBatch {
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
				pgutils.NilOrUUID(obj.GetDeploymentId()),
				obj.GetContainerName(),
				obj.GetPodId(),
				pgutils.NilOrUUID(obj.GetPodUid()),
				obj.GetSignal().GetContainerId(),
				protocompat.NilOrTime(obj.GetSignal().GetTime()),
				obj.GetSignal().GetName(),
				obj.GetSignal().GetArgs(),
				obj.GetSignal().GetExecFilePath(),
				obj.GetSignal().GetUid(),
				pgutils.NilOrUUID(obj.GetClusterId()),
				obj.GetNamespace(),
				protocompat.NilOrTime(obj.GetContainerStartTime()),
				serialized,
			})

			// Add the ID to be deleted.
			deletes = append(deletes, obj.GetId())
		}

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

	return nil
}
