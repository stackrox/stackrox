package datastore

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/processbaseline/index"
	"github.com/stackrox/rox/central/processbaseline/search"
	"github.com/stackrox/rox/central/processbaseline/store"
	processBaselineResultsStore "github.com/stackrox/rox/central/processbaselineresults/datastore"
	processIndicatorDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	processBaselinePkg "github.com/stackrox/rox/pkg/processbaseline"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

var (
	processBaselineSAC = sac.ForResource(resources.ProcessWhitelist)

	genDuration = env.BaselineGenerationDuration.DurationSetting()
)

type datastoreImpl struct {
	storage      store.Store
	indexer      index.Indexer
	searcher     search.Searcher
	baselineLock *concurrency.KeyedMutex

	processBaselineResults processBaselineResultsStore.DataStore
	processesDataStore     processIndicatorDatastore.DataStore
}

func (ds *datastoreImpl) SearchRawProcessBaselines(ctx context.Context, q *v1.Query) ([]*storage.ProcessBaseline, error) {
	return ds.searcher.SearchRawProcessBaselines(ctx, q)
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds *datastoreImpl) GetProcessBaseline(ctx context.Context, key *storage.ProcessBaselineKey) (*storage.ProcessBaseline, bool, error) {
	if ok, err := processBaselineSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(key).Allowed(); err != nil || !ok {
		return nil, false, err
	}
	id, err := keyToID(key)
	if err != nil {
		return nil, false, err
	}
	processBaseline, exists, err := ds.storage.Get(ctx, id)
	if err != nil || !exists {
		return nil, false, err
	}
	return processBaseline, exists, nil
}

func (ds *datastoreImpl) AddProcessBaseline(ctx context.Context, baseline *storage.ProcessBaseline) (string, error) {
	if ok, err := processBaselineSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(baseline.GetKey()).Allowed(); err != nil {
		return "", err
	} else if !ok {
		return "", sac.ErrResourceAccessDenied
	}

	id, err := keyToID(baseline.GetKey())
	if err != nil {
		return "", err
	}
	ds.baselineLock.Lock(id)
	defer ds.baselineLock.Unlock(id)
	return ds.addProcessBaselineUnlocked(ctx, id, baseline)
}

func (ds *datastoreImpl) addProcessBaselineUnlocked(ctx context.Context, id string, baseline *storage.ProcessBaseline) (string, error) {
	baseline.Id = id
	baseline.Created = types.TimestampNow()
	baseline.LastUpdate = baseline.GetCreated()
	baseline.StackRoxLockedTimestamp = ds.generateLockTimestamp()

	if err := ds.storage.Upsert(ctx, baseline); err != nil {
		return id, errors.Wrapf(err, "inserting process baseline %q into store", baseline.GetId())
	}
	if err := ds.indexer.AddProcessBaseline(baseline); err != nil {
		err = errors.Wrapf(err, "inserting process baseline %q into index", baseline.GetId())
		subErr := ds.storage.Delete(ctx, id)
		if subErr != nil {
			err = errors.Wrap(err, "error rolling back process process baseline addition")
		}
		return id, err
	}
	return id, nil
}

func (ds *datastoreImpl) addProcessBaselineLocked(ctx context.Context, baseline *storage.ProcessBaseline) (string, error) {
	if err := ds.storage.Upsert(ctx, baseline); err != nil {
		return baseline.GetId(), errors.Wrapf(err, "inserting process baseline %q into store", baseline.GetId())
	}
	if err := ds.indexer.AddProcessBaseline(baseline); err != nil {
		err = errors.Wrapf(err, "inserting process baseline %q into index", baseline.GetId())
		subErr := ds.storage.Delete(ctx, baseline.GetId())
		if subErr != nil {
			err = errors.Wrap(err, "error rolling back process process baseline addition")
		}
		return baseline.GetId(), err
	}
	return baseline.GetId(), nil
}

func (ds *datastoreImpl) removeProcessBaselineByID(ctx context.Context, id string) error {
	ds.baselineLock.Lock(id)
	defer ds.baselineLock.Unlock(id)
	if err := ds.indexer.DeleteProcessBaseline(id); err != nil {
		return errors.Wrap(err, "error removing process baseline from index")
	}
	if err := ds.storage.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "error removing process baseline from store")
	}
	return nil
}

