package datastore

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/stackrox/rox/central/metrics"
	countMetrics "github.com/stackrox/rox/central/metrics"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processlisteningonport/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/process/id"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
)

type datastoreImpl struct {
	storage            store.Store
	indicatorDataStore processIndicatorStore.DataStore
}

var (
	plopSAC = sac.ForResource(resources.DeploymentExtension)
	log     = logging.LoggerForModule()
)

func newDatastoreImpl(
	storage store.Store,
	indicatorDataStore processIndicatorStore.DataStore,
) *datastoreImpl {
	return &datastoreImpl{
		storage:            storage,
		indicatorDataStore: indicatorDataStore,
	}
}

func checkIfShouldUpdate(
	existingPLOP *storage.ProcessListeningOnPortStorage,
	newPLOP *storage.ProcessListeningOnPortFromSensor) bool {

	return existingPLOP.CloseTimestamp != newPLOP.CloseTimestamp ||
		(existingPLOP.PodUid == "" && newPLOP.PodUid != "")
}

func getIndicatorIDForPlop(plop *storage.ProcessListeningOnPortFromSensor) string {
	if plop == nil {
		log.Warn("Plop is nil. Unable to set process indicator id. Plop will not appear in the API")
		return ""
	}

	if plop.Process == nil {
		log.Infof("Plop process is nil. Unable to set process indicator id. Plop will not appear in the API. plop: %s", plopToNoSecretsString(plop))
		return ""
	}

	return id.GetIndicatorIDFromProcessIndicatorUniqueKey(plop.Process)
}

func getIndicatorIdsForPlops(plops []*storage.ProcessListeningOnPortFromSensor) []string {
	indicatorIds := make([]string, 0)
	for _, plop := range plops {
		indicatorID := getIndicatorIDForPlop(plop)
		indicatorIds = append(indicatorIds, indicatorID)
	}

	return indicatorIds
}

func processToNoSecretsString(process *storage.ProcessIndicatorUniqueKey) string {
	if process == nil {
		return ""
	}

	return fmt.Sprintf("%s_%s_%s_%s",
		process.GetContainerName(),
		process.GetPodId(),
		process.GetProcessName(),
		process.GetProcessExecFilePath(),
	)
}

func plopToNoSecretsString(plop *storage.ProcessListeningOnPortFromSensor) string {
	if plop == nil {
		return ""
	}

	return fmt.Sprintf("%d_%d_%s", plop.GetProtocol(), plop.GetPort(), processToNoSecretsString(plop.Process))
}

