package m196tom197

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	podSchema "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/schema/pods"
	listeningEndpointsSchema "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/schema/listening_endpoints"
	processIndicatorSchema "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/schema/process_indicators"
	podDatastore "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/store/pod"
	plopDatastore "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/store/processlisteningonport"
	processIndicatorDatastore "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/store/processindicator"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	log       = logging.LoggerForModule()
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())
	pgutils.CreateTableFromModel(ctx, database.GormDB, podSchema.CreateTablePodsStmt)
	pgutils.CreateTableFromModel(ctx, database.GormDB, listeningEndpointsSchema.CreateTableListeningEndpointsStmt)
	pgutils.CreateTableFromModel(ctx, database.GormDB, processIndicatorSchema.CreateTableProcessIndicatorsStmt)

	return updateGlobalScope(ctx, database)
}

func getProcessIndicatorPodUIDMap(ctx context.Context, processIndicatorStore processIndicatorDatastore.Store, processIndicatorIds map[string]bool) (map[string]string, error) {
	podUIDMap := make(map[string]string)

	err := processIndicatorStore.Walk(ctx,
		func(processIndicator *storage.ProcessIndicator) error {
			_, exists := processIndicatorIds[processIndicator.Id]
			if exists {
				podUIDMap[processIndicator.Id] = processIndicator.GetPodUid()
				delete(processIndicatorIds, processIndicator.Id)
			}
			return nil
		})

	processIndicatorIds = make(map[string]bool)

	return podUIDMap, err
}

// Find out the process indicator ids of plops that could possibly be updated with a poduid from the process_indicators table.
// This way we don't have to store all id, poduid pairs from the process_indicators table.
func getProcessIndicatorIdsOfInterest(ctx context.Context, plopStore plopDatastore.Store) (map[string]bool, error) {
	processIndicatorIds := make(map[string]bool)

	err := plopStore.Walk(ctx,
		func(plop *storage.ProcessListeningOnPortStorage) error {
			if plop.GetPodUid() == "" && plop.GetProcess() == nil {
				processIndicatorIds[plop.GetProcessIndicatorId()] = true
			}
			return nil
		})

	return processIndicatorIds, err
}

func setPodUidsUsingProcessIndicators(ctx context.Context, plopStore plopDatastore.Store, podUIDMap map[string]string, batchSize int) error {
	plops := make([]*storage.ProcessListeningOnPortStorage, batchSize)
	count := 0
	err := plopStore.Walk(ctx,
		func(plop *storage.ProcessListeningOnPortStorage) error {
			if plop.GetPodUid() == "" {
				podUID, exists := podUIDMap[plop.GetProcessIndicatorId()]
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

func setPodUIDsUsingProcessIndicators(ctx context.Context, processIndicatorStore processIndicatorDatastore.Store, plopStore plopDatastore.Store, batchSize int) error {
	processIndicatorIds, err := getProcessIndicatorIdsOfInterest(ctx, plopStore)
	if err != nil {
		return err
	}

	podUIDMap, err := getProcessIndicatorPodUIDMap(ctx, processIndicatorStore, processIndicatorIds)
	if err != nil {
		return err
	}

	err = setPodUidsUsingProcessIndicators(ctx, plopStore, podUIDMap, batchSize)

	return err
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

func setPodUidsUsingPods(ctx context.Context, plopStore plopDatastore.Store, podUIDMap map[string]string, batchSize int) error {
	plops := make([]*storage.ProcessListeningOnPortStorage, batchSize)
	count := 0
	err := plopStore.Walk(ctx,
		func(plop *storage.ProcessListeningOnPortStorage) error {
			if plop.GetPodUid() == "" && plop.GetProcess() != nil {
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


func setPodUIDsUsingPods(ctx context.Context, podStore podDatastore.Store, plopStore plopDatastore.Store, batchSize int) error {
	podUIDMap, err := getPodUIDMap(ctx, podStore)
	if err != nil {
		return err
	}

	err = setPodUidsUsingPods(ctx, plopStore, podUIDMap, batchSize)

	return err
}

func updateGlobalScope(ctx context.Context, database *types.Databases) error {
	batchSize := 2000
	podStore := podDatastore.New(database.PostgresDB)
	plopStore := plopDatastore.New(database.PostgresDB)
	processIndicatorStore := processIndicatorDatastore.New(database.PostgresDB)

	err := setPodUIDsUsingProcessIndicators(ctx, processIndicatorStore, plopStore, batchSize)
	if err != nil {
		return err
	}

	err = setPodUIDsUsingPods(ctx, podStore, plopStore, batchSize)
	if err != nil {
		return err
	}

	return err
}