func (ds *datastoreImpl) removeProcessBaselineResults(ctx context.Context, deploymentID string) error {
	if err := ds.processBaselineResults.DeleteBaselineResults(ctx, deploymentID); err != nil {
		return errors.Wrap(err, "removing process baseline results")
	}
	return nil
}

func (ds *datastoreImpl) RemoveProcessBaseline(ctx context.Context, key *storage.ProcessBaselineKey) error {
	if ok, err := processBaselineSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(key).Allowed(); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	id, err := keyToID(key)
	if err != nil {
		return err
	}
	if err := ds.removeProcessBaselineByID(ctx, id); err != nil {
		return err
	}
	// Delete process baseline results if this is the last process baseline with the given deploymentID
	deploymentID := key.GetDeploymentId()
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.DeploymentID, deploymentID).ProtoQuery()
	results, err := ds.indexer.Search(q)
	if err != nil {
		return errors.Wrapf(err, "failed to query for deployment %s during process baseline deletion", deploymentID)
	}
	if len(results) == 0 {
		return ds.removeProcessBaselineResults(ctx, deploymentID)
	}
	return nil
}

func (ds *datastoreImpl) RemoveProcessBaselinesByDeployment(ctx context.Context, deploymentID string) error {
	if ok, err := processBaselineSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	query := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.DeploymentID, deploymentID).ProtoQuery()
	results, err := ds.indexer.Search(query)
	if err != nil {
		return err
	}

	var errList []error
	for _, result := range results {
		err := ds.removeProcessBaselineByID(ctx, result.ID)
		if err != nil {
			errList = append(errList, err)
		}
	}

	if err := ds.removeProcessBaselineResults(ctx, deploymentID); err != nil {
		errList = append(errList, err)
	}

	if len(errList) > 0 {
		return errorhelpers.NewErrorListWithErrors("errors cleaning up process baselines", errList).ToError()
	}

	return nil
}

func (ds *datastoreImpl) getBaselineForUpdate(ctx context.Context, id string) (*storage.ProcessBaseline, error) {
	baseline, exists, err := ds.storage.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Errorf("no process baseline with id %q", id)
	}
	return baseline, nil
}

func makeElementMap(elementList []*storage.BaselineElement) map[string]*storage.BaselineElement {
	elementMap := make(map[string]*storage.BaselineElement, len(elementList))
	for _, listItem := range elementList {
		elementMap[listItem.GetElement().GetProcessName()] = listItem
	}
	return elementMap
}

func makeElementList(elementMap map[string]*storage.BaselineElement) []*storage.BaselineElement {
	elementList := make([]*storage.BaselineElement, 0, len(elementMap))
	for _, process := range elementMap {
		elementList = append(elementList, process)
	}
	return elementList
}

func (ds *datastoreImpl) updateProcessBaselineAndSetTimestamp(ctx context.Context, baseline *storage.ProcessBaseline) error {
	baseline.LastUpdate = types.TimestampNow()
	return ds.storage.Upsert(ctx, baseline)
}

func (ds *datastoreImpl) updateProcessBaselineElements(ctx context.Context, baseline *storage.ProcessBaseline, addElements []*storage.BaselineItem, removeElements []*storage.BaselineItem, auto bool) (*storage.ProcessBaseline, error) {
	baselineMap := makeElementMap(baseline.GetElements())
	graveyardMap := makeElementMap(baseline.GetElementGraveyard())

	for _, element := range addElements {
		// Don't automatically add anything which has been previously removed
		if _, ok := graveyardMap[element.GetProcessName()]; auto && ok {
			continue
		}
		existing, ok := baselineMap[element.GetProcessName()]
		if !ok || existing.Auto {
			delete(graveyardMap, element.GetProcessName())
			baselineMap[element.GetProcessName()] = &storage.BaselineElement{
				Element: element,
				Auto:    auto,
			}
		}
	}

	for _, removeElement := range removeElements {
		delete(baselineMap, removeElement.GetProcessName())
		existing, ok := graveyardMap[removeElement.GetProcessName()]
		if !ok || existing.Auto {
			graveyardMap[removeElement.GetProcessName()] = &storage.BaselineElement{
				Element: removeElement,
				Auto:    auto,
			}
		}
	}

	baseline.Elements = makeElementList(baselineMap)
	baseline.ElementGraveyard = makeElementList(graveyardMap)

	err := ds.updateProcessBaselineAndSetTimestamp(ctx, baseline)
	if err != nil {
		return nil, err
	}

	// no need to index the process baseline here because the only indexed things are
	// top level fields that are immutable
	return baseline, nil
}

