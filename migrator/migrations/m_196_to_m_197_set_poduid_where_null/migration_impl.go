package m196tom197

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	podSchema "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/schema/pods"
	listeningEndpointsSchema "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/schema/listening_endpoints"
	podDatastore "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/store/pod"
	plopDatastore "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/store/processlisteningonport"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	batchSize = 2000
	log       = logging.LoggerForModule()
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())
	pgutils.CreateTableFromModel(ctx, database.GormDB, podSchema.CreateTablePodsStmt)
	pgutils.CreateTableFromModel(ctx, database.GormDB, listeningEndpointsSchema.CreateTableListeningEndpointsStmt)

	return updateGlobalScope(ctx, database)
}

func getPodKey(podName, deploymentId string) string {
	return fmt.Sprintf("%s_%s", podName, deploymentId)
}

func getPodUIDMap(ctx context.Context, podStore podDatastore.Store) (map[string]string, error) {
	podUIDMap := make(map[string]string)

	err := podStore.Walk(ctx,
		func(pod *storage.Pod) error {
			podKey := getPodKey(pod.GetName(), pod.GetDeploymentId())
			podUIDMap[podKey] = pod.GetId()
			return nil
		})

	return podUIDMap, err
}

func setPodUids(ctx context.Context, plopStore plopDatastore.Store, podUIDMap map[string]string) error {
	plops := make([]*storage.ProcessListeningOnPortStorage, batchSize)
	count := 0
	err := plopStore.Walk(ctx,
		func(plop *storage.ProcessListeningOnPortStorage) error {
			podKey := getPodKey(plop.GetProcess().GetPodId(), plop.GetDeploymentId())
			podUID, exists := podUIDMap[podKey]
			if exists {
				plop.PodUid = podUID
				plops[count] = plop
				count += 1
			}

			if count == batchSize {
				err := plopStore.UpsertMany(ctx, plops)
				count = 0
				if err != nil {
					return err
				}
			}

			return nil
		})

	if count > 0 {
		plops = plops[:count]
		err := plopStore.UpsertMany(ctx, plops)
		if err != nil {
			return err
		}
	}

	return err
}

func updateGlobalScope(ctx context.Context, database *types.Databases) error {
	podStore := podDatastore.New(database.PostgresDB)
	plopStore := plopDatastore.New(database.PostgresDB)
	podUIDMap, err := getPodUIDMap(ctx, podStore)
	if err != nil {
		return err
	}

	err = setPodUids(ctx, plopStore, podUIDMap)

	return err
}
