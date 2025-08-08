package datastore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	alertutils "github.com/stackrox/rox/central/alert/utils"
	"github.com/stackrox/rox/central/metrics"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/alert/convert"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	searchCommon "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/sync"
)

const whenUnlimited = 100

var (
	log = logging.LoggerForModule()

	alertSAC = sac.ForResource(resources.Alert)
)

// datastoreImpl is a transaction script with methods that provide the domain logic for CRUD uses cases for Alert
// objects.
type datastoreImpl struct {
	storage         store.Store
	keyedMutex      *concurrency.KeyedMutex
	keyFence        concurrency.KeyFence
	platformMatcher platformmatcher.PlatformMatcher
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query, excludeResolved bool) ([]searchCommon.Result, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "Search")

	if excludeResolved {
		q = applyDefaultState(q)
	}
	return ds.storage.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query, excludeResolved bool) (int, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "Count")

	if excludeResolved {
		q = applyDefaultState(q)
	}
	return ds.storage.Count(ctx, q)
}

func (ds *datastoreImpl) SearchListAlerts(ctx context.Context, q *v1.Query, excludeResolved bool) ([]*storage.ListAlert, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "SearchListAlerts")

	if excludeResolved {
		q = applyDefaultState(q)
	}
	listAlerts := make([]*storage.ListAlert, 0, paginated.GetLimit(q.GetPagination().GetLimit(), whenUnlimited))
	err := ds.storage.GetByQueryFn(ctx, q, func(alert *storage.Alert) error {
		listAlerts = append(listAlerts, convert.AlertToListAlert(alert))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return listAlerts, nil
}

// SearchAlerts returns search results for the given request. This will exclude resolved alerts by default unless Violation State = Resolved is explicitly specified in the query
func (ds *datastoreImpl) SearchAlerts(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "SearchAlerts")

	results, err := ds.Search(ctx, q, true)
	if err != nil {
		return nil, err
	}
	alerts, missingIndices, err := ds.storage.GetMany(ctx, searchCommon.ResultsToIDs(results))
	if err != nil {
		return nil, err
	}
	listAlerts := make([]*storage.ListAlert, 0, len(alerts))
	for _, alert := range alerts {
		listAlerts = append(listAlerts, convert.AlertToListAlert(alert))
	}
	results = searchCommon.RemoveMissingResults(results, missingIndices)

	protoResults := make([]*v1.SearchResult, 0, len(alerts))
	for i, alert := range listAlerts {
		protoResults = append(protoResults, convertAlert(alert, results[i]))
	}
	return protoResults, nil
}

// SearchRawAlerts returns search results for the given request in the form of a slice of alerts.
func (ds *datastoreImpl) SearchRawAlerts(ctx context.Context, q *v1.Query, excludeResolved bool) ([]*storage.Alert, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "SearchRawAlerts")

	if excludeResolved {
		q = applyDefaultState(q)
	}

	alerts := make([]*storage.Alert, 0, paginated.GetLimit(q.GetPagination().GetLimit(), whenUnlimited))
	err := ds.storage.GetByQueryFn(ctx, q, func(alert *storage.Alert) error {
		alerts = append(alerts, alert)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to search alerts")
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
	activeQuery := searchCommon.NewQueryBuilder().AddExactMatches(searchCommon.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery()
	return ds.Count(ctx, activeQuery, true)
}

// UpsertAlert inserts an alert into storage
func (ds *datastoreImpl) UpsertAlert(ctx context.Context, alert *storage.Alert) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "UpsertAlert")

	if ok, err := alertSAC.WriteAllowed(ctx, sacKeyForAlert(alert)...); err != nil || !ok {
		return sac.ErrResourceAccessDenied
	}

	ds.keyedMutex.Lock(alert.GetId())
	defer ds.keyedMutex.Unlock(alert.GetId())

	return ds.updateAlertNoLock(ctx, alert)
}

// UpdateAlertBatch updates an alert in storage
func (ds *datastoreImpl) UpdateAlertBatch(ctx context.Context, alert *storage.Alert, waitGroup *sync.WaitGroup, c chan error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "UpdateAlertBatch")

	defer waitGroup.Done()
	ds.keyedMutex.Lock(alert.GetId())
	defer ds.keyedMutex.Unlock(alert.GetId())
	// Avoid `ds.GetAlert` since that leads to extra read SAC check in addition to read-write check below.
	oldAlert, exists, err := ds.storage.Get(ctx, alert.GetId())
	if err != nil {
		log.Errorf("error in get alert: %+v", err)
		c <- err
		return
	}
	if exists {
		ok, err := alertSAC.WriteAllowed(ctx, sacKeyForAlert(alert)...)
		if err != nil {
			c <- err
			return
		}
		if !ok {
			c <- sac.ErrResourceAccessDenied
			return
		}

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

// UpsertAlerts updates an alert in storage
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

func (ds *datastoreImpl) MarkAlertsResolvedBatch(ctx context.Context, ids ...string) ([]*storage.Alert, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "MarkAlertsResolvedBatch")

	resolvedAt := protocompat.TimestampNow()
	idsAsBytes := make([][]byte, 0, len(ids))
	for _, id := range ids {
		idsAsBytes = append(idsAsBytes, []byte(id))
	}

	var resolvedAlerts []*storage.Alert
	err := ds.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(idsAsBytes...), func() error {
		var err error
		var missing []int
		// Avoid `ds.GetAlert` since that leads to extra read SAC check in addition to read-write check below.
		resolvedAlerts, missing, err = ds.storage.GetMany(ctx, ids)
		if err != nil {
			return err
		}
		if len(missing) > 0 {
			// Warn and continue marking the found alerts stale instead of returning error.
			// Marking alerts stale essentially removes the alerts from APIs by default anyway.
			log.Warnf("%d/%d alert(s) to be marked stale do not exist", len(missing), len(ids))
		}

		for _, alert := range resolvedAlerts {
			ok, err := alertSAC.WriteAllowed(ctx, sacKeyForAlert(alert)...)
			if err != nil {
				return err
			}
			if !ok {
				return sac.ErrResourceAccessDenied
			}

			alert.State = storage.ViolationState_RESOLVED
			alert.ResolvedAt = resolvedAt
		}

		return ds.updateAlertNoLock(ctx, resolvedAlerts...)
	})
	return resolvedAlerts, err
}

