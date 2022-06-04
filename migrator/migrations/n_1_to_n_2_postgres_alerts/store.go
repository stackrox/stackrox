package n1ton2

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres"
)

type storeImpl struct {
	db *pgxpool.Pool
}

var (
	batchSize = 10000
	schema    = pkgSchema.AlertsSchema
)

// newStore returns a new Store instance using the provided sql instance.
func newStore(db *pgxpool.Pool) *storeImpl {
	return &storeImpl{
		db: db,
	}
}

func (s *storeImpl) copyFromAlerts(ctx context.Context, tx pgx.Tx, objs ...*storage.Alert) error {

	inputRows := [][]interface{}{}

	var err error

	// This is a copy so first we must delete the rows and re-add them
	// Which is essentially the desired behaviour of an upsert.
	var deletes []string

	copyCols := []string{

		"id",

		"policy_id",

		"policy_name",

		"policy_description",

		"policy_disabled",

		"policy_categories",

		"policy_lifecyclestages",

		"policy_severity",

		"policy_enforcementactions",

		"policy_lastupdated",

		"policy_sortname",

		"policy_sortlifecyclestage",

		"policy_sortenforcement",

		"lifecyclestage",

		"clusterid",

		"clustername",

		"namespace",

		"namespaceid",

		"deployment_id",

		"deployment_name",

		"deployment_inactive",

		"image_id",

		"image_name_registry",

		"image_name_remote",

		"image_name_tag",

		"image_name_fullname",

		"resource_resourcetype",

		"resource_name",

		"enforcement_action",

		"time",

		"state",

		"tags",

		"serialized",
	}

	for idx, obj := range objs {
		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{

			obj.GetId(),

			obj.GetPolicy().GetId(),

			obj.GetPolicy().GetName(),

			obj.GetPolicy().GetDescription(),

			obj.GetPolicy().GetDisabled(),

			obj.GetPolicy().GetCategories(),

			obj.GetPolicy().GetLifecycleStages(),

			obj.GetPolicy().GetSeverity(),

			obj.GetPolicy().GetEnforcementActions(),

			pgutils.NilOrTime(obj.GetPolicy().GetLastUpdated()),

			obj.GetPolicy().GetSORTName(),

			obj.GetPolicy().GetSORTLifecycleStage(),

			obj.GetPolicy().GetSORTEnforcement(),

			obj.GetLifecycleStage(),

			obj.GetClusterId(),

			obj.GetClusterName(),

			obj.GetNamespace(),

			obj.GetNamespaceId(),

			obj.GetDeployment().GetId(),

			obj.GetDeployment().GetName(),

			obj.GetDeployment().GetInactive(),

			obj.GetImage().GetId(),

			obj.GetImage().GetName().GetRegistry(),

			obj.GetImage().GetName().GetRemote(),

			obj.GetImage().GetName().GetTag(),

			obj.GetImage().GetName().GetFullName(),

			obj.GetResource().GetResourceType(),

			obj.GetResource().GetName(),

			obj.GetEnforcement().GetAction(),

			pgutils.NilOrTime(obj.GetTime()),

			obj.GetState(),

			obj.GetTags(),

			serialized,
		})

		// Add the id to be deleted.
		deletes = append(deletes, obj.GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			if err := s.DeleteMany(ctx, deletes); err != nil {
				return err
			}
			// clear the inserts and vals for the next batch
			deletes = nil

			_, err = tx.CopyFrom(ctx, pgx.Identifier{"alerts"}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return err
}

func (s *storeImpl) copyFrom(ctx context.Context, objs ...*storage.Alert) error {
	conn, release, err := s.acquireConn(ctx, ops.Get, "Alert")
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	if err := s.copyFromAlerts(ctx, tx, objs...); err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return err
		}
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *storeImpl) acquireConn(ctx context.Context, _ ops.Op, _ string) (*pgxpool.Conn, func(), error) {
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

func (s *storeImpl) DeleteMany(_ context.Context, ids []string) error {
	q := search.NewQueryBuilder().AddDocIDs(ids...).ProtoQuery()
	return postgres.RunDeleteRequestForSchema(schema, q, s.db)
}