func (ds *datastoreImpl) UpdateProcessBaselineElements(ctx context.Context, key *storage.ProcessBaselineKey, addElements []*storage.BaselineItem, removeElements []*storage.BaselineItem, auto bool) (*storage.ProcessBaseline, error) {
	if ok, err := processBaselineSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(key).Allowed(); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	id, err := keyToID(key)
	if err != nil {
		return nil, err
	}

	ds.baselineLock.Lock(id)
	defer ds.baselineLock.Unlock(id)

	baseline, err := ds.getBaselineForUpdate(ctx, id)
	if err != nil {
		return nil, err
	}

	return ds.updateProcessBaselineElements(ctx, baseline, addElements, removeElements, auto)
}

func (ds *datastoreImpl) UpsertProcessBaseline(ctx context.Context, key *storage.ProcessBaselineKey, addElements []*storage.BaselineItem, auto bool, lock bool) (*storage.ProcessBaseline, error) {
	if ok, err := processBaselineSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(key).Allowed(); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	id, err := keyToID(key)
	if err != nil {
		return nil, err
	}

	ds.baselineLock.Lock(id)
	defer ds.baselineLock.Unlock(id)

	baseline, exists, err := ds.GetProcessBaseline(ctx, key)
	if err != nil {
		return nil, err
	}

	if exists {
		return ds.updateProcessBaselineElements(ctx, baseline, addElements, nil, auto)
	}

	timestamp := types.TimestampNow()
	var elements []*storage.BaselineElement
	for _, element := range addElements {
		elements = append(elements, &storage.BaselineElement{Element: &storage.BaselineItem{Item: &storage.BaselineItem_ProcessName{ProcessName: element.GetProcessName()}}, Auto: auto})
	}

	baseline = &storage.ProcessBaseline{
		Id:                      id,
		Key:                     key,
		Elements:                elements,
		Created:                 timestamp,
		LastUpdate:              timestamp,
		StackRoxLockedTimestamp: timestamp,
	}
	if lock {
		_, err = ds.addProcessBaselineLocked(ctx, baseline)
	} else {
		_, err = ds.addProcessBaselineUnlocked(ctx, id, baseline)
	}
	if err != nil {
		return nil, err
	}
	return baseline, nil
}

func (ds *datastoreImpl) UserLockProcessBaseline(ctx context.Context, key *storage.ProcessBaselineKey, locked bool) (*storage.ProcessBaseline, error) {
	if ok, err := processBaselineSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(key).Allowed(); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	id, err := keyToID(key)
	if err != nil {
		return nil, err
	}
	ds.baselineLock.Lock(id)
	defer ds.baselineLock.Unlock(id)

	baseline, err := ds.getBaselineForUpdate(ctx, id)
	if err != nil {
		return nil, err
	}

	if locked && baseline.GetUserLockedTimestamp() == nil {
		baseline.UserLockedTimestamp = types.TimestampNow()
		err = ds.updateProcessBaselineAndSetTimestamp(ctx, baseline)
	} else if !locked && baseline.GetUserLockedTimestamp() != nil {
		baseline.UserLockedTimestamp = nil
		err = ds.updateProcessBaselineAndSetTimestamp(ctx, baseline)
	}
	if err != nil {
		return nil, err
	}
	return baseline, nil
}

