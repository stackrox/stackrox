package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/alert/mappings"
	baselineDatastore "github.com/stackrox/rox/central/processbaseline/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	pkgNotifier "github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/processbaseline"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/grpc"
)

var (
	alertSAC = sac.ForResource(resources.Alert)
)

const (
	badSnoozeErrorMsg = "'snooze_till' timestamp must be at a future time"

	maxListAlertsReturned = 1000
	alertResolveBatchSize = 100
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Alert)): {
			"/v1.AlertService/GetAlert",
			"/v1.AlertService/ListAlerts",
			"/v1.AlertService/CountAlerts",
			"/v1.AlertService/GetAlertsGroup",
			"/v1.AlertService/GetAlertsCounts",
			"/v1.AlertService/GetAlertTimeseries",
		},
		user.With(permissions.Modify(resources.Alert)): {
			"/v1.AlertService/ResolveAlert",
			"/v1.AlertService/SnoozeAlert",
			"/v1.AlertService/ResolveAlerts",
			"/v1.AlertService/DeleteAlerts",
		},
	})

	// groupByFunctions provides a map of functions that group slices of result objects by category or by cluster.
	groupByFunctions = map[v1.GetAlertsCountsRequest_RequestGroup]func(result search.Result) []string{
		v1.GetAlertsCountsRequest_UNSET: func(result search.Result) []string { return []string{""} },
		v1.GetAlertsCountsRequest_CATEGORY: func(a search.Result) (output []string) {
			field := mappings.OptionsMap.MustGet(search.Category.String())
			output = append(output, a.Matches[field.GetFieldPath()]...)
			return
		},
		v1.GetAlertsCountsRequest_CLUSTER: func(a search.Result) []string {
			field := mappings.OptionsMap.MustGet(search.Cluster.String())
			return []string{a.Matches[field.GetFieldPath()][0]}
		},
	}
)

// serviceImpl is a thin facade over a domain layer that handles CRUD use cases on Alert objects from API clients.
type serviceImpl struct {
	v1.UnimplementedAlertServiceServer

	dataStore         datastore.DataStore
	notifier          pkgNotifier.Processor
	baselines         baselineDatastore.DataStore
	connectionManager connection.Manager
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterAlertServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterAlertServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetAlert returns the alert with given id.
func (s *serviceImpl) GetAlert(ctx context.Context, request *v1.ResourceByID) (*storage.Alert, error) {
	alert, exists, err := s.dataStore.GetAlert(ctx, request.GetId())
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "alert with id '%s' does not exist", request.GetId())
	}

	return alert, nil
}

// listAlertsRequestToQuery converts a v1.ListAlertsRequest to a search query
func listAlertsRequestToQuery(request *v1.ListAlertsRequest, sort bool) (*v1.Query, error) {
	var q *v1.Query
	if request.GetQuery() == "" {
		q = search.EmptyQuery()
	} else {
		var err error
		q, err = search.ParseQuery(request.GetQuery())
		if err != nil {
			return nil, err
		}
	}

	paginated.FillPagination(q, request.GetPagination(), math.MaxInt32)
	if sort {
		q = paginated.FillDefaultSortOption(q, paginated.GetViolationTimeSortOption())
	}
	return q, nil
}

// ListAlerts returns ListAlerts according to the request.
func (s *serviceImpl) ListAlerts(ctx context.Context, request *v1.ListAlertsRequest) (*v1.ListAlertsResponse, error) {
	if request.GetPagination() == nil {
		request.Pagination = &v1.Pagination{
			Limit: maxListAlertsReturned,
		}
	}
	q, err := listAlertsRequestToQuery(request, true)
	if err != nil {
		return nil, err
	}
	alerts, err := s.dataStore.SearchListAlerts(ctx, q)
	if err != nil {
		return nil, err
	}
	return &v1.ListAlertsResponse{Alerts: alerts}, nil
}

// CountAlerts counts the number of alerts that match the input query.
func (s *serviceImpl) CountAlerts(ctx context.Context, request *v1.RawQuery) (*v1.CountAlertsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	count, err := s.dataStore.Count(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}
	return &v1.CountAlertsResponse{Count: int32(count)}, nil
}

func ensureAllAlertsAreFetched(req *v1.ListAlertsRequest) *v1.ListAlertsRequest {
	if req == nil {
		req = &v1.ListAlertsRequest{}
	}
	if req.GetPagination() == nil {
		req.Pagination = &v1.Pagination{}
	}
	req.Pagination.Offset = 0
	req.Pagination.Limit = math.MaxInt32
	return req
}

