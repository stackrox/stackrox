package search

import (
	"fmt"

	"github.com/stackrox/rox/central/alert/index"
	"github.com/stackrox/rox/central/alert/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

const (
	alertBatchSize = 1000
)

var (
	log = logging.LoggerForModule()
)

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage store.Store
	indexer index.Indexer
}

func (ds *searcherImpl) buildIndex() error {
	defer debug.FreeOSMemory()

	stateAlerts, err := ds.storage.GetAlertStates()
	if err != nil {
		return err
	}

	var resolvedIDs, unresolvedIDs []string
	for _, a := range stateAlerts {
		if a.GetState() == storage.ViolationState_RESOLVED {
			resolvedIDs = append(resolvedIDs, a.GetId())
		} else {
			unresolvedIDs = append(unresolvedIDs, a.GetId())
		}
	}

	if err := ds.getAndIndexAlertsBatch(unresolvedIDs); err != nil {
		return err
	}

	// Asynchronously index the resolved alerts because there is no hard
	// dependency on resolved alerts being indexed
	go func() {
		if err := ds.getAndIndexAlertsBatch(resolvedIDs); err != nil {
			log.Error(err)
		}
	}()
	return nil
}

func (ds *searcherImpl) getAndIndexAlertsBatch(ids []string) error {
	b := batcher.New(len(ids), alertBatchSize)
	for start, end, ok := b.Next(); ok; start, end, ok = b.Next() {
		if err := ds.getAndIndexAlerts(ids[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (ds *searcherImpl) getAndIndexAlerts(ids []string) error {
	defer debug.FreeOSMemory()
	alerts, err := ds.storage.GetAlerts(ids...)
	if err != nil {
		return err
	}
	if err := ds.indexer.AddAlerts(alerts); err != nil {
		return err
	}
	return nil
}

// SearchAlerts retrieves SearchResults from the indexer and storage
func (ds *searcherImpl) SearchAlerts(q *v1.Query) ([]*v1.SearchResult, error) {
	alerts, results, err := ds.searchListAlerts(q)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(alerts))
	for i, alert := range alerts {
		protoResults = append(protoResults, convertAlert(alert, results[i]))
	}
	return protoResults, nil
}

// SearchRawAlerts retrieves Alerts from the indexer and storage
func (ds *searcherImpl) SearchListAlerts(q *v1.Query) ([]*storage.ListAlert, error) {
	alerts, _, err := ds.searchListAlerts(q)
	return alerts, err
}

// SearchRawAlerts retrieves Alerts from the indexer and storage
func (ds *searcherImpl) SearchRawAlerts(q *v1.Query) ([]*storage.Alert, error) {
	alerts, err := ds.searchAlerts(q)
	return alerts, err
}

func (ds *searcherImpl) searchListAlerts(q *v1.Query) ([]*storage.ListAlert, []search.Result, error) {
	results, err := ds.indexer.Search(q)
	if err != nil {
		return nil, nil, err
	}
	var alerts []*storage.ListAlert
	var newResults []search.Result
	for _, result := range results {
		alert, exists, err := ds.storage.ListAlert(result.ID)
		if err != nil {
			return nil, nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		alerts = append(alerts, alert)
		newResults = append(newResults, result)
	}
	return alerts, newResults, nil
}

func (ds *searcherImpl) searchAlerts(q *v1.Query) ([]*storage.Alert, error) {
	results, err := ds.indexer.Search(q)
	if err != nil {
		return nil, err
	}
	var alerts []*storage.Alert
	for _, result := range results {
		alert, exists, err := ds.storage.GetAlert(result.ID)
		if err != nil {
			return nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		alerts = append(alerts, alert)
	}
	return alerts, nil
}

// ConvertAlert returns proto search result from an alert object and the internal search result
func convertAlert(alert *storage.ListAlert, result search.Result) *v1.SearchResult {
	deployment := alert.GetDeployment()
	return &v1.SearchResult{
		Category:       v1.SearchCategory_ALERTS,
		Id:             alert.GetId(),
		Name:           alert.GetPolicy().GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
		Location:       fmt.Sprintf("/%s/%s/%s", deployment.GetClusterName(), deployment.GetNamespace(), deployment.GetName()),
	}
}
