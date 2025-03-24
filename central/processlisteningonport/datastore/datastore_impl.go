package datastore

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/central/metrics"
	countMetrics "github.com/stackrox/rox/central/metrics"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processlisteningonport/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/process/id"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

type datastoreImpl struct {
	storage            store.Store
	indicatorDataStore processIndicatorStore.DataStore
	mutex              sync.RWMutex
	pool               postgres.DB
}

var (
	plopSAC = sac.ForResource(resources.DeploymentExtension)
	log     = logging.LoggerForModule()
)

func newDatastoreImpl(
	storage store.Store,
	indicatorDataStore processIndicatorStore.DataStore,
	pool postgres.DB,
) *datastoreImpl {
	return &datastoreImpl{
		storage:            storage,
		indicatorDataStore: indicatorDataStore,
		pool:               pool,
	}
}

func checkIfShouldUpdate(
	existingPLOP *storage.ProcessListeningOnPortStorage,
	newPLOP *storage.ProcessListeningOnPortFromSensor) bool {

	return existingPLOP.CloseTimestamp != newPLOP.CloseTimestamp ||
		(existingPLOP.PodUid == "" && newPLOP.PodUid != "") ||
		(existingPLOP.ClusterId == "" && newPLOP.ClusterId != "") ||
		(existingPLOP.Namespace == "" && newPLOP.Namespace != "")
}