// GetAlertsGroup returns alerts according to the request, grouped by category and policy.
func (s *serviceImpl) GetAlertsGroup(ctx context.Context, request *v1.ListAlertsRequest) (*v1.GetAlertsGroupResponse, error) {
	request = ensureAllAlertsAreFetched(request)
	q, err := listAlertsRequestToQuery(request, false)
	if err != nil {
		return nil, err
	}
	alerts, err := s.dataStore.SearchListAlerts(ctx, q)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	response := alertsGroupResponseFrom(alerts)
	return response, nil
}

// GetAlertsCounts returns alert counts by severity according to the request.
// Counts can be grouped by policy category or cluster.
func (s *serviceImpl) GetAlertsCounts(ctx context.Context, request *v1.GetAlertsCountsRequest) (*v1.GetAlertsCountsResponse, error) {
	if request == nil {
		request = &v1.GetAlertsCountsRequest{}
	}

	request.Request = ensureAllAlertsAreFetched(request.GetRequest())
	requestQ, err := search.ParseQuery(request.GetRequest().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, err
	}

	var hasClusterQ, hasSeverityQ, hasCategoryQ bool
	search.ApplyFnToAllBaseQueries(requestQ, func(bq *v1.BaseQuery) {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}

		if matchFieldQuery.MatchFieldQuery.GetField() == search.Cluster.String() {
			hasClusterQ = true
			matchFieldQuery.MatchFieldQuery.Highlight = true
		}
		if matchFieldQuery.MatchFieldQuery.GetField() == search.Category.String() {
			hasCategoryQ = true
			matchFieldQuery.MatchFieldQuery.Highlight = true
		}
		if matchFieldQuery.MatchFieldQuery.GetField() == search.Severity.String() {
			hasSeverityQ = true
			matchFieldQuery.MatchFieldQuery.Highlight = true
		}
	})

	var conjuncts []*v1.Query
	if !hasClusterQ {
		conjuncts = append(conjuncts, search.NewQueryBuilder().AddStringsHighlighted(search.Cluster, search.WildcardString).ProtoQuery())
	}
	if !hasSeverityQ {
		conjuncts = append(conjuncts, search.NewQueryBuilder().AddStringsHighlighted(search.Severity, search.WildcardString).ProtoQuery())
	}
	if !hasCategoryQ {
		conjuncts = append(conjuncts, search.NewQueryBuilder().AddStringsHighlighted(search.Category, search.WildcardString).ProtoQuery())
	}
	for _, conjunct := range conjuncts {
		requestQ = search.ConjunctionQuery(requestQ, conjunct)
	}

	alerts, err := s.dataStore.Search(ctx, requestQ)
	if err != nil {
		return nil, err
	}

	if response, ok := alertsCountsResponseFrom(alerts, request.GetGroupBy()); ok {
		return response, nil
	}

	return nil, errors.Wrapf(errox.InvalidArgs, "unknown group by: %v", request.GetGroupBy())
}

// GetAlertTimeseries returns the timeseries format of the events based on the request parameters
func (s *serviceImpl) GetAlertTimeseries(ctx context.Context, req *v1.ListAlertsRequest) (*v1.GetAlertTimeseriesResponse, error) {
	ensureAllAlertsAreFetched(req)

	q, err := listAlertsRequestToQuery(req, false)
	if err != nil {
		return nil, err
	}

	alerts, err := s.dataStore.SearchListAlerts(ctx, q)
	if err != nil {
		return nil, err
	}
	response := alertTimeseriesResponseFrom(alerts)
	return response, nil
}

