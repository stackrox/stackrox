package groups

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/generated/storage"
	schemaPkg "github.com/stackrox/rox/migrator/migrations/m_197_to_m_198_add_oidc_claim_mappings/schema"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac/resources"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

const (
	batchSize = 10000
)

var (
	log            = logging.LoggerForModule()
	schema         = schemaPkg.GroupsSchema
	targetResource = resources.Access
)

type storeType = storage.Group

// Store is the interface to interact with the storage for storage.Group
type Store interface {
	Walk(ctx context.Context, fn func(obj *storeType) error) error
	UpsertMany(ctx context.Context, objs []*storeType) error
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
	return pgSearch.NewGenericStore[storeType, *storeType](
		db,
		schema,
		pkGetter,
		insertIntoGroups,
		copyFromGroups,
		metricsSetAcquireDBConnDuration,
		metricsSetPostgresOperationDurationTime,
		pgSearch.GloballyScopedUpsertChecker[storeType, *storeType](targetResource),
		targetResource,
	)
}

func metricsSetPostgresOperationDurationTime(_ time.Time, _ ops.Op) {
}

func metricsSetAcquireDBConnDuration(_ time.Time, _ ops.Op) {
}

func pkGetter(obj *storeType) string {
	return obj.GetProps().GetId()
}

func insertIntoGroups(batch *pgx.Batch, obj *storage.Group) error {
	serialized, marshalErr := obj.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		obj.GetProps().GetId(),
		obj.GetProps().GetAuthProviderId(),
		obj.GetProps().GetKey(),
		obj.GetProps().GetValue(),
		obj.GetRoleName(),
		serialized,
	}

	finalStr := "INSERT INTO groups (Props_Id, Props_AuthProviderId, Props_Key, Props_Value, RoleName, serialized) VALUES($1, $2, $3, $4, $5, $6) ON CONFLICT(Props_Id) DO UPDATE SET Props_Id = EXCLUDED.Props_Id, Props_AuthProviderId = EXCLUDED.Props_AuthProviderId, Props_Key = EXCLUDED.Props_Key, Props_Value = EXCLUDED.Props_Value, RoleName = EXCLUDED.RoleName, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

func copyFromGroups(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, objs ...*storage.Group) error {
	inputRows := make([][]interface{}, 0, batchSize)

	// This is a copy so first we must delete the rows and re-add them
	// Which is essentially the desired behaviour of an upsert.
	deletes := make([]string, 0, batchSize)

	copyCols := []string{
		"props_id",
		"props_authproviderid",
		"props_key",
		"props_value",
		"rolename",
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
			obj.GetProps().GetId(),
			obj.GetProps().GetAuthProviderId(),
			obj.GetProps().GetKey(),
			obj.GetProps().GetValue(),
			obj.GetRoleName(),
			serialized,
		})

		// Add the ID to be deleted.
		deletes = append(deletes, obj.GetProps().GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			if err := s.DeleteMany(ctx, deletes); err != nil {
				return err
			}
			// clear the inserts and vals for the next batch
			deletes = deletes[:0]

			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"groups"}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return nil
}