func getIndicatorIDForPlop(plop *storage.ProcessListeningOnPortFromSensor) string {
	if plop == nil {
		log.Warn("Plop is nil. Unable to set process indicator id. Plop will not appear in the API")
		return ""
	}

	if plop.Process == nil {
		log.Warnf("Plop process is nil. Unable to set process indicator id. Plop will not appear in the API. plop: %s", plopToNoSecretsString(plop))
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
	clusterID string,
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

	newPlopObjects := []*storage.ProcessListeningOnPortStorage{}
	updatePlopObjects := []*storage.ProcessListeningOnPortStorage{}
	for _, val := range normalizedPLOPs {
		val.ClusterId = clusterID
		var processInfo *storage.ProcessIndicatorUniqueKey

		indicatorID := getIndicatorIDForPlop(val)
		if indicatorID == "" {
			log.Warnf("Unable to set indicatorID. Plop will not appear in the API. %s", plopToNoSecretsString(val))
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
			log.Debugf("Found no matching indicators for %s", plopToNoSecretsString(val))
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
			log.Debugf("Got existing PLOP: %s", plopStorageToNoSecretsString(existingPLOP))

			// Update the timestamp and PodUid
			existingPLOP.CloseTimestamp = val.CloseTimestamp
			existingPLOP.Closed = existingPLOP.CloseTimestamp != nil
			existingPLOP.PodUid = val.PodUid
			existingPLOP.ClusterId = val.ClusterId
			existingPLOP.Namespace = val.Namespace
			updatePlopObjects = append(updatePlopObjects, existingPLOP)
		}

		if !prevExists {
			newPlopObjects = addNewPLOP(newPlopObjects, indicatorID, processInfo, val)
		}
	}

	// Verify what to do about pairs of open/close events that close the
	// lifecycle within the batch. There are only few options:
	// * If an existing open PLOP is present in the db, they will do nothing
	// * If an existing closed PLOP is present in the db, they will update the
	// timestamp
	// * If no existing PLOP is present, they will create a new closed PLOP
	for _, val := range completedInBatch {
		val.ClusterId = clusterID
		var processInfo *storage.ProcessIndicatorUniqueKey

		indicatorID := getIndicatorIDForPlop(val)
		if indicatorID == "" {
			log.Warnf("Unable to set indicatorID. Plop will not appear in the API. %s", plopToNoSecretsString(val))
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
			log.Debugf("Found no matching indicators for %s", plopToNoSecretsString(val))
			processInfo = val.GetProcess()
		}

		if prevExists && checkIfShouldUpdate(existingPLOP, val) {
			log.Debugf("Got existing PLOP: %s", plopStorageToNoSecretsString(existingPLOP))

			// Update the timestamp and PodUid
			if existingPLOP.Closed {
				existingPLOP.CloseTimestamp = val.CloseTimestamp
			}
			existingPLOP.PodUid = val.PodUid
			existingPLOP.ClusterId = val.ClusterId
			existingPLOP.Namespace = val.Namespace
			updatePlopObjects = append(updatePlopObjects, existingPLOP)
		}

		if !prevExists {
			if val.CloseTimestamp == nil {
				// This events should always be closing by definition
				log.Warnf("Found active PLOP completed in the batch %s", plopToNoSecretsString(val))
			}

			newPlopObjects = addNewPLOP(newPlopObjects, indicatorID, processInfo, val)
		}
	}

	// Save new PLOP objects
	err = ds.storage.UpsertMany(ctx, newPlopObjects)
	if err != nil {
		return err
	}

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	// Update existing PLOP objects while using a lock
	return ds.storage.UpsertMany(ctx, updatePlopObjects)
}

func (ds *datastoreImpl) GetProcessListeningOnPort(
	ctx context.Context,
	deploymentID string,
) (
	processesListeningOnPorts []*storage.ProcessListeningOnPort, err error,
) {

	processesListeningOnPorts, err = ds.storage.GetProcessListeningOnPort(ctx, deploymentID)

	if err != nil {
		log.Warnf("In GetProcessListeningOnPort. Query for deployment %s returned err: %+v", deploymentID, err)
		return nil, err
	}

	if processesListeningOnPorts == nil {
		log.Debugf("In GetProcessListeningOnPort. Query for deployment %s returned nil", deploymentID)
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
			log.Warnf("A PLOP %s is already present, overwrite with %s",
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
		return protocompat.CompareTimestamps(values[i].GetCloseTimestamp(), values[j].GetCloseTimestamp()) == -1
	})
}

func addNewPLOP(plopObjects []*storage.ProcessListeningOnPortStorage,
	indicatorID string,
	processInfo *storage.ProcessIndicatorUniqueKey,
	value *storage.ProcessListeningOnPortFromSensor) []*storage.ProcessListeningOnPortStorage {

	if value == nil || indicatorID == "" {
		log.Warnf("Unable to insert plop object. Info from sensor= %s\nindicatorID= %s\nprocessInfo= %s", plopToNoSecretsString(value), indicatorID, processToNoSecretsString(processInfo))
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
		ClusterId:          value.ClusterId,
		Namespace:          value.Namespace,
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

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	q := search.NewQueryBuilder().AddExactMatches(search.PodUID, id).ProtoQuery()
	_, storeErr := ds.storage.DeleteByQuery(ctx, q)
	return storeErr
}

// PruneOrphanedPLOPs prunes old closed PLOPs and those without deployments or pods
func (ds *datastoreImpl) PruneOrphanedPLOPs(ctx context.Context, orphanWindow time.Duration) int64 {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	query := fmt.Sprintf(pruneOrphanedPLOPs, int(orphanWindow.Minutes()))
	commandTag, err := ds.pool.Exec(ctx, query)
	if err != nil {
		log.Errorf("failed to prune PLOP: %v", err)
	}

	// Delete processes listening on ports orphaned due to missing deployments
	if _, err := ds.pool.Exec(ctx, deleteOrphanedPLOPDeployments); err != nil {
		log.Errorf("failed to prune process listening on ports by deployment: %v", err)
	}

	// Delete processes listening on ports orphaned due to missing pods.
	if _, err := ds.pool.Exec(ctx, deleteOrphanedPLOPPodsWithPodUID); err != nil {
		log.Errorf("failed to prune process listening on ports by pods: %v", err)
	}

	return commandTag.RowsAffected()
}

// PruneOrphanedPLOPsByProcessIndicators prunes PLOPs that match process indicators without pods
func (ds *datastoreImpl) PruneOrphanedPLOPsByProcessIndicators(ctx context.Context, orphanWindow time.Duration) {
	// TODO(ROX-22443): Once it is guaranteed that all listening endpoints have PodUIDs, remove this function
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	// Delete processes listening on ports orphaned because process indicators are orphaned due to
	// missing deployments
	query := fmt.Sprintf(deleteOrphanedPLOPDeploymentsAndPI, int(orphanWindow.Minutes()))
	if _, err := ds.pool.Exec(ctx, query); err != nil {
		log.Errorf("failed to prune process listening on ports by deployment: %v", err)
	}

	// Delete processes listening on ports orphaned because process indicators are orphaned due to
	// missing pods.
	query = fmt.Sprintf(deleteOrphanedPLOPPods, int(orphanWindow.Minutes()))
	if _, err := ds.pool.Exec(ctx, query); err != nil {
		log.Errorf("failed to prune process listening on ports by pods: %v", err)
	}
}

func (ds *datastoreImpl) readRowsToFindPLOPsWithNoProcessInformation(rows pgx.Rows) ([]string, error) {
	var ids []string

	for rows.Next() {
		var serialized []byte

		if err := rows.Scan(&serialized); err != nil {
			return nil, pgutils.ErrNilIfNoRows(err)
		}

		var msg storage.ProcessListeningOnPortStorage
		if err := msg.UnmarshalVTUnsafe(serialized); err != nil {
			return nil, err
		}

		process := msg.GetProcess()

		if process == nil {
			ids = append(ids, msg.GetId())
		}
	}
	return ids, nil
}

// First gets PLOPs with no matching process indicators and then checks the
// serialized data to check for process information.
func (ds *datastoreImpl) getPLOPsToDelete(ctx context.Context) ([]string, error) {
	rows, err := ds.pool.Query(ctx, getPotentiallyOrphanedPLOPs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return ds.readRowsToFindPLOPsWithNoProcessInformation(rows)
}

// RemovePLOPsWithoutProcessIndicatorOrProcessInfo Finds PLOPs without a matching process indicator and no
// process information and deletes them. PLOPs can have no matching process indicators, but such PLOPs need
// to have process information.
func (ds *datastoreImpl) RemovePLOPsWithoutProcessIndicatorOrProcessInfo(ctx context.Context) (int64, error) {
	plopsToDelete, err := ds.getPLOPsToDelete(ctx)
	if err != nil {
		return 0, err
	}

	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	err = ds.storage.PruneMany(ctx, plopsToDelete)
	if err != nil {
		return 0, err
	}

	return int64(len(plopsToDelete)), nil
}

// Removes PLOPs without poduids between a range of ids.
func (ds *datastoreImpl) removePLOPsWithoutPodUIDOnePage(ctx context.Context, prevId string, nextId string) (int64, error) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	query := fmt.Sprintf(deletePLOPsWithoutPoduidInPage, prevId, nextId)
	commandTag, err := ds.pool.Exec(ctx, query)

	if err != nil {
		return 0, err
	}

	return commandTag.RowsAffected(), nil
}

// Given a set of rows with ids, returns the id of the last row. This is useful for pagination.
func (ds *datastoreImpl) getLastIdFromRows(ctx context.Context, rows pgx.Rows) (string, error) {
	id := ""

	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			return "", pgutils.ErrNilIfNoRows(err)
		}
	}

	return id, rows.Err()
}