func (ds *datastoreImpl) CreateUnlockedProcessBaseline(ctx context.Context, key *storage.ProcessBaselineKey) (*storage.ProcessBaseline, error) {
	if ok, err := processBaselineSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(key).Allowed(); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	id, err := keyToID(key)
	if err != nil {
		return nil, err
	}
	ds.baselineLock.Lock(id)
	defer ds.baselineLock.Unlock(id)

	// See if we already have a baseline
	baseline, exists, err := ds.GetProcessBaseline(ctx, key)
	if err != nil {
		return nil, err
	}

	if exists {
		return baseline, nil
	}

	// Get the list of processes
	baselineList, err := ds.getProcessList(ctx, key)
	if err != nil {
		return nil, err
	}

	// Build the de-duped elements for the baseline
	elements := make(map[string]*storage.BaselineItem, len(baselineList))

	for _, indicator := range baselineList {
		baselineItem := processBaselinePkg.BaselineItemFromProcess(indicator)

		insertableElement := &storage.BaselineItem{Item: &storage.BaselineItem_ProcessName{ProcessName: baselineItem}}

		elements[baselineItem] = insertableElement
	}

	baseElements := make([]*storage.BaselineElement, 0, len(elements))
	for _, element := range elements {
		baseElements = append(baseElements, &storage.BaselineElement{Element: &storage.BaselineItem{Item: &storage.BaselineItem_ProcessName{ProcessName: element.GetProcessName()}}, Auto: true})
	}

	// Build the baseline itself
	timestamp := types.TimestampNow()
	baseline = &storage.ProcessBaseline{
		Id:         id,
		Key:        key,
		Elements:   baseElements,
		Created:    timestamp,
		LastUpdate: timestamp,
	}

	// Store the unlocked baseline.
	_, err = ds.addProcessBaselineUnlocked(ctx, id, baseline)

	return baseline, err
}

func (ds *datastoreImpl) getProcessList(ctx context.Context, key *storage.ProcessBaselineKey) ([]*storage.ProcessIndicator, error) {
	// Simply use search to find the process indicators for the baseline key
	return ds.processesDataStore.SearchRawProcessIndicators(
		ctx,
		pkgSearch.NewQueryBuilder().
			AddExactMatches(pkgSearch.DeploymentID, key.GetDeploymentId()).
			AddExactMatches(pkgSearch.ContainerName, key.GetContainerName()).
			AddExactMatches(pkgSearch.Cluster, key.GetClusterId()).
			ProtoQuery(),
	)
}

func (ds *datastoreImpl) WalkAll(ctx context.Context, fn func(baseline *storage.ProcessBaseline) error) error {
	if ok, err := processBaselineSAC.ReadAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return ds.storage.Walk(ctx, fn)
}

func (ds *datastoreImpl) RemoveProcessBaselinesByIDs(ctx context.Context, ids []string) error {
	for _, id := range ids {
		key, err := IDToKey(id)
		if err != nil {
			return err
		}
		if ok, err := processBaselineSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(key).Allowed(); err != nil {
			return err
		} else if !ok {
			return sac.ErrResourceAccessDenied
		}
		if err := ds.removeProcessBaselineByID(ctx, id); err != nil {
			return errors.Wrapf(err, "removing baseline %s", id)
		}
	}
	return nil
}

func (ds *datastoreImpl) ClearProcessBaselines(ctx context.Context, ids []string) error {
	// Get all the baselines we are interested in
	baselines, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return err
	}

	// Go through the baselines and clear them out
	for _, baseline := range baselines {
		if ok, err := processBaselineSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(baseline.Key).Allowed(); err != nil {
			return err
		} else if !ok {
			return sac.ErrResourceAccessDenied
		}

		baseline.Elements = nil
		baseline.ElementGraveyard = nil

		// We need to extend the stackrox lock timestamp to re-observe the processes.
		baseline.StackRoxLockedTimestamp = ds.generateLockTimestamp()
		baseline.LastUpdate = types.TimestampNow()
	}
	return ds.storage.UpsertMany(ctx, baselines)
}

func (ds *datastoreImpl) generateLockTimestamp() *types.Timestamp {
	lockTimestamp, err := types.TimestampProto(time.Now().Add(genDuration))
	// This should not occur unless genDuration is in a bad state.  If that happens just
	// set it to one hour in the future.
	if err != nil {
		lockTimestamp, _ = types.TimestampProto(time.Now().Add(1 * time.Hour))
	}
	return lockTimestamp
}
