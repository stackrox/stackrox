package datastore

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/alert/datastore/internal/index"
	"github.com/stackrox/rox/central/alert/datastore/internal/search"
	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/alert/convert"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	searchCommon "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	alertSAC = sac.ForResource(resources.Alert)
)

const (
	alertBatchSize = 1000
)

// datastoreImpl is a transaction script with methods that provide the domain logic for CRUD uses cases for Alert
// objects.
type datastoreImpl struct {
	storage    store.Store
	indexer    index.Indexer
	searcher   search.Searcher
	keyedMutex *concurrency.KeyedMutex
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchCommon.Result, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "Search")

	return ds.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "Count")

	return ds.searcher.Count(ctx, q)
}

func (ds *datastoreImpl) SearchListAlerts(ctx context.Context, q *v1.Query) ([]*storage.ListAlert, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "SearchListAlerts")

	return ds.searcher.SearchListAlerts(ctx, q)
}

// SearchAlerts returns search results for the given request.
func (ds *datastoreImpl) SearchAlerts(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "SearchAlerts")

	return ds.searcher.SearchAlerts(ctx, q)
}

// SearchRawAlerts returns search results for the given request in the form of a slice of alerts.
func (ds *datastoreImpl) SearchRawAlerts(ctx context.Context, q *v1.Query) ([]*storage.Alert, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "SearchRawAlerts")

	return ds.searcher.SearchRawAlerts(ctx, q)
}

func (ds *datastoreImpl) ListAlerts(ctx context.Context, request *v1.ListAlertsRequest) ([]*storage.ListAlert, error) {
	var q *v1.Query
	if request.GetQuery() == "" {
		q = searchCommon.EmptyQuery()
	} else {
		var err error
		q, err = searchCommon.ParseQuery(request.GetQuery())
		if err != nil {
			return nil, err
		}
	}

	paginated.FillPagination(q, request.GetPagination(), math.MaxInt32)

	alerts, err := ds.SearchListAlerts(ctx, q)
	if err != nil {
		return nil, err
	}
	return alerts, nil
}

// GetAlert returns an alert by id.
func (ds *datastoreImpl) GetAlert(ctx context.Context, id string) (*storage.Alert, bool, error) {
	alert, exists, err := ds.storage.Get(ctx, id)
	if err != nil || !exists {
		return nil, false, err
	}

	if ok, err := alertSAC.ReadAllowed(ctx, sacKeyForAlert(alert)...); err != nil || !ok {
		return nil, false, err
	}
	return alert, true, nil
}

// CountAlerts returns the number of alerts that are active
func (ds *datastoreImpl) CountAlerts(ctx context.Context) (int, error) {
	activeQuery := searchCommon.NewQueryBuilder().AddStrings(searchCommon.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery()
	return ds.Count(ctx, activeQuery)
}

// UpsertAlert inserts an alert into storage and into the indexer
func (ds *datastoreImpl) UpsertAlert(ctx context.Context, alert *storage.Alert) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "UpsertAlert")

	if ok, err := alertSAC.WriteAllowed(ctx, sacKeyForAlert(alert)...); err != nil || !ok {
		return sac.ErrResourceAccessDenied
	}

	ds.keyedMutex.Lock(alert.GetId())
	defer ds.keyedMutex.Unlock(alert.GetId())

	return ds.updateAlertNoLock(ctx, alert)
}

// UpdateAlertBatch updates an alert in storage and in the indexer
func (ds *datastoreImpl) UpdateAlertBatch(ctx context.Context, alert *storage.Alert, waitGroup *sync.WaitGroup, c chan error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "UpdateAlertBatch")

	defer waitGroup.Done()
	ds.keyedMutex.Lock(alert.GetId())
	defer ds.keyedMutex.Unlock(alert.GetId())
	oldAlert, exists, err := ds.GetAlert(ctx, alert.GetId())
	if err != nil {
		log.Errorf("error in get alert: %+v", err)
		c <- err
		return
	}
	if exists {
		if !hasSameScope(getNSScopedObjectFromAlert(alert), getNSScopedObjectFromAlert(oldAlert)) {
			c <- fmt.Errorf("cannot change the cluster or namespace of an existing alert %q", alert.GetId())
			return
		}
	}
	err = ds.updateAlertNoLock(ctx, alert)
	if err != nil {
		c <- err
	}
}

// UpsertAlerts updates an alert in storage and in the indexer
func (ds *datastoreImpl) UpsertAlerts(ctx context.Context, alertBatch []*storage.Alert) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "UpsertAlerts")

	var waitGroup sync.WaitGroup
	c := make(chan error, len(alertBatch))
	for _, alert := range alertBatch {
		waitGroup.Add(1)
		go ds.UpdateAlertBatch(ctx, alert, &waitGroup, c)
	}
	waitGroup.Wait()
	close(c)
	if len(c) > 0 {
		errorList := errorhelpers.NewErrorList(fmt.Sprintf("found %d errors while resolving alerts", len(c)))
		for err := range c {
			errorList.AddError(err)
		}
		return errorList.ToError()
	}
	return nil
}

