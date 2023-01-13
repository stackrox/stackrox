package datastore

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/rox/central/metrics"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
)

type datastoreImpl struct {
	storage            postgres.Store
	indicatorDataStore processIndicatorStore.DataStore
}

var (
	plopSAC = sac.ForResource(resources.DeploymentExtension)
	log     = logging.LoggerForModule()
)

func newDatastoreImpl(
	storage postgres.Store,
	indicatorDataStore processIndicatorStore.DataStore,
) *datastoreImpl {
	return &datastoreImpl{
		storage:            storage,
		indicatorDataStore: indicatorDataStore,
	}
}

func getPlopsFromNormalizedResult(
	normalized []*OpenClosedPLOPs) []*storage.ProcessListeningOnPortFromSensor {

	plops := []*storage.ProcessListeningOnPortFromSensor{}

	for _, value := range normalized {
		plops = append(plops, value.plop)
	}

	return plops
}

func (ds *datastoreImpl) AddProcessListeningOnPort(
	ctx context.Context,
	portProcesses ...*storage.ProcessListeningOnPortFromSensor,
) error {
	defer metrics.SetDatastoreFunctionDuration(
		time.Now(),
		"ProcessListeningOnPort",
		"AddProcessListeningOnPort",
	)

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		// PLOP is a Postgres-only feature, do nothing.
		log.Warnf("Tried to add PLOP not on Postgres, ignore: %+v", portProcesses)
		return nil
	}

	if ok, err := plopSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	normalizedPlops := normalizePLOPs(portProcesses)
	portProcesses = getPlopsFromNormalizedResult(normalizedPlops)

	// XXX: The next two calls, fetchIndicators and fetchExistingPLOPs, have to
	// be done in a single join query fetching both ProcessIndicator and needed
	// bits from PLOP.
	indicatorsMap, indicatorIds, err := ds.fetchIndicators(ctx, portProcesses...)
	if err != nil {
		return err
	}

	existingPLOPMap, err := ds.fetchExistingPLOPs(ctx, indicatorIds, portProcesses...)
	if err != nil {
		return err
	}

	plopObjects := make([]*storage.ProcessListeningOnPortStorage, len(portProcesses))
	for i, val := range normalizedPlops {
		indicatorID := ""
		var processInfo *storage.ProcessIndicatorUniqueKey

		key := getProcesUniqueKey(val.plop)

		if indicator, ok := indicatorsMap[key]; ok {
			indicatorID = indicator.GetId()
			log.Debugf("Got indicator %s: %+v", indicatorID, indicator)
		} else {
			// XXX: Create a metric for this
			log.Warnf("Found no matching indicators for %s", key)
			processInfo = val.plop.Process
		}

		plopKey := fmt.Sprintf("%d %d %s",
			val.plop.GetProtocol(),
			val.plop.GetPort(),
			indicatorID,
		)

		existingPLOP, prevExists := existingPLOPMap[plopKey]

		// There are three options:
		// * We found an existing PLOP object with different close timestamp.
		//   It has to be updated.
		// * We found an existing PLOP object with the same close timestamp.
		//   Nothing has to be changed (XXX: Ideally it has to be excluded from
		//   the upsert later on).
		// * No existing PLOP object, create a new one with whatever close
		//   timestamp we have received and fetched indicator ID.
		if prevExists {
			log.Debugf("Got existing PLOP: %+v", existingPLOP)

			if val.nopen > val.nclosed {
				existingPLOP.CloseTimestamp = nil
				existingPLOP.Closed = false
			}

			if val.nopen < val.nclosed {
				existingPLOP.CloseTimestamp = val.plop.CloseTimestamp
				existingPLOP.Closed = true
			}

			if val.nopen == val.nclosed {
				if existingPLOP.Closed == true {
					existingPLOP.CloseTimestamp = val.plop.CloseTimestamp
				}
			}

			//if existingPLOP.CloseTimestamp != val.CloseTimestamp {
			//	existingPLOP.CloseTimestamp = val.CloseTimestamp
			//	existingPLOP.Closed = existingPLOP.CloseTimestamp != nil
			//}

			plopObjects[i] = existingPLOP
		} else {
			if val.plop.CloseTimestamp != nil {
				// We try to close a not existing Endpoint, something is wrong
				log.Warnf("Found no matching PLOP to close for %s", key)
			}

			newPLOP := &storage.ProcessListeningOnPortStorage{
				// XXX, ResignatingFacepalm: Use regular GENERATE ALWAYS AS
				// IDENTITY, which would require changes in store generator
				Id:                 uuid.NewV4().String(),
				Port:               val.plop.Port,
				Protocol:           val.plop.Protocol,
				ProcessIndicatorId: indicatorID,
				Process:            processInfo,
				Closed:             val.plop.CloseTimestamp != nil,
				CloseTimestamp:     val.plop.CloseTimestamp,
			}

			plopObjects[i] = newPLOP
		}
	}

	// Now save actual PLOP objects
	err = ds.storage.UpsertMany(ctx, plopObjects)
	if err != nil {
		return err
	}

	return nil
}

