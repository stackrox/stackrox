package datastore

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/set"
	"github.com/golang/protobuf/ptypes"
)

var (
	logger = logging.LoggerForModule()
)

// DataStore is a wrapper around the flow of data
type DataStore struct {
	db.Storage // This is an embedded type so we don't have to override all functions. Indexing is a subset of Storage
	indexer    search.Indexer
}

// NewDataStore takes in a storage implementation and and indexer implementation
func NewDataStore(storage db.Storage, indexer search.Indexer) (*DataStore, error) {
	ds := &DataStore{
		Storage: storage,
		indexer: indexer,
	}
	if err := ds.loadDefaults(); err != nil {
		return nil, err
	}
	return ds, nil
}

// Close closes both the database and the indexer
func (ds *DataStore) Close() {
	ds.Storage.Close()
	ds.indexer.Close()
}

// AddAlert inserts an alert into storage and into the indexer
func (ds *DataStore) AddAlert(alert *v1.Alert) error {
	if err := ds.Storage.AddAlert(alert); err != nil {
		return err
	}
	return ds.indexer.AddAlert(alert)
}

// UpdateAlert updates an alert in storage and in the indexer
func (ds *DataStore) UpdateAlert(alert *v1.Alert) error {
	if err := ds.Storage.UpdateAlert(alert); err != nil {
		return err
	}
	return ds.indexer.AddAlert(alert)
}

// RemoveAlert removes an alert from the storage and the indexer
func (ds *DataStore) RemoveAlert(id string) error {
	if err := ds.Storage.RemoveAlert(id); err != nil {
		return err
	}
	return ds.indexer.DeleteAlert(id)
}

type severitiesWrap []v1.Severity

func (wrap severitiesWrap) asSet() map[v1.Severity]struct{} {
	output := make(map[v1.Severity]struct{})

	for _, s := range wrap {
		output[s] = struct{}{}
	}

	return output
}

// GetAlerts fetches the data from the database or searches for it based on the passed filters
func (ds *DataStore) GetAlerts(request *v1.GetAlertsRequest) ([]*v1.Alert, error) {
	var alerts []*v1.Alert
	var err error
	if request.GetQuery() == "" {
		alerts, err = ds.Storage.GetAlerts(request)
		if err != nil {
			return nil, err
		}
	} else {
		parsedQuery, err := search.ParseRawQuery(request.GetQuery())
		if err != nil {
			return nil, err
		}
		alerts, err = ds.SearchRawAlerts(parsedQuery)
		if err != nil {
			return nil, err
		}
	}
	sinceTime, sinceTimeErr := ptypes.Timestamp(request.GetSince())
	untilTime, untilTimeErr := ptypes.Timestamp(request.GetUntil())
	sinceStaleTime, sinceStaleTimeErr := ptypes.Timestamp(request.GetSinceStale())
	untilStaleTime, untilStaleTimeErr := ptypes.Timestamp(request.GetUntilStale())

	severitySet := severitiesWrap(request.GetSeverity()).asSet()
	categorySet := set.NewSetFromStringSlice(request.GetCategory())
	filtered := alerts[:0]

	for _, alert := range alerts {
		if len(request.GetStale()) == 1 && alert.GetStale() != request.GetStale()[0] {
			continue
		}
		if request.GetDeploymentId() != "" && request.GetDeploymentId() != alert.GetDeployment().GetId() {
			continue
		}

		if request.GetPolicyId() != "" && request.GetPolicyId() != alert.GetPolicy().GetId() {
			continue
		}

		if _, ok := severitySet[alert.GetPolicy().GetSeverity()]; len(severitySet) > 0 && !ok {
			continue
		}
		alertCategoriesSet := set.NewSetFromStringSlice(alert.GetPolicy().GetCategories())
		if categorySet.Cardinality() != 0 && categorySet.Intersect(alertCategoriesSet).Cardinality() == 0 {
			continue
		}
		if sinceTimeErr == nil && !sinceTime.IsZero() {
			if alertTime, alertTimeErr := ptypes.Timestamp(alert.GetTime()); alertTimeErr == nil && !sinceTime.Before(alertTime) {
				continue
			}
		}
		if untilTimeErr == nil && !untilTime.IsZero() {
			if alertTime, alertTimeErr := ptypes.Timestamp(alert.GetTime()); alertTimeErr == nil && !untilTime.After(alertTime) {
				continue
			}
		}
		if sinceStaleTimeErr == nil && !sinceStaleTime.IsZero() {
			if alertTime, alertTimeErr := ptypes.Timestamp(alert.GetTime()); alertTimeErr == nil && !sinceStaleTime.Before(alertTime) {
				continue
			}
		}
		if untilStaleTimeErr == nil && !untilStaleTime.IsZero() {
			if alertTime, alertTimeErr := ptypes.Timestamp(alert.GetTime()); alertTimeErr == nil && !untilStaleTime.After(alertTime) {
				continue
			}
		}
		filtered = append(filtered, alert)
	}
	// Sort by descending timestamp.
	sort.SliceStable(filtered, func(i, j int) bool {
		if sI, sJ := filtered[i].GetTime().GetSeconds(), filtered[j].GetTime().GetSeconds(); sI != sJ {
			return sI > sJ
		}
		return filtered[i].GetTime().GetNanos() > filtered[j].GetTime().GetNanos()
	})
	return filtered, nil
}