func (s *serviceImpl) ResolveAlert(ctx context.Context, req *v1.ResolveAlertRequest) (*v1.Empty, error) {
	alert, exists, err := s.dataStore.GetAlert(ctx, req.GetId())
	if err != nil {
		err = errors.Wrap(err, "could not change alert state to RESOLVED")
		log.Error(err)
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "alert with id '%s' does not exist", req.GetId())
	}

	if req.GetWhitelist() || req.GetAddToBaseline() {
		// This isn't great as it assumes a single baseline key
		itemMap := make(map[string][]*storage.BaselineItem)
		for _, process := range alert.GetProcessViolation().GetProcesses() {
			itemMap[process.GetContainerName()] = append(itemMap[process.GetContainerName()], &storage.BaselineItem{
				Item: &storage.BaselineItem_ProcessName{
					ProcessName: processbaseline.BaselineItemFromProcess(process),
				},
			})
		}
		for containerName, items := range itemMap {
			key := &storage.ProcessBaselineKey{
				DeploymentId:  alert.GetDeployment().GetId(),
				ContainerName: containerName,
				ClusterId:     alert.GetDeployment().GetClusterId(),
				Namespace:     alert.GetDeployment().GetNamespace(),
			}
			baseline, err := s.baselines.UpdateProcessBaselineElements(ctx, key, items, nil, false)
			if err != nil {
				return nil, err
			}
			err = s.connectionManager.SendMessage(alert.GetDeployment().GetClusterId(), &central.MsgToSensor{
				Msg: &central.MsgToSensor_BaselineSync{
					BaselineSync: &central.BaselineSync{
						Baselines: []*storage.ProcessBaseline{baseline},
					},
				},
			})
			if err != nil {
				log.Errorf("Error syncing baseline with cluster %q: %v", alert.GetDeployment().GetClusterId(), err)
			}
		}
	}

	if alert.State == storage.ViolationState_ATTEMPTED || alert.LifecycleStage == storage.LifecycleStage_RUNTIME {
		if err := s.changeAlertState(ctx, alert, storage.ViolationState_RESOLVED); err != nil {
			err = errors.Wrap(err, "could not change alert state to RESOLVED")
			log.Error(err)
			return nil, err
		}
	}

	return &v1.Empty{}, nil
}

func (s *serviceImpl) ResolveAlerts(ctx context.Context, req *v1.ResolveAlertsRequest) (*v1.Empty, error) {
	query, err := search.ParseQuery(req.GetQuery())
	if err != nil {
		log.Error(err)
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	runtimeQuery := search.NewQueryBuilder().AddExactMatches(search.LifecycleStage, storage.LifecycleStage_RUNTIME.String()).ProtoQuery()
	cq := search.ConjunctionQuery(query, runtimeQuery)
	alerts, err := s.dataStore.SearchRawAlerts(ctx, cq)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	err = s.changeAlertsState(ctx, alerts, storage.ViolationState_RESOLVED)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) checkAlertSAC(ctx context.Context, alert *storage.Alert, c chan error, waitGroup *sync.WaitGroup) {
	if ok, err := alertSAC.WriteAllowed(ctx, sac.KeyForNSScopedObj(alert.GetDeployment())...); err != nil || !ok {
		c <- errors.Wrapf(sac.ErrResourceAccessDenied, "alert id %q", alert.GetId())
	}
	waitGroup.Done()
}

func (s *serviceImpl) checkAlertsSAC(ctx context.Context, alerts []*storage.Alert) error {
	var waitGroup sync.WaitGroup
	c := make(chan error, len(alerts))
	for _, alert := range alerts {
		waitGroup.Add(1)
		go s.checkAlertSAC(ctx, alert, c, &waitGroup)
	}
	waitGroup.Wait()
	if len(c) > 0 {
		errorList := errorhelpers.NewErrorList(fmt.Sprintf("found %d sac permission denials while resolving alerts", len(c)))
		for err := range c {
			errorList.AddError(err)
		}
		return errorList.ToError()
	}
	return nil
}

func (s *serviceImpl) changeAlertsState(ctx context.Context, alerts []*storage.Alert, state storage.ViolationState) error {
	err := s.checkAlertsSAC(ctx, alerts)
	if err != nil {
		return err
	}

	b := batcher.New(len(alerts), alertResolveBatchSize)
	for start, end, valid := b.Next(); valid; start, end, valid = b.Next() {
		for _, alert := range alerts[start:end] {
			if state != storage.ViolationState_SNOOZED {
				alert.SnoozeTill = nil
			}
			alert.State = state
		}
		err := s.dataStore.UpsertAlerts(ctx, alerts[start:end])
		if err != nil {
			log.Error(err)
			return err
		}
		for _, alert := range alerts[start:end] {
			s.notifier.ProcessAlert(ctx, alert)
		}
	}
	return nil
}

func (s *serviceImpl) changeAlertState(ctx context.Context, alert *storage.Alert, state storage.ViolationState) error {
	if state != storage.ViolationState_SNOOZED {
		alert.SnoozeTill = nil
	}
	alert.State = state
	err := s.dataStore.UpsertAlert(ctx, alert)
	if err != nil {
		log.Error(err)
		return err
	}
	s.notifier.ProcessAlert(ctx, alert)
	return nil
}

func (s *serviceImpl) SnoozeAlert(ctx context.Context, req *v1.SnoozeAlertRequest) (*v1.Empty, error) {
	if req.GetSnoozeTill() == nil {
		return nil, errors.Wrap(errox.InvalidArgs, "'snooze_till' cannot be nil")
	}
	if protoconv.ConvertTimestampToTimeOrNow(req.GetSnoozeTill()).Before(time.Now()) {
		return nil, errors.Wrap(errox.InvalidArgs, badSnoozeErrorMsg)
	}
	alert, exists, err := s.dataStore.GetAlert(ctx, req.GetId())
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "alert with id '%s' does not exist", req.GetId())
	}
	alert.SnoozeTill = req.GetSnoozeTill()
	err = s.changeAlertState(ctx, alert, storage.ViolationState_SNOOZED)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return &v1.Empty{}, nil
}