func (ds *datastoreImpl) GetProcessListeningOnPort(
	ctx context.Context,
	deployment string,
) (
	processesListeningOnPorts []*storage.ProcessListeningOnPort, err error,
) {
	if ok, err := plopSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	processesListeningOnPorts, err = ds.storage.GetProcessListeningOnPort(ctx, deployment)

	if err != nil {
		return nil, err
	}

	return processesListeningOnPorts, nil
}

func (ds *datastoreImpl) WalkAll(ctx context.Context, fn WalkFn) error {
	if ok, err := plopSAC.ReadAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.Walk(ctx, fn)
}

func (ds *datastoreImpl) RemovePLOP(ctx context.Context, ids []string) error {
	if ok, err := plopSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.removePLOP(ctx, ids)
}

func (ds *datastoreImpl) removePLOP(ctx context.Context, ids []string) error {
	log.Infof("Deleting PLOP records")

	if len(ids) == 0 {
		return nil
	}
	if err := ds.storage.DeleteMany(ctx, ids); err != nil {
		return err
	}

	return nil
}

// fetchExistingPLOPs: Query already existing PLOP objects belonging to the
// specified process indicators.
//
// XXX: This function queries all PLOP, no matter if they are matching port +
// protocol we've got or not. This means potentially dangerous corner cases
// when one process listenst to a huge number of ports. To address it we could
// introduce filtering by port and protocol to the query, and even without
// extra indices PostgreSQL will be able to do it relatively efficiently using
// bitmap scan.
func (ds *datastoreImpl) fetchExistingPLOPs(
	ctx context.Context,
	indicatorIds []string,
	portProcesses ...*storage.ProcessListeningOnPortFromSensor,
) (map[string]*storage.ProcessListeningOnPortStorage, error) {

	var existingPLOPMap = map[string]*storage.ProcessListeningOnPortStorage{}

	if len(indicatorIds) == 0 {
		return nil, nil
	}

	// If no corresponding processes found, we can't verify if the PLOP
	// object is opening/closing an existing one. Collect existingPLOPMap
	// only if there are some matching indicators.
	existingPLOP, err := ds.storage.GetByQuery(ctx, search.NewQueryBuilder().
		AddStrings(search.ProcessID, indicatorIds...).ProtoQuery())
	if err != nil {
		return nil, err
	}

	for _, val := range existingPLOP {
		key := fmt.Sprintf("%d %d %s",
			val.GetProtocol(),
			val.GetPort(),
			val.GetProcessIndicatorId(),
		)

		// A bit of paranoia is always good
		if old, ok := existingPLOPMap[key]; ok {
			log.Warnf("A PLOP %s is already present, overwrite with %s",
				old.GetId(), val.GetId())
		}

		existingPLOPMap[key] = val
	}

	return existingPLOPMap, nil
}