func (ds *DataStore) searchAlerts(request *v1.ParsedSearchRequest) ([]*v1.Alert, []search.Result, error) {
	results, err := ds.indexer.SearchAlerts(request)
	if err != nil {
		return nil, nil, err
	}
	var alerts []*v1.Alert
	var newResults []search.Result
	for _, result := range results {
		alert, exists, err := ds.GetAlert(result.ID)
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

// SearchAlerts retrieves SearchResults from the indexer and storage
func (ds *DataStore) SearchAlerts(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	alerts, results, err := ds.searchAlerts(request)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(alerts))
	for i, alert := range alerts {
		protoResults = append(protoResults, search.ConvertAlert(alert, results[i]))
	}
	return protoResults, nil
}

// SearchRawAlerts retrieves Alerts from the indexer and storage
func (ds *DataStore) SearchRawAlerts(request *v1.ParsedSearchRequest) ([]*v1.Alert, error) {
	alerts, _, err := ds.searchAlerts(request)
	return alerts, err
}

func (ds *DataStore) searchImages(request *v1.ParsedSearchRequest) ([]*v1.Image, []search.Result, error) {
	results, err := ds.indexer.SearchImages(request)
	if err != nil {
		return nil, nil, err
	}
	var images []*v1.Image
	var newResults []search.Result
	for _, result := range results {
		image, exists, err := ds.GetImage(result.ID)
		if err != nil {
			return nil, nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		images = append(images, image)
		newResults = append(newResults, result)
	}
	return images, newResults, nil
}

// SearchImages retrieves SearchResults from the indexer and storage
func (ds *DataStore) SearchImages(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	images, results, err := ds.searchImages(request)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(images))
	for i, image := range images {
		protoResults = append(protoResults, search.ConvertImage(image, results[i]))
	}
	return protoResults, nil
}

// SearchRawImages retrieves SearchResults from the indexer and storage
func (ds *DataStore) SearchRawImages(request *v1.ParsedSearchRequest) ([]*v1.Image, error) {
	images, _, err := ds.searchImages(request)
	return images, err
}

func (ds *DataStore) searchPolicies(request *v1.ParsedSearchRequest) ([]*v1.Policy, []search.Result, error) {
	results, err := ds.indexer.SearchPolicies(request)
	if err != nil {
		return nil, nil, err
	}
	var policies []*v1.Policy
	var newResults []search.Result
	for _, result := range results {
		policy, exists, err := ds.GetPolicy(result.ID)
		if err != nil {
			return nil, nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		policies = append(policies, policy)
		newResults = append(newResults, result)
	}
	return policies, newResults, nil
}

// SearchRawPolicies retrieves Policies from the indexer and storage
func (ds *DataStore) SearchRawPolicies(request *v1.ParsedSearchRequest) ([]*v1.Policy, error) {
	policies, _, err := ds.searchPolicies(request)
	return policies, err
}

// SearchPolicies retrieves SearchResults from the indexer and storage
func (ds *DataStore) SearchPolicies(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	policies, results, err := ds.searchPolicies(request)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(policies))
	for i, policy := range policies {
		protoResults = append(protoResults, search.ConvertPolicy(policy, results[i]))
	}
	return protoResults, nil
}

func (ds *DataStore) searchDeployments(request *v1.ParsedSearchRequest) ([]*v1.Deployment, []search.Result, error) {
	results, err := ds.indexer.SearchDeployments(request)
	if err != nil {
		return nil, nil, err
	}
	var deployments []*v1.Deployment
	var newResults []search.Result
	for _, result := range results {
		deployment, exists, err := ds.GetDeployment(result.ID)
		if err != nil {
			return nil, nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		deployments = append(deployments, deployment)
		newResults = append(newResults, result)
	}
	return deployments, newResults, nil
}

// SearchRawDeployments retrieves deployments from the indexer and storage
func (ds *DataStore) SearchRawDeployments(request *v1.ParsedSearchRequest) ([]*v1.Deployment, error) {
	deployments, _, err := ds.searchDeployments(request)
	return deployments, err
}

// SearchDeployments retrieves SearchResults from the indexer and storage
func (ds *DataStore) SearchDeployments(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	deployments, results, err := ds.searchDeployments(request)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(deployments))
	for i, deployment := range deployments {
		protoResults = append(protoResults, search.ConvertDeployment(deployment, results[i]))
	}
	return protoResults, nil
}

// AddDeployment adds a deployment into the storage and the indexer
func (ds *DataStore) AddDeployment(deployment *v1.Deployment) error {
	if err := ds.Storage.AddDeployment(deployment); err != nil {
		return err
	}
	return ds.indexer.AddDeployment(deployment)
}

// UpdateDeployment updates a deployment in the storage and the indexer
func (ds *DataStore) UpdateDeployment(deployment *v1.Deployment) error {
	if err := ds.Storage.UpdateDeployment(deployment); err != nil {
		return err
	}
	return ds.indexer.AddDeployment(deployment)
}

// RemoveDeployment removes a deployment from the storage and the indexer
func (ds *DataStore) RemoveDeployment(id string) error {
	if err := ds.Storage.RemoveDeployment(id); err != nil {
		return err
	}
	return ds.indexer.DeleteDeployment(id)
}

// AddPolicy inserts a policy into the storage and the indexer
func (ds *DataStore) AddPolicy(policy *v1.Policy) (string, error) {
	id, err := ds.Storage.AddPolicy(policy)
	if err != nil {
		return id, err
	}
	return id, ds.indexer.AddPolicy(policy)
}

// UpdatePolicy updates a policy from the storage and the indexer
func (ds *DataStore) UpdatePolicy(policy *v1.Policy) error {
	if err := ds.Storage.UpdatePolicy(policy); err != nil {
		return err
	}
	return ds.indexer.AddPolicy(policy)
}

// RemovePolicy removes a policy from the storage and the indexer
func (ds *DataStore) RemovePolicy(id string) error {
	if err := ds.Storage.RemovePolicy(id); err != nil {
		return err
	}
	return ds.indexer.DeletePolicy(id)
}

// AddImage adds an image to the storage and the indexer
func (ds *DataStore) AddImage(image *v1.Image) error {
	if err := ds.Storage.AddImage(image); err != nil {
		return err
	}
	return ds.indexer.AddImage(image)
}

// UpdateImage updates an image in storage and the indexer
func (ds *DataStore) UpdateImage(image *v1.Image) error {
	if err := ds.Storage.UpdateImage(image); err != nil {
		return err
	}
	return ds.indexer.AddImage(image)
}

// RemoveImage removes an image from storage and the indexer
func (ds *DataStore) RemoveImage(id string) error {
	if err := ds.Storage.RemoveImage(id); err != nil {
		return err
	}
	return ds.indexer.DeleteImage(id)
}