// DeleteAlerts is a maintenance function that deletes alerts from the store
func (s *serviceImpl) DeleteAlerts(ctx context.Context, request *v1.DeleteAlertsRequest) (*v1.DeleteAlertsResponse, error) {
	if request.GetQuery() == nil {
		return nil, errors.New("a scoping query is required")
	}

	query, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "error parsing query: %v", err)
	}
	paginated.FillPagination(query, request.GetQuery().GetPagination(), math.MaxInt32)

	specified := false
	search.ApplyFnToAllBaseQueries(query, func(bq *v1.BaseQuery) {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}
		if matchFieldQuery.MatchFieldQuery.GetField() == search.ViolationState.String() {
			if matchFieldQuery.MatchFieldQuery.Value != storage.ViolationState_RESOLVED.String() {
				err = errors.Wrapf(errox.InvalidArgs, "invalid value for violation state: %q. Only resolved alerts can be deleted", matchFieldQuery.MatchFieldQuery.Value)
				return
			}
			specified = true
		}
	})
	if err != nil {
		return nil, err
	}
	if !specified {
		return nil, errors.Wrapf(errox.InvalidArgs, "please specify Violation State:%s in the query to confirm deletion", storage.ViolationState_RESOLVED.String())
	}

	results, err := s.dataStore.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	response := &v1.DeleteAlertsResponse{
		NumDeleted: uint32(len(results)),
		DryRun:     !request.GetConfirm(),
	}

	if !request.GetConfirm() {
		return response, nil
	}

	idSlice := search.ResultsToIDs(results)
	if err := s.dataStore.DeleteAlerts(ctx, idSlice...); err != nil {
		return nil, err
	}
	return response, nil
}

// alertsGroupResponseFrom returns a slice of storage.ListAlert objects translated into a v1.GetAlertsGroupResponse object.
func alertsGroupResponseFrom(alerts []*storage.ListAlert) (output *v1.GetAlertsGroupResponse) {
	policiesMap := make(map[string]*storage.ListAlertPolicy)
	alertCountsByPolicy := make(map[string]int)

	for _, a := range alerts {
		if _, ok := policiesMap[a.GetPolicy().GetId()]; !ok {
			policiesMap[a.GetPolicy().GetId()] = a.GetPolicy()
		}
		alertCountsByPolicy[a.GetPolicy().GetId()]++
	}

	output = new(v1.GetAlertsGroupResponse)
	output.AlertsByPolicies = make([]*v1.GetAlertsGroupResponse_PolicyGroup, 0, len(policiesMap))

	for id, p := range policiesMap {
		output.AlertsByPolicies = append(output.AlertsByPolicies, &v1.GetAlertsGroupResponse_PolicyGroup{
			Policy:    p,
			NumAlerts: int64(alertCountsByPolicy[id]),
		})
	}

	sort.Slice(output.AlertsByPolicies, func(i, j int) bool {
		return output.AlertsByPolicies[i].GetPolicy().GetName() < output.AlertsByPolicies[j].GetPolicy().GetName()
	})

	return
}

// alertsCountsResponseFrom returns a slice of search.Result objects translated into a v1.GetAlertsCountsResponse
// object. True is returned if the translation was successful; otherwise false when the requested group is unknown.
func alertsCountsResponseFrom(alerts []search.Result, groupBy v1.GetAlertsCountsRequest_RequestGroup) (*v1.GetAlertsCountsResponse, bool) {
	if groupByFunc, ok := groupByFunctions[groupBy]; ok {
		response := countAlerts(alerts, groupByFunc)
		return response, true
	}

	return nil, false
}