func plopStorageToNoSecretsString(plop *storage.ProcessListeningOnPortStorage) string {
	if plop == nil {
		return ""
	}

	return fmt.Sprintf("%d_%d_%s", plop.GetProtocol(), plop.GetPort(), processToNoSecretsString(plop.Process))
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
	if ok, err := plopSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	normalizedPLOPs, completedInBatch := normalizePLOPs(portProcesses)
	allPLOPs := append(normalizedPLOPs, completedInBatch...)
	indicatorIds := getIndicatorIdsForPlops(allPLOPs)

	// Errors are not handled, because we can still report information on the plop without a
	// matching process in the process_indicator table.
	indicators, nonempty, err := ds.indicatorDataStore.GetProcessIndicators(ctx, indicatorIds)
	indicatorsMap := make(map[string]bool)

	// Used to do best efforts of identifying orphaned PLOP. Note, that as
	// usual, this could be done together with fetchExistingPLOPs in the same
	// custom query.
	if nonempty && err == nil {
		for _, ind := range indicators {
			indicatorsMap[ind.GetId()] = true
		}
	}

	existingPLOPMap, err := ds.fetchExistingPLOPs(ctx, indicatorIds)
	if err != nil {
		return err
	}

	plopObjects := []*storage.ProcessListeningOnPortStorage{}
	for _, val := range normalizedPLOPs {
		log.Infof("normalizedPLOP= %+v", val)
		var processInfo *storage.ProcessIndicatorUniqueKey

		indicatorID := getIndicatorIDForPlop(val)
		if indicatorID == "" {
			log.Infof("Unable to set indicatorID. Plop will not appear in the API. %s", plopToNoSecretsString(val))
			continue
		}
		plopKey := getPlopKeyFromParts(val.GetProtocol(), val.GetPort(), indicatorID)

		existingPLOP, prevExists := existingPLOPMap[plopKey]

		// Best effort to not duplicate data. If no process indicator with
		// such an id exists, we deal with a potentially orphaned PLOP, and
		// need to store the process information. Otherwise processInfo is nil
		// and will not be stored.
		if _, indicatorExists := indicatorsMap[indicatorID]; !indicatorExists {
			countMetrics.IncrementOrphanedPLOPCounter(val.GetClusterId())
			log.Infof("Found no matching indicators for %s", plopToNoSecretsString(val))
			processInfo = val.GetProcess()
		}

		// There are three options:
		// * We found an existing PLOP object with different close timestamp or non-empty PodUid
		//   It has to be updated.
		// * We found an existing PLOP object with the same close timestamp.
		//   Nothing has to be changed (XXX: Ideally it has to be excluded from
		//   the upsert later on).
		// * No existing PLOP object, create a new one with whatever close
		//   timestamp we have received and fetched indicator ID.
		if prevExists && checkIfShouldUpdate(existingPLOP, val) {
			log.Infof("Got existing PLOP: %s", plopStorageToNoSecretsString(existingPLOP))

			// Update the timestamp and PodUid
			existingPLOP.CloseTimestamp = val.CloseTimestamp
			existingPLOP.Closed = existingPLOP.CloseTimestamp != nil
			existingPLOP.PodUid = val.PodUid
			plopObjects = append(plopObjects, existingPLOP)
		}

		if !prevExists {
			plopObjects = addNewPLOP(plopObjects, indicatorID, processInfo, val)
		}
	}

	// Verify what to do about pairs of open/close events that close the
	// lifecycle within the batch. There are only few options:
	// * If an existing open PLOP is present in the db, they will do nothing
	// * If an existing closed PLOP is present in the db, they will update the
	// timestamp
	// * If no existing PLOP is present, they will create a new closed PLOP
	for _, val := range completedInBatch {
		log.Info("completedInBatch %+v", val)
		var processInfo *storage.ProcessIndicatorUniqueKey

		indicatorID := getIndicatorIDForPlop(val)
		if indicatorID == "" {
			log.Infof("Unable to set indicatorID. Plop will not appear in the API. %s", plopToNoSecretsString(val))
			continue
		}
		plopKey := getPlopKeyFromParts(val.GetProtocol(), val.GetPort(), indicatorID)

		existingPLOP, prevExists := existingPLOPMap[plopKey]

		// Best effort to not duplicate data. If no process indicator with
		// such an id exists, we deal with a potentially orphaned PLOP, and
		// need to store the process information. Otherwise processInfo is nil
		// and will not be stored.
		if _, indicatorExists := indicatorsMap[indicatorID]; !indicatorExists {
			countMetrics.IncrementOrphanedPLOPCounter(val.GetClusterId())
			log.Infof("Found no matching indicators for %s", plopToNoSecretsString(val))
			processInfo = val.GetProcess()
		}

		if prevExists && checkIfShouldUpdate(existingPLOP, val) {
			log.Infof("Got existing PLOP: %s", plopStorageToNoSecretsString(existingPLOP))

			// Update the timestamp and PodUid
			if existingPLOP.Closed {
				existingPLOP.CloseTimestamp = val.CloseTimestamp
			}
			existingPLOP.PodUid = val.PodUid
			plopObjects = append(plopObjects, existingPLOP)
		}

		if !prevExists {
			if val.CloseTimestamp == nil {
				// This events should always be closing by definition
				log.Infof("Found active PLOP completed in the batch %s", plopToNoSecretsString(val))
			}

			plopObjects = addNewPLOP(plopObjects, indicatorID, processInfo, val)
		}
	}

	for _, plop := range plopObjects {
		log.Infof("plop to be inserted= %+v", plop)
	}

	// Now save actual PLOP objects
	return ds.storage.UpsertMany(ctx, plopObjects)
}

