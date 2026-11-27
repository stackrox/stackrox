package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/generated/storage"
	pkgSchema "github.com/stackrox/rox/migrator/migrations/m_230_to_m_231_drop_machine_to_machine_configuration_unique_index_per_issuer/schema/new"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac/resources"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

var (
	log            = logging.LoggerForModule()
	schema         = pkgSchema.AuthMachineToMachineConfigsSchema
	targetResource = resources.Access
)

type (
	storeType = storage.AuthMachineToMachineConfig
	callback  = func(obj *storeType) error
)

// Store is the interface to interact with the storage for storage.AuthMachineToMachineConfig
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
		insertIntoAuthMachineToMachineConfigs,
		copyFromAuthMachineToMachineConfigs,
		metricsSetAcquireDBConnDuration,
		metricsSetPostgresOperationDurationTime,
		targetResource,
		nil,
		nil,
	)
}

// region Helper functions

func pkGetter(obj *storeType) string {
	return obj.GetId()
}

func metricsSetPostgresOperationDurationTime(_ time.Time, _ ops.Op) {}

func metricsSetAcquireDBConnDuration(_ time.Time, _ ops.Op) {}

func insertIntoAuthMachineToMachineConfigs(batch *pgx.Batch, obj *storage.AuthMachineToMachineConfig) error {

	serialized, marshalErr := obj.MarshalVT()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		pgutils.NilOrUUID(obj.GetId()),
		serialized,
	}

	finalStr := "INSERT INTO auth_machine_to_machine_configs (Id, serialized) VALUES($1, $2) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	var query string

	for childIndex, child := range obj.GetMappings() {
		if err := insertIntoAuthMachineToMachineConfigsMappings(batch, child, obj.GetId(), childIndex); err != nil {
			return err
		}
	}

	query = "delete from auth_machine_to_machine_configs_mappings where auth_machine_to_machine_configs_Id = $1 AND idx >= $2"
	batch.Queue(query, pgutils.NilOrUUID(obj.GetId()), len(obj.GetMappings()))
	return nil
}

func insertIntoAuthMachineToMachineConfigsMappings(batch *pgx.Batch, obj *storage.AuthMachineToMachineConfig_Mapping, authMachineToMachineConfigID string, idx int) error {

	values := []interface{}{
		// parent primary keys start
		pgutils.NilOrUUID(authMachineToMachineConfigID),
		idx,
		obj.GetRole(),
	}

	finalStr := "INSERT INTO auth_machine_to_machine_configs_mappings (auth_machine_to_machine_configs_Id, idx, Role) VALUES($1, $2, $3) ON CONFLICT(auth_machine_to_machine_configs_Id, idx) DO UPDATE SET auth_machine_to_machine_configs_Id = EXCLUDED.auth_machine_to_machine_configs_Id, idx = EXCLUDED.idx, Role = EXCLUDED.Role"
	batch.Queue(finalStr, values...)

	return nil
}

var copyColsAuthMachineToMachineConfigs = []string{
	"id",
	"serialized",
}

func copyFromAuthMachineToMachineConfigs(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, objs ...*storage.AuthMachineToMachineConfig) error {
	if len(objs) == 0 {
		return nil
	}

	{
		// CopyFrom does not upsert, so delete existing rows first to achieve upsert behavior.
		// Parent deletion cascades to children, so only the top-level parent needs deletion.
		deletes := make([]string, 0, len(objs))
		for _, obj := range objs {
			deletes = append(deletes, obj.GetId())
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
			pgutils.NilOrUUID(obj.GetId()),
			serialized,
		}, nil
	})

	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"auth_machine_to_machine_configs"}, copyColsAuthMachineToMachineConfigs, inputRows); err != nil {
		return err
	}

	for _, obj := range objs {
		if err := copyFromAuthMachineToMachineConfigsMappings(ctx, s, tx, obj.GetId(), obj.GetMappings()...); err != nil {
			return err
		}
	}

	return nil
}

var copyColsAuthMachineToMachineConfigsMappings = []string{
	"auth_machine_to_machine_configs_id",
	"idx",
	"role",
}

func copyFromAuthMachineToMachineConfigsMappings(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, authMachineToMachineConfigID string, objs ...*storage.AuthMachineToMachineConfig_Mapping) error {
	if len(objs) == 0 {
		return nil
	}

	idx := 0
	inputRows := pgx.CopyFromFunc(func() ([]any, error) {
		if idx >= len(objs) {
			return nil, nil
		}
		obj := objs[idx]
		idx++

		return []interface{}{
			pgutils.NilOrUUID(authMachineToMachineConfigID),
			idx,
			obj.GetRole(),
		}, nil
	})

	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"auth_machine_to_machine_configs_mappings"}, copyColsAuthMachineToMachineConfigsMappings, inputRows); err != nil {
		return err
	}

	return nil
}

// endregion Helper functions