func (ds *datastoreImpl) MarkAlertStale(ctx context.Context, id string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "MarkAlertStale")

	ds.keyedMutex.Lock(id)
	defer ds.keyedMutex.Unlock(id)

	alert, exists, err := ds.GetAlert(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("alert with id '%s' does not exist", id)
	}

	if ok, err := alertSAC.WriteAllowed(ctx, sacKeyForAlert(alert)...); err != nil || !ok {
		return sac.ErrResourceAccessDenied
	}
	alert.State = storage.ViolationState_RESOLVED
	alert.ResolvedAt = types.TimestampNow()
	return ds.updateAlertNoLock(ctx, alert)
}

func (ds *datastoreImpl) DeleteAlerts(ctx context.Context, ids ...string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "DeleteAlerts")

	if ok, err := alertSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	errorList := errorhelpers.NewErrorList("deleting alert")
	if err := ds.storage.DeleteMany(ctx, ids); err != nil {
		errorList.AddError(err)
	}
	if err := ds.indexer.DeleteListAlerts(ids); err != nil {
		errorList.AddError(err)
	}
	if err := ds.storage.AckKeysIndexed(ctx, ids...); err != nil {
		errorList.AddError(err)
	}

	return errorList.ToError()
}

func (ds *datastoreImpl) AddAlertTags(ctx context.Context, resourceID string, tags []string) ([]string, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "AddAlertTags")

	ds.keyedMutex.Lock(resourceID)
	defer ds.keyedMutex.Unlock(resourceID)

	alert, exists, err := ds.storage.Get(ctx, resourceID)
	if err != nil {
		return nil, errors.Wrapf(err, "error fetching alert %q from the DB", resourceID)
	}
	if !exists {
		return nil, fmt.Errorf("cannot add tags to alert %q that no longer exists", resourceID)
	}
	if ok, err := alertSAC.WriteAllowed(ctx, sacKeyForAlert(alert)...); err != nil || !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	allTags := sliceutils.StringUnion(alert.GetTags(), tags)
	sort.Strings(allTags)
	alert.Tags = allTags
	if err := ds.updateAlertNoLock(ctx, alert); err != nil {
		return nil, errors.Wrapf(err, "error upserting alert %q", alert.GetId())
	}

	return alert.GetTags(), nil
}

func (ds *datastoreImpl) RemoveAlertTags(ctx context.Context, resourceID string, tags []string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "DeleteAlertTags")

	ds.keyedMutex.Lock(resourceID)
	defer ds.keyedMutex.Unlock(resourceID)

	alert, exists, err := ds.storage.Get(ctx, resourceID)
	if err != nil {
		return errors.Wrapf(err, "error fetching alert %q from the DB", resourceID)
	}
	if !exists {
		return fmt.Errorf("cannot add tags to alert %q that no longer exists", resourceID)
	}
	if ok, err := alertSAC.WriteAllowed(ctx, sacKeyForAlert(alert)...); err != nil || !ok {
		return sac.ErrResourceAccessDenied
	}

	remainingTags := sliceutils.StringDifference(alert.GetTags(), tags)
	sort.Strings(remainingTags)

	if len(remainingTags) == 0 {
		alert.Tags = nil
	} else {
		alert.Tags = remainingTags
	}
	if err := ds.updateAlertNoLock(ctx, alert); err != nil {
		return fmt.Errorf("error upserting alert %q", alert.GetId())
	}

	return nil
}

func sacKeyForAlert(alert *storage.Alert) []sac.ScopeKey {
	scopedObj := getNSScopedObjectFromAlert(alert)
	if scopedObj == nil {
		return sac.GlobalScopeKey()
	}
	return sac.KeyForNSScopedObj(scopedObj)
}

func getNSScopedObjectFromAlert(alert *storage.Alert) sac.NamespaceScopedObject {
	switch alert.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		return alert.GetDeployment()
	case *storage.Alert_Resource_:
		return alert.GetResource()
	case *storage.Alert_Image:
		return nil // This is theoretically possible even though image doesn't have a ns/cluster
	default:
		log.Errorf("UNEXPECTED: Alert Entity %s unknown", alert.GetEntity())
	}
	return nil
}

func copyScopingInformationToObjectRoot(alert *storage.Alert) {
	switch alert.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		entity := alert.GetDeployment()
		alert.ClusterId = entity.ClusterId
		alert.ClusterName = entity.ClusterName
		alert.Namespace = entity.Namespace
		alert.NamespaceId = entity.NamespaceId
	case *storage.Alert_Resource_:
		entity := alert.GetResource()
		alert.ClusterId = entity.ClusterId
		alert.ClusterName = entity.ClusterName
		alert.Namespace = entity.Namespace
		alert.NamespaceId = entity.NamespaceId
	case *storage.Alert_Image:
		// Image doesn't have a ns/cluster
		alert.ClusterId = ""
		alert.ClusterName = ""
		alert.Namespace = ""
		alert.NamespaceId = ""
	default:
		log.Errorf("UNEXPECTED: Alert Entity %s unknown", alert.GetEntity())
	}
}