// fetchIndicators: Query all needed process indicators references from PLOPS
// in one go. Besides the indicator map it also returns the list of ids for
// convenience to pass it further.
func (ds *datastoreImpl) fetchIndicators(
	ctx context.Context,
	portProcesses ...*storage.ProcessListeningOnPortFromSensor,
) (map[string]*storage.ProcessIndicator, []string, error) {

	var (
		indicatorLookups []*v1.Query
		indicatorIds     []string
		indicatorsMap    = map[string]*storage.ProcessIndicator{}
	)

	for _, val := range portProcesses {
		if val.Process == nil {
			log.Warnf("Got PLOP object without Process information, ignore: %+v", val)
			continue
		}

		indicatorLookups = append(indicatorLookups,
			search.NewQueryBuilder().
				AddExactMatches(search.ContainerName, val.Process.ContainerName).
				AddExactMatches(search.PodID, val.Process.PodId).
				AddExactMatches(search.ProcessName, val.Process.ProcessName).
				AddExactMatches(search.ProcessArguments, val.Process.ProcessArgs).
				AddExactMatches(search.ProcessExecPath, val.Process.ProcessExecFilePath).
				ProtoQuery())
	}

	indicatorsQuery := search.DisjunctionQuery(indicatorLookups...)
	log.Debugf("Sending query: %s", indicatorsQuery.String())
	indicators, err := ds.indicatorDataStore.SearchRawProcessIndicators(ctx, indicatorsQuery)
	if err != nil {
		return nil, nil, err
	}

	for _, val := range indicators {
		key := fmt.Sprintf("%s %s %s %s %s",
			val.GetContainerName(),
			val.GetPodId(),
			val.GetSignal().GetName(),
			val.GetSignal().GetArgs(),
			val.GetSignal().GetExecFilePath(),
		)

		// A bit of paranoia is always good
		if old, ok := indicatorsMap[key]; ok {
			log.Warnf("An indicator %s is already present, overwrite with %s",
				old.GetId(), val.GetId())
		}

		indicatorsMap[key] = val
		indicatorIds = append(indicatorIds, val.GetId())
	}

	return indicatorsMap, indicatorIds, nil
}

// OpenClosedPLOPs is a convenient type alias to use in PLOP normalization
type OpenClosedPLOPs struct {
	plop   *storage.ProcessListeningOnPortFromSensor
	nopen	int
	nclosed	int
}

// normalizePLOPs
//
// In the batch of PLOP events there could be many open & close events for the
// same combination of port, protocol, process. Find and squash them into a
// single event.
//
// Open/close state will be calculated from the totall number of open/close
// events in the batch, assuming every single open will eventually be followed
// by close. In this way in the case of:
// * Out-of-order events, we would be able to establish correct status
// * Pairs split across two batches, the status will be correct after processing both batches
// * Lost events, the status will be incorrect
// Another alternative would be to set the status based on the final PLOP
// event, which will produce the same results for case 2 and 3. But such
// approach will produce incorrect status in the case 1 as well, so counting
// seems to be more preferrable.
func normalizePLOPs(
	plops []*storage.ProcessListeningOnPortFromSensor,
) []*OpenClosedPLOPs {


	normalizedMap := map[string]OpenClosedPLOPs{}
	normalizedResult := []*OpenClosedPLOPs{}

	for _, val := range plops {
		key := fmt.Sprintf("%d %d %s",
			val.GetProtocol(),
			val.GetPort(),
			getProcesUniqueKey(val),
		)

		if prev, ok := normalizedMap[key]; ok {

			if val.GetCloseTimestamp() == nil {
				prev.nopen++
			} else {
				prev.nclosed++
				if val.CloseTimestamp.Compare(prev.plop.CloseTimestamp) == 1 {
					prev.plop = val
				}
			}

			normalizedMap[key] = prev

		} else {

			newValue := OpenClosedPLOPs{}

			if val.GetCloseTimestamp() == nil {
				newValue.nclosed = 0
				newValue.nopen = 1
			} else {
				newValue.nclosed = 1
				newValue.nopen = 0
			}

			newValue.plop = val
			normalizedMap[key] = newValue
		}
	}

	for _, value := range normalizedMap {
		normalizedResult = append(normalizedResult, &value)
	}

	return normalizedResult
}

func getProcesUniqueKey(plop *storage.ProcessListeningOnPortFromSensor) string {
	return fmt.Sprintf("%s %s %s %s %s",
		plop.Process.ContainerName,
		plop.Process.PodId,
		plop.Process.ProcessName,
		plop.Process.ProcessArgs,
		plop.Process.ProcessExecFilePath,
	)
}