func (ds *datastoreImpl) GetProcessListeningOnPort(
	ctx context.Context,
	deploymentID string,
) (
	processesListeningOnPorts []*storage.ProcessListeningOnPort, err error,
) {
	if ok, err := plopSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	processesListeningOnPorts, err = ds.storage.GetProcessListeningOnPort(ctx, deploymentID)

	if err != nil {
		log.Infof("In GetProcessListeningOnPort. Query for deployment %s returned err: %+v", deploymentID, err)
		return nil, err
	}

	if processesListeningOnPorts == nil {
		log.Infof("In GetProcessListeningOnPort. Query for deployment %s returned nil", deploymentID)
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

func (ds *datastoreImpl) RemoveProcessListeningOnPort(ctx context.Context, ids []string) error {
	if ok, err := plopSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.removePLOP(ctx, ids)
}

func (ds *datastoreImpl) removePLOP(ctx context.Context, ids []string) error {

	if len(ids) == 0 {
		return nil
	}

	return ds.storage.DeleteMany(ctx, ids)
}

// fetchExistingPLOPs: Query already existing PLOP objects belonging to the
// specified process indicators.
//
// XXX: This function queries all PLOP, no matter if they are matching port +
// protocol we've got or not. This means potentially dangerous corner cases
// when one process listens to a huge number of ports. To address it we could
// introduce filtering by port and protocol to the query, and even without
// extra indices PostgreSQL will be able to do it relatively efficiently using
// bitmap scan.
func (ds *datastoreImpl) fetchExistingPLOPs(
	ctx context.Context,
	indicatorIds []string,
) (map[string]*storage.ProcessListeningOnPortStorage, error) {

	var existingPLOPMap = map[string]*storage.ProcessListeningOnPortStorage{}

	if len(indicatorIds) == 0 {
		return existingPLOPMap, nil
	}

	// If no corresponding processes found, we can't verify if the PLOP
	// object is opening/closing an existing one. Collect existingPLOPMap
	// only if there are some matching indicators.
	existingPLOPs, err := ds.storage.GetByQuery(ctx, search.NewQueryBuilder().
		AddStrings(search.ProcessID, indicatorIds...).ProtoQuery())
	if err != nil {
		return nil, err
	}

	for _, val := range existingPLOPs {
		key := getPlopKey(val)

		// A bit of paranoia is always good
		if old, ok := existingPLOPMap[key]; ok {
			log.Infof("A PLOP %s is already present, overwrite with %s",
				old.GetId(), val.GetId())
		}

		existingPLOPMap[key] = val
	}

	return existingPLOPMap, nil
}

// OpenClosedPLOPs is a convenient type alias to use in PLOP normalization
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
// Open/close state will be calculated from the total number of open/close
// events in the batch, assuming every single open will eventually be followed
// by close. In this way in the case of:
// * Out-of-order events, we would be able to establish correct status
// * Pairs split across two batches, the status will be correct after processing both batches
// * Lost events, the status will be incorrect
//
// A special case is when the batch has equal number of open/close events. For
// such cases the agreement is they do not contribute anything for already
// existing PLOP events, and produce a closed PLOP event if nothing is found in
// the db.
//
// Another alternative would be to set the status based on the final PLOP
// event, which will produce the same results for case 2 and 3. But such
// approach will produce incorrect status in the case 1 as well, so counting
// seems to be more preferrable.
//
// The function returns two slices of PLOP events, the first one contains
// events that have to change existing PLOP status, the second one contains
// those events that have to be verified against existing PLOP events (i.e.
// every open has matching close whithin the batch.
func normalizePLOPs(
	plops []*storage.ProcessListeningOnPortFromSensor,
) (normalizedResult []*storage.ProcessListeningOnPortFromSensor,
	completedEvents []*storage.ProcessListeningOnPortFromSensor,
) {

	normalizedMap := map[string]OpenClosedPLOPs{}
	normalizedResult = []*storage.ProcessListeningOnPortFromSensor{}
	completedEvents = []*storage.ProcessListeningOnPortFromSensor{}

	for _, val := range plops {
		key := getPlopKeyFromParts(
			val.GetProtocol(),
			val.GetPort(),
			getPlopProcessUniqueKey(val),
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
		sortByCloseTimestamp(value.open)
		sortByCloseTimestamp(value.closed)
		nOpen := len(value.open)
		nClosed := len(value.closed)

		if nOpen == nClosed {
			completedEvents = append(completedEvents, value.closed[nClosed-1])
			continue
		}

		// Take the last open PLOP if there are more open in total, otherwise
		// the last closed PLOP
		if nOpen > nClosed {
			normalizedResult = append(normalizedResult, value.open[nOpen-1])
		} else {
			normalizedResult = append(normalizedResult, value.closed[nClosed-1])
		}
	}

	return normalizedResult, completedEvents
}

func getProcessUniqueKeyFromParts(containerName string,
	podID string,
	processName string,
	processArgs string,
	processExecFilePath string,
) string {
	return fmt.Sprintf("%s_%s_%s_%s_%s",
		containerName,
		podID,
		processName,
		processArgs,
		processExecFilePath,
	)
}

func getPlopProcessUniqueKey(plop *storage.ProcessListeningOnPortFromSensor) string {
	return getProcessUniqueKeyFromParts(
		plop.Process.ContainerName,
		plop.Process.PodId,
		plop.Process.ProcessName,
		plop.Process.ProcessArgs,
		plop.Process.ProcessExecFilePath,
	)
}

func getPlopKeyFromParts(protocol storage.L4Protocol, port uint32, indicatorID string) string {
	return fmt.Sprintf("%d_%d_%s",
		protocol,
		port,
		indicatorID,
	)
}

func getPlopKey(plop *storage.ProcessListeningOnPortStorage) string {
	return getPlopKeyFromParts(plop.GetProtocol(), plop.GetPort(), plop.GetProcessIndicatorId())
}

func sortByCloseTimestamp(values []*storage.ProcessListeningOnPortFromSensor) {
	sort.Slice(values, func(i, j int) bool {
		return values[i].GetCloseTimestamp().Compare(values[j].GetCloseTimestamp()) == -1
	})
}

func addNewPLOP(plopObjects []*storage.ProcessListeningOnPortStorage,
	indicatorID string,
	processInfo *storage.ProcessIndicatorUniqueKey,
	value *storage.ProcessListeningOnPortFromSensor) []*storage.ProcessListeningOnPortStorage {

	if value == nil || indicatorID == "" {
		log.Infof("Unable to insert plop object. Info from sensor= %s\nindicatorID= %s\nprocessInfo= %s", plopToNoSecretsString(value), indicatorID, processToNoSecretsString(processInfo))
		return plopObjects
	}

	newPLOP := &storage.ProcessListeningOnPortStorage{
		// XXX, ResignatingFacepalm: Use regular GENERATE ALWAYS AS
		// IDENTITY, which would require changes in store generator
		Id:                 uuid.NewV4().String(),
		Port:               value.Port,
		Protocol:           value.Protocol,
		ProcessIndicatorId: indicatorID,
		Process:            processInfo,
		DeploymentId:       value.DeploymentId,
		PodUid:             value.PodUid,
		Closed:             value.CloseTimestamp != nil,
		CloseTimestamp:     value.CloseTimestamp,
	}

	return append(plopObjects, newPLOP)
}

func (ds *datastoreImpl) RemovePlopsByPod(ctx context.Context, id string) error {
	if ok, err := plopSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	q := search.NewQueryBuilder().AddExactMatches(search.PodUID, id).ProtoQuery()
	return ds.storage.DeleteByQuery(ctx, q)
}