func (ds *datastoreImpl) updateAlertNoLock(ctx context.Context, alert *storage.Alert) error {
	// Checks pass then update.
	copyScopingInformationToObjectRoot(alert)
	if err := ds.storage.Upsert(ctx, alert); err != nil {
		return err
	}
	if err := ds.indexer.AddListAlert(fillSortHelperFields(convert.AlertToListAlert(alert))); err != nil {
		return err
	}
	return ds.storage.AckKeysIndexed(ctx, alert.GetId())
}

func hasSameScope(o1, o2 sac.NamespaceScopedObject) bool {
	return o1 != nil && o2 != nil && o1.GetClusterId() == o2.GetClusterId() && o1.GetNamespace() == o2.GetNamespace()
}

func (ds *datastoreImpl) fullReindex(ctx context.Context) error {
	log.Info("[STARTUP] Reindexing all alerts")

	alertIDs, err := ds.storage.GetIDs(ctx)
	if err != nil {
		return err
	}
	log.Infof("[STARTUP] Found %d alerts to index", len(alertIDs))
	alertBatcher := batcher.New(len(alertIDs), alertBatchSize)
	for start, end, valid := alertBatcher.Next(); valid; start, end, valid = alertBatcher.Next() {
		listAlerts, _, err := ds.getListAlerts(ctx, alertIDs[start:end])
		if err != nil {
			return err
		}
		if err := ds.indexer.AddListAlerts(listAlerts); err != nil {
			return err
		}
		if end%(alertBatchSize*10) == 0 {
			log.Infof("[STARTUP] Successfully indexed %d/%d alerts", end, len(alertIDs))
		}
	}
	log.Infof("[STARTUP] Successfully indexed %d alerts", len(alertIDs))

	// Clear the keys because we just re-indexed everything
	keys, err := ds.storage.GetKeysToIndex(ctx)
	if err != nil {
		return err
	}
	if err := ds.storage.AckKeysIndexed(ctx, keys...); err != nil {
		return err
	}

	// Write out that initial indexing is complete
	if err := ds.indexer.MarkInitialIndexingComplete(); err != nil {
		return err
	}

	return nil
}

func (ds *datastoreImpl) buildIndex(ctx context.Context) error {
	if features.PostgresDatastore.Enabled() {
		return nil
	}
	defer debug.FreeOSMemory()

	needsFullIndexing, err := ds.indexer.NeedsInitialIndexing()
	if err != nil {
		return err
	}
	if needsFullIndexing {
		return ds.fullReindex(ctx)
	}

	log.Info("[STARTUP] Determining if alert db/indexer reconciliation is needed")
	keysToIndex, err := ds.storage.GetKeysToIndex(ctx)
	if err != nil {
		return errors.Wrap(err, "error retrieving keys to index from store")
	}

	log.Infof("[STARTUP] Found %d Alerts to index", len(keysToIndex))

	defer debug.FreeOSMemory()

	alertBatcher := batcher.New(len(keysToIndex), alertBatchSize)
	for start, end, valid := alertBatcher.Next(); valid; start, end, valid = alertBatcher.Next() {
		listAlerts, missingIndices, err := ds.getListAlerts(ctx, keysToIndex[start:end])
		if err != nil {
			return err
		}
		if err := ds.indexer.AddListAlerts(listAlerts); err != nil {
			return err
		}

		if len(missingIndices) > 0 {
			idsToRemove := make([]string, 0, len(missingIndices))
			for _, missingIdx := range missingIndices {
				idsToRemove = append(idsToRemove, keysToIndex[start:end][missingIdx])
			}
			if err := ds.indexer.DeleteListAlerts(idsToRemove); err != nil {
				return err
			}
		}
		// Ack keys so that even if central restarts, we don't need to reindex them again
		if err := ds.storage.AckKeysIndexed(ctx, keysToIndex[start:end]...); err != nil {
			return err
		}

		log.Infof("[STARTUP] Successfully indexed %d/%d alerts", end, len(keysToIndex))
	}

	log.Info("[STARTUP] Successfully indexed all out of sync alerts")
	return nil
}

func (ds *datastoreImpl) getListAlerts(ctx context.Context, ids []string) ([]*storage.ListAlert, []int, error) {
	alerts, missingIndices, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	listAlerts := make([]*storage.ListAlert, 0, len(ids))
	for _, alert := range alerts {
		listAlerts = append(listAlerts, convert.AlertToListAlert(alert))
	}
	return listAlerts, missingIndices, nil
}

func (ds *datastoreImpl) WalkAll(ctx context.Context, fn func(*storage.ListAlert) error) error {
	if ok, err := alertSAC.ReadAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.Walk(ctx, func(alert *storage.Alert) error {
		listAlert := convert.AlertToListAlert(alert)
		return fn(listAlert)
	})
}