func (ds *datastoreImpl) DeleteAlerts(ctx context.Context, ids ...string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "DeleteAlerts")

	if ok, err := alertSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := ds.storage.DeleteMany(ctx, ids); err != nil {
		return errors.Wrap(err, "deleting alert")
	}
	return nil
}

func (ds *datastoreImpl) PruneAlerts(ctx context.Context, ids ...string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Alert", "PruneAlerts")

	if ok, err := alertSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := ds.storage.PruneMany(ctx, ids); err != nil {
		return errors.Wrap(err, "pruning alert")
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

func (ds *datastoreImpl) updateAlertNoLock(ctx context.Context, alerts ...*storage.Alert) error {
	if len(alerts) == 0 {
		return nil
	}

	if features.PlatformComponents.Enabled() {
		for _, alert := range alerts {
			alert.EntityType = alertutils.GetEntityType(alert)
			match, err := ds.platformMatcher.MatchAlert(alert)
			if err != nil {
				return err
			}
			alert.PlatformComponent = match
		}
	}

	return ds.storage.UpsertMany(ctx, alerts)
}

func hasSameScope(o1, o2 sac.NamespaceScopedObject) bool {
	return o1 != nil && o2 != nil && o1.GetClusterId() == o2.GetClusterId() && o1.GetNamespace() == o2.GetNamespace()
}

func (ds *datastoreImpl) WalkByQuery(ctx context.Context, q *v1.Query, fn func(alert *storage.Alert) error) error {
	return ds.storage.GetByQueryFn(ctx, q, fn)
}

func (ds *datastoreImpl) WalkAll(ctx context.Context, fn func(*storage.ListAlert) error) error {
	if ok, err := alertSAC.ReadAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	walkFn := func() error {
		return ds.storage.Walk(ctx, func(alert *storage.Alert) error {
			listAlert := convert.AlertToListAlert(alert)
			return fn(listAlert)
		})
	}
	return pgutils.RetryIfPostgres(ctx, walkFn)
}

// DefaultStateAlertDataStoreImpl will only return unresolved alerts unless Violation State=Resolved is explicitly provided by the query
type DefaultStateAlertDataStoreImpl struct {
	DataStore *DataStore
}

func (ds *DefaultStateAlertDataStoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchCommon.Result, error) {
	return (*ds.DataStore).Search(ctx, q, true)
}

func (ds *DefaultStateAlertDataStoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return (*ds.DataStore).Count(ctx, q, true)
}

func applyDefaultState(q *v1.Query) *v1.Query {
	// By default, set stale to false.
	querySpecifiesStateField := false
	searchCommon.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}
		if matchFieldQuery.MatchFieldQuery.GetField() == searchCommon.ViolationState.String() {
			querySpecifiesStateField = true
		}
	})

	if !querySpecifiesStateField {
		cq := searchCommon.ConjunctionQuery(q, searchCommon.NewQueryBuilder().AddExactMatches(
			searchCommon.ViolationState,
			storage.ViolationState_ACTIVE.String(),
			storage.ViolationState_ATTEMPTED.String()).ProtoQuery())
		cq.Pagination = q.GetPagination()
		return cq
	}
	return q
}

// convertAlert returns proto search result from an alert object and the internal search result
func convertAlert(alert *storage.ListAlert, result searchCommon.Result) *v1.SearchResult {
	entityInfo := alert.GetCommonEntityInfo()
	var entityName string
	switch entity := alert.GetEntity().(type) {
	case *storage.ListAlert_Resource:
		entityName = entity.Resource.GetName()
	case *storage.ListAlert_Deployment:
		entityName = entity.Deployment.GetName()
	}
	resourceTypeTitleCase := strings.Title(strings.ToLower(entityInfo.GetResourceType().String()))
	var location string
	if entityInfo.GetNamespace() != "" {
		location = fmt.Sprintf("/%s/%s/%s/%s",
			entityInfo.GetClusterName(), entityInfo.GetNamespace(), resourceTypeTitleCase, entityName)
	} else {
		location = fmt.Sprintf("/%s/%s/%s",
			entityInfo.GetClusterName(), resourceTypeTitleCase, entityName)
	}
	return &v1.SearchResult{
		Category:       v1.SearchCategory_ALERTS,
		Id:             alert.GetId(),
		Name:           alert.GetPolicy().GetName(),
		FieldToMatches: searchCommon.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
		Location:       location,
	}
}
