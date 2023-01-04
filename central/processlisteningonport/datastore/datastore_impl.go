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

func (ds *datastoreImpl) AddProcessListeningOnPort(
	ctx context.Context,
	portProcesses ...*storage.ProcessListeningOnPortFromSensor,
) error {
	defer metrics.SetDatastoreFunctionDuration(
		time.Now(),
		"ProcessListeningOnPort",
		"AddProcessListeningOnPort",
	)

	var (
		indicatorLookups []*v1.Query
		indicatorIds     []string
		indicatorsMap    = map[string]*storage.ProcessIndicator{}
		existingPLOPMap  = map[string]*storage.ProcessListeningOnPortStorage{}
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

	portProcesses = normalizePLOPs(portProcesses)

	// First query all needed process indicators in one go
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
		return err
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

	if len(indicatorIds) > 0 {
		// If no corresponding processes found, we can't verify if the PLOP
		// object is closing an existing one. Collect existingPLOPMap only if
		// there are some matching indicators.

		// XXX: This has to be done in a single join query fetching both
		// ProcessIndicator and needed bits from PLOP.
		existingPLOP, err := ds.storage.GetByQuery(ctx, search.NewQueryBuilder().
			AddStrings(search.ProcessID, indicatorIds...).
			AddBools(search.Closed, false).
			ProtoQuery())
		if err != nil {
			return err
		}

		for _, val := range existingPLOP {
			key := fmt.Sprintf("%s %d %s",
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
	}

	plopObjects := make([]*storage.ProcessListeningOnPortStorage, len(portProcesses))
	for i, val := range portProcesses {
		indicatorID := ""
		var processInfo *storage.ProcessIndicatorUniqueKey

		key := fmt.Sprintf("%s %s %s %s %s",
			val.Process.ContainerName,
			val.Process.PodId,
			val.Process.ProcessName,
			val.Process.ProcessArgs,
			val.Process.ProcessExecFilePath,
		)

		if indicator, ok := indicatorsMap[key]; ok {
			indicatorID = indicator.GetId()
			log.Debugf("Got indicator %s: %+v", indicatorID, indicator)
		} else {
			// XXX: Create a metric for this
			log.Warnf("Found no matching indicators for %s", key)
			processInfo = val.Process
		}

		newPLOP := &storage.ProcessListeningOnPortStorage{
			// XXX, ResignatingFacepalm: Use regular GENERATE ALWAYS AS
			// IDENTITY, which would require changes in store generator
			Id:                 uuid.NewV4().String(),
			Port:               val.Port,
			Protocol:           val.Protocol,
			ProcessIndicatorId: indicatorID,
			Process:            processInfo,
			Closed:             false,
			CloseTimestamp:     val.CloseTimestamp,
		}

		if val.CloseTimestamp != nil {
			// We receive a closing PLOP information, check if it's present in
			// the database
			newPLOP.Closed = true

			plopKey := fmt.Sprintf("%s %d %s",
				val.GetProtocol(),
				val.GetPort(),
				indicatorID,
			)

			if activePLOP, ok := existingPLOPMap[plopKey]; ok {
				log.Debugf("Got active PLOP: %+v", activePLOP)

				activePLOP.Closed = true
				plopObjects[i] = activePLOP
			} else {
				log.Warnf("Found no matching PLOP to close for %s", key)
				plopObjects[i] = newPLOP
			}
		} else {
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

// OpenClosedPLOPs: Convenient type alias to use in PLOP normalization
type OpenClosedPLOPs struct {
	open   []*storage.ProcessListeningOnPortFromSensor
	closed []*storage.ProcessListeningOnPortFromSensor
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
) []*storage.ProcessListeningOnPortFromSensor {

	normalizedMap := map[string]OpenClosedPLOPs{}
	normalizedResult := []*storage.ProcessListeningOnPortFromSensor{}

	for _, val := range plops {
		key := fmt.Sprintf("%d %d %s",
			val.GetProtocol(),
			val.GetPort(),
			getProcesUniqueKey(val),
		)

		if prev, ok := normalizedMap[key]; ok {

			if val.GetCloseTimestamp() == nil {
				prev.open = append(prev.open, val)
			} else {
				prev.closed = append(prev.closed, val)
			}

			normalizedMap[key] = prev

		} else {

			newValue := OpenClosedPLOPs{
				open:   []*storage.ProcessListeningOnPortFromSensor{},
				closed: []*storage.ProcessListeningOnPortFromSensor{},
			}

			if val.GetCloseTimestamp() == nil {
				newValue.open = append(newValue.open, val)
			} else {
				newValue.closed = append(newValue.closed, val)
			}

			normalizedMap[key] = newValue
		}
	}

	for _, value := range normalizedMap {
		nOpen := len(value.open)
		nClosed := len(value.closed)

		// Take the last open PLOP if there are more open in total, otherwise
		// the last closed PLOP
		if nOpen > nClosed {
			normalizedResult = append(normalizedResult, value.open[nOpen-1])
		} else {
			normalizedResult = append(normalizedResult, value.closed[nClosed-1])
		}
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