// alertTimeseriesResponseFrom returns a slice of storage.ListAlert objects translated into a v1.GetAlertTimeseriesResponse
// object.
func alertTimeseriesResponseFrom(alerts []*storage.ListAlert) *v1.GetAlertTimeseriesResponse {
	response := new(v1.GetAlertTimeseriesResponse)
	for cluster, severityMap := range getGroupToAlertEvents(alerts) {
		alertCluster := &v1.GetAlertTimeseriesResponse_ClusterAlerts{Cluster: cluster}
		for severity, alertEvents := range severityMap {
			// Sort the alert events so they are chronological
			sort.SliceStable(alertEvents, func(i, j int) bool { return alertEvents[i].GetTime() < alertEvents[j].GetTime() })
			alertCluster.Severities = append(alertCluster.Severities, &v1.GetAlertTimeseriesResponse_ClusterAlerts_AlertEvents{
				Severity: severity,
				Events:   alertEvents,
			})
		}
		sort.Slice(alertCluster.Severities, func(i, j int) bool { return alertCluster.Severities[i].Severity < alertCluster.Severities[j].Severity })
		response.Clusters = append(response.Clusters, alertCluster)
	}
	sort.SliceStable(response.Clusters, func(i, j int) bool { return response.Clusters[i].Cluster < response.Clusters[j].Cluster })
	return response
}

func countAlerts(alerts []search.Result, groupByFunc func(result search.Result) []string) (output *v1.GetAlertsCountsResponse) {
	groups := getMapOfAlertCounts(alerts, groupByFunc)

	output = new(v1.GetAlertsCountsResponse)
	output.Groups = make([]*v1.GetAlertsCountsResponse_AlertGroup, 0, len(groups))

	for group, countsBySeverity := range groups {
		bySeverity := make([]*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts, 0, len(countsBySeverity))

		for severity, count := range countsBySeverity {
			bySeverity = append(bySeverity, &v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
				Severity: severity,
				Count:    int64(count),
			})
		}

		sort.Slice(bySeverity, func(i, j int) bool {
			return bySeverity[i].Severity < bySeverity[j].Severity
		})

		output.Groups = append(output.Groups, &v1.GetAlertsCountsResponse_AlertGroup{
			Group:  group,
			Counts: bySeverity,
		})
	}

	sort.Slice(output.Groups, func(i, j int) bool {
		return output.Groups[i].Group < output.Groups[j].Group
	})

	return
}

func getMapOfAlertCounts(alerts []search.Result, groupByFunc func(alert search.Result) []string) (groups map[string]map[storage.Severity]int) {
	groups = make(map[string]map[storage.Severity]int)
	field := mappings.OptionsMap.MustGet(search.Severity.String())

	for _, a := range alerts {
		for _, g := range groupByFunc(a) {
			if groups[g] == nil {
				groups[g] = make(map[storage.Severity]int)
			}
			if len(a.Matches[field.GetFieldPath()]) == 0 {
				continue
			}
			// There is a difference in how enum matches are stored in postgres vs rockdb. In postgres they are
			// stored as string values, in rocksdb as int values. Courtesy: Mandar.
			severity := storage.Severity_value[a.Matches[field.GetFieldPath()][0]]
			groups[g][(storage.Severity(severity))]++
		}
	}
	return
}

func getGroupToAlertEvents(alerts []*storage.ListAlert) (clusters map[string]map[storage.Severity][]*v1.AlertEvent) {
	clusters = make(map[string]map[storage.Severity][]*v1.AlertEvent)
	for _, a := range alerts {
		alertCluster := a.GetDeployment().GetClusterName()
		if clusters[alertCluster] == nil {
			clusters[alertCluster] = make(map[storage.Severity][]*v1.AlertEvent)
		}
		eventList := clusters[alertCluster][a.GetPolicy().GetSeverity()]
		eventList = append(eventList, &v1.AlertEvent{Time: a.GetTime().GetSeconds() * 1000, Id: a.GetId(), Type: v1.Type_CREATED})
		if a.GetState() == storage.ViolationState_RESOLVED {
			eventList = append(eventList, &v1.AlertEvent{Time: a.GetTime().GetSeconds() * 1000, Id: a.GetId(), Type: v1.Type_REMOVED})
		}
		clusters[alertCluster][a.GetPolicy().GetSeverity()] = eventList
	}

	for _, v1 := range clusters {
		for k2, v2 := range v1 {
			sort.SliceStable(v2, func(i, j int) bool { return v2[i].GetTime() < v2[j].GetTime() })
			v1[k2] = v2
		}
	}
	return
}
