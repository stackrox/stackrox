package updated

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	newSchema "github.com/stackrox/rox/migrator/migrations/m_204_to_m_205_clusters_platform_type_and_k8_version/schema/new"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

var (
	schema         = newSchema.ClustersSchema
	targetResource = resources.Cluster
)

type storeType = storage.Cluster

// Store is the interface to interact with the storage for storage.Cluster
type Store interface {
	GetByQuery(ctx context.Context, query *v1.Query) ([]*storeType, error)
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
	// Use of pgSearch.NewGenericStoreWithCache can be dangerous with high cardinality stores,
	// and be the source of memory pressure. Think twice about the need for in-memory caching
	// of the whole store.
	return pgSearch.NewGenericStore[storeType, *storeType](
		db,
		schema,
		pkGetter,
		insertIntoClusters,
		nil,
		nil,
		nil,

		isUpsertAllowed,
		targetResource,
	)
}

// region Helper functions

func pkGetter(obj *storeType) string {
	return obj.GetId()
}

func isUpsertAllowed(ctx context.Context, objs ...*storeType) error {
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_WRITE_ACCESS).Resource(targetResource)
	if scopeChecker.IsAllowed() {
		return nil
	}
	var deniedIDs []string
	for _, obj := range objs {
		subScopeChecker := scopeChecker.ClusterID(obj.GetId())
		if !subScopeChecker.IsAllowed() {
			deniedIDs = append(deniedIDs, obj.GetId())
		}
	}
	if len(deniedIDs) != 0 {
		return errors.Wrapf(sac.ErrResourceAccessDenied, "modifying clusters with IDs [%s] was denied", strings.Join(deniedIDs, ", "))
	}
	return nil
}

func insertIntoClusters(batch *pgx.Batch, obj *storage.Cluster) error {

	serialized, marshalErr := obj.MarshalVT()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		pgutils.NilOrUUID(obj.GetId()),
		obj.GetName(),
		obj.GetType(),
		pgutils.EmptyOrMap(obj.GetLabels()),
		obj.GetStatus().GetProviderMetadata().GetCluster().GetType(),
		obj.GetStatus().GetOrchestratorMetadata().GetVersion(),
		serialized,
	}

	finalStr := "INSERT INTO clusters (Id, Name, Type, Labels, Status_ProviderMetadata_Cluster_Type, Status_OrchestratorMetadata_Version, serialized) VALUES($1, $2, $3, $4, $5, $6, $7) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, Name = EXCLUDED.Name, Type = EXCLUDED.Type, Labels = EXCLUDED.Labels, Status_ProviderMetadata_Cluster_Type = EXCLUDED.Status_ProviderMetadata_Cluster_Type, Status_OrchestratorMetadata_Version = EXCLUDED.Status_OrchestratorMetadata_Version, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

// endregion Helper functions
