package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac/resources"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

const (
	baseTable = "roles"
	storeName = "Role"
)

var (
	log            = logging.LoggerForModule()
	schema         = pkgSchema.RolesSchema
	targetResource = resources.Access
)

type (
	storeType = storage.Role
	callback  = func(obj *storeType) error
)

// Store is the interface to interact with the storage for storage.Role
type Store interface {
	Upsert(ctx context.Context, obj *storeType) error

	Walk(ctx context.Context, fn callback) error
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
	return pgSearch.NewGloballyScopedGenericStore[storeType, *storeType](
		db,
		schema,
		pkGetter,
		insertIntoRoles,
		copyFromRoles,
		metricsSetAcquireDBConnDuration,
		metricsSetPostgresOperationDurationTime,
		targetResource,
		nil,
		nil,
	)
}

// region Helper functions

func pkGetter(obj *storeType) string {
	return obj.GetName()
}

func metricsSetPostgresOperationDurationTime(_ time.Time, _ ops.Op) {}

func metricsSetAcquireDBConnDuration(_ time.Time, _ ops.Op) {}

func insertIntoRoles(batch *pgx.Batch, obj *storage.Role) error {

	serialized, marshalErr := obj.MarshalVT()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		obj.GetName(),
		serialized,
	}

	finalStr := "INSERT INTO roles (Name, serialized) VALUES($1, $2) ON CONFLICT(Name) DO UPDATE SET Name = EXCLUDED.Name, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

var copyColsRoles = []string{
	"name",
	"serialized",
}

func copyFromRoles(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, objs ...*storage.Role) error {
	if len(objs) == 0 {
		return nil
	}

	{
		// CopyFrom does not upsert, so delete existing rows first to achieve upsert behavior.
		// Parent deletion cascades to children, so only the top-level parent needs deletion.
		deletes := make([]string, 0, len(objs))
		for _, obj := range objs {
			deletes = append(deletes, obj.GetName())
		}
		if err := s.DeleteMany(ctx, deletes); err != nil {
			return err
		}
	}

	idx := 0
	inputRows := pgx.CopyFromFunc(func() ([]any, error) {
		if idx >= len(objs) {
			return nil, nil
		}
		obj := objs[idx]
		idx++

		serialized, marshalErr := obj.MarshalVT()
		if marshalErr != nil {
			return nil, marshalErr
		}

		return []interface{}{
			obj.GetName(),
			serialized,
		}, nil
	})

	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"roles"}, copyColsRoles, inputRows); err != nil {
		return err
	}

	return nil
}

// endregion Helper functions