// Given an id and a limit, returns the id a limit number of rows after the given id. This is useful
// for efficient pagination.
func (ds *datastoreImpl) getNextPageId(ctx context.Context, prevId string, limit int) (string, error) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	query := fmt.Sprintf(getLastIdFromPage, prevId, limit)
	rows, err := ds.pool.Query(ctx, query)

	if err != nil {
		// Do not be alarmed if the error is simply NoRows
		err = pgutils.ErrNilIfNoRows(err)
		if err != nil {
			log.Warnf("%s: %s", query, err)
		}
		return "", err
	}
	defer rows.Close()

	nextId, err := ds.getLastIdFromRows(ctx, rows)

	if err != nil {
		return "", err
	}

	return nextId, nil
}

func (ds *datastoreImpl) retryableRemovePLOPsWithoutPodUID(ctx context.Context) (int64, error) {
	limit := 10000
	totalRows := int64(0)

	prevId := "00000000-0000-0000-0000-000000000000"
	for {
		nextId, err := ds.getNextPageId(ctx, prevId, limit)
		if err != nil {
			return totalRows, err
		}
		if nextId == "" {
			break
		}
		nrows, err := ds.removePLOPsWithoutPodUIDOnePage(ctx, prevId, nextId)
		if err != nil {
			return totalRows, err
		}
		totalRows += nrows
		prevId = nextId
	}

	return totalRows, nil
}

func (ds *datastoreImpl) RemovePLOPsWithoutPodUID(ctx context.Context) (int64, error) {
	return pgutils.Retry2(ctx, func() (int64, error) {
		return ds.retryableRemovePLOPsWithoutPodUID(ctx)
	})
}
