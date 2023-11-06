package m197tom198

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	listeningEndpointsSchema "github.com/stackrox/rox/migrator/migrations/m_197_to_m_198_set_poduid_where_null/schema/listening_endpoints"
	podSchema "github.com/stackrox/rox/migrator/migrations/m_197_to_m_198_set_poduid_where_null/schema/pods"
	processIndicatorSchema "github.com/stackrox/rox/migrator/migrations/m_197_to_m_198_set_poduid_where_null/schema/process_indicators"
	podDatastore "github.com/stackrox/rox/migrator/migrations/m_197_to_m_198_set_poduid_where_null/store/pod"
	processIndicatorDatastore "github.com/stackrox/rox/migrator/migrations/m_197_to_m_198_set_poduid_where_null/store/processindicator"
	plopDatastore "github.com/stackrox/rox/migrator/migrations/m_197_to_m_198_set_poduid_where_null/store/processlisteningonport"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	log = logging.LoggerForModule()
)

// The purpose of this migration is to set the value of PodUid in the listening_endpoints table where
// it is null and it is possible to give it a value.
func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())
	pgutils.CreateTableFromModel(ctx, database.GormDB, podSchema.CreateTablePodsStmt)
	pgutils.CreateTableFromModel(ctx, database.GormDB, listeningEndpointsSchema.CreateTableListeningEndpointsStmt)
	pgutils.CreateTableFromModel(ctx, database.GormDB, processIndicatorSchema.CreateTableProcessIndicatorsStmt)

	return updateGlobalScope(ctx, database)
}

// Walk through the process_indicators table and get a map where the key is the id and the value is the
// PodUid. It doesn't do this for every process indicator, but only for the ones where it could possibly
// be used to set PodUid in the listening endpoints table.
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

// Find out the process indicator ids of plops that could possibly be updated with a poduid from the
// process_indicators table. This way we don't have to store all id, poduid pairs from the
// process_indicators table. Go doesn't have sets so use a map where the value is always true instead.
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

// Walks through the listening_endpoints table and sets the value of PodUid where it is null using a
// map obtained from the process_indicators table, where the key is a processindicatorid and the value
// is a PodUid.
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
					count++
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

// Sets PodUids by using information from the process_indicators table. It first walks through the
// listening_endpoints table and gets processindicatorids where PodUid is null and the processindicatorid
// could possibly be matched to an id in the process_indicators table. It then uses that to get
// a map where the key is the processindicatorid and the value is the PodUid. It then walks through the
// the listening_endpoints table again and sets the PodUids using the map. This is effectively a join.
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

// Returns a string which is a combination of the pod name and deploymentid so that listening_endpoints
// can be matched to pods.
func getPodKey(podName, deploymentID string) string {
	// The _ character cannot appear in a podName, so it is a good separator
	return fmt.Sprintf("%s_%s", podName, deploymentID)
}

// Get a map where the key is a combination of the pod name and deploymenid and the value is the
// PodUid. This is later used to set the PodUids in the listening endpoints table.
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

// Walks through the listening_endpoints table and sets the value of PodUid using a map obtained from the
// pods table where the key is a combination of the pod name and deploymentid.
func setPodUidsUsingPods(ctx context.Context, plopStore plopDatastore.Store, podUIDMap map[string]string, batchSize int) error {
	plops := make([]*storage.ProcessListeningOnPortStorage, batchSize)
	count := 0
	err := plopStore.Walk(ctx,
		func(plop *storage.ProcessListeningOnPortStorage) error {
			// Don't set the PodUid if it is already set. The process information has the
			// podid, so it must be set to proceed.
			if plop.GetPodUid() == "" && plop.GetProcess() != nil {
				podKey := getPodKey(plop.GetProcess().GetPodId(), plop.GetDeploymentId())
				podUID, exists := podUIDMap[podKey]
				if exists {
					plop.PodUid = podUID
					plops[count] = plop
					count++
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

// Sets PodUids by using information from the pods table. It first walks through the pods table and
// makes a table where the key is a combination of the pod name and deploymentid and the value is the
// PodUid. It then walks through the listening_endpoints table and uses the map to set the value of
// PodUid where it is null and the process information exists. This is effectively a join. If the
// process information does not exist the podid (pod name) doesn't exist, and the PodUid cannot be set
// using this strategy.
func setPodUIDsUsingPods(ctx context.Context, podStore podDatastore.Store, plopStore plopDatastore.Store, batchSize int) error {
	podUIDMap, err := getPodUIDMap(ctx, podStore)
	if err != nil {
		return err
	}

	err = setPodUidsUsingPods(ctx, plopStore, podUIDMap, batchSize)

	return err
}

// There are two strategies for setting PodUid in the listening_endpoints table. The first is to use
// the process_indicators table. The listening_endpoints table can be joined to the process_indicators
// table using the processindicatorid column from listening_endpoints and the id column from the
// process_indicators table. Then the PodUid can be set using the value from the process_indicators
// table. In cases where the processindicatorid does not have any matching process indicator the pods
// table can be used to set the PodUid. In cases where there is a matching process indicator the
// listening_endpoints table stores no process information and in cases where there is no matching
// process indicator the listening_endpoints tables store process information, including the podid.
// The podid is not the PodUid. It is the name of the pod. The podid along with the deploymentid
// can be used to do a join on the pods table and obtain the PodUid. Where one strategy doesn't work
// the other should work, unless the pod for the listening endpoint has been deleted.
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
