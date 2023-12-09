package m198tom199

import (
	"github.com/stackrox/rox/migrator/types"
)
import (
	"context"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	listeningEndpointsSchema "github.com/stackrox/rox/migrator/migrations/m_198_to_m_199_remove_listening_endpoints_where_pod_uid_is_null/schema/listening_endpoints"
	plopDatastore "github.com/stackrox/rox/migrator/migrations/m_198_to_m_199_remove_listening_endpoints_where_pod_uid_is_null/store/processlisteningonport"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	log = logging.LoggerForModule()
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())
	pgutils.CreateTableFromModel(ctx, database.GormDB, listeningEndpointsSchema.CreateTableListeningEndpointsStmt)

	batchSize := 10000

	return updatePodUids(ctx, database, batchSize
}

func removeListeningEndpointsWherePodUidIsNull(ctx context.Context, database *types.Databases, batchSize int) error {
	idsToDelete := make([]string, batchSize)
	count := 0

	err := plopStore.Walk(ctx,
		func(plop *storage.ProcessListeningOnPortStorage) error {
			if plop.GetPodUid() == "" {
				idsToDelete[count] = plop.Id
				count++
			}

			if count == batchSize {
				err := plopStore.DeleteMany(ctx, idsToDelete)
				count = 0
				if err != nil {
					return err
				}
			}

			return nil
		})

		if count > 0 {
			idsToDelete = idsToDelete[:count]
			err := plopStore.DeleteMany(ctx, idsToDelete)
			if err != nil {
				return err
			}
		}

	return nil
}
