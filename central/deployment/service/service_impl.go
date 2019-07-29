package service

import (
	"context"
	"math"
	"sort"

	"github.com/gogo/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/deployment/mappings"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	processWhitelistStore "github.com/stackrox/rox/central/processwhitelist/datastore"
	processWhitelistResultsStore "github.com/stackrox/rox/central/processwhitelistresults/datastore"
	"github.com/stackrox/rox/central/ranking"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxDeploymentsReturned = 1000
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Deployment)): {
			"/v1.DeploymentService/GetDeployment",
			"/v1.DeploymentService/CountDeployments",
			"/v1.DeploymentService/ListDeployments",
			"/v1.DeploymentService/GetLabels",
		},
		user.With(permissions.View(resources.Deployment), permissions.View(resources.ProcessWhitelist), permissions.View(resources.Indicator)): {
			"/v1.DeploymentService/ListDeploymentsWithProcessInfo",
		},
	})
)

// serviceImpl provides APIs for deployments.
type serviceImpl struct {
	datastore               datastore.DataStore
	processWhitelists       processWhitelistStore.DataStore
	processIndicators       processIndicatorStore.DataStore
	processWhitelistResults processWhitelistResultsStore.DataStore
	risks                   riskDataStore.DataStore
	manager                 manager.Manager
}

func (s *serviceImpl) whitelistResultsForDeployment(ctx context.Context, deployment *storage.ListDeployment) (*storage.ProcessWhitelistResults, error) {
	whitelistResults, err := s.processWhitelistResults.GetWhitelistResults(ctx, deployment.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return whitelistResults, nil
}

func (s *serviceImpl) ListDeploymentsWithProcessInfo(ctx context.Context, rawQuery *v1.RawQuery) (*v1.ListDeploymentsWithProcessInfoResponse, error) {
	deployments, err := s.ListDeployments(ctx, rawQuery)
	if err != nil {
		return nil, err
	}

	resp := &v1.ListDeploymentsWithProcessInfoResponse{}
	for _, deployment := range deployments.Deployments {
		whitelistResults, err := s.whitelistResultsForDeployment(ctx, deployment)
		if err != nil {
			return nil, err
		}

		var whitelistStatuses []*storage.ContainerNameAndWhitelistStatus
		if whitelistResults != nil {
			whitelistStatuses = whitelistResults.WhitelistStatuses
		}
		resp.Deployments = append(resp.Deployments, &v1.ListDeploymentsWithProcessInfoResponse_DeploymentWithProcessInfo{
			Deployment:        deployment,
			WhitelistStatuses: whitelistStatuses,
		})
	}
	return resp, nil
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterDeploymentServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterDeploymentServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetDeployment returns the deployment with given id.
func (s *serviceImpl) GetDeployment(ctx context.Context, request *v1.ResourceByID) (*storage.Deployment, error) {
	deployment, exists, err := s.datastore.GetDeployment(ctx, request.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "deployment with id '%s' does not exist", request.GetId())
	}
	return deployment, nil
}

// CountDeployments counts the number of deployments that match the input query.
func (s *serviceImpl) CountDeployments(ctx context.Context, request *v1.RawQuery) (*v1.CountDeploymentsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseRawQueryOrEmpty(request.GetQuery())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	deployments, err := s.datastore.Search(ctx, s.getQuery(ctx, parsedQuery))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.CountDeploymentsResponse{Count: int32(len(deployments))}, nil
}

// ListDeployments returns ListDeployments according to the request.
func (s *serviceImpl) ListDeployments(ctx context.Context, request *v1.RawQuery) (*v1.ListDeploymentsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseRawQueryOrEmpty(request.GetQuery())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Fill in pagination.
	paginated.FillPagination(parsedQuery, request.Pagination, maxDeploymentsReturned)

	deployments, err := s.datastore.SearchListDeployments(ctx, s.getQuery(ctx, parsedQuery))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.ListDeploymentsResponse{
		Deployments: deployments,
	}, nil
}

func queryForLabels() *v1.Query {
	q := search.NewQueryBuilder().AddStringsHighlighted(search.Label, search.WildcardString).ProtoQuery()
	q.Pagination = &v1.Pagination{
		Limit: math.MaxInt32,
	}
	return q
}

// GetLabels returns label keys and values for current deployments.
func (s *serviceImpl) GetLabels(ctx context.Context, _ *v1.Empty) (*v1.DeploymentLabelsResponse, error) {
	q := queryForLabels()
	searchRes, err := s.datastore.Search(ctx, q)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	labelsMap, values := labelsMapFromSearchResults(searchRes)

	return &v1.DeploymentLabelsResponse{
		Labels: labelsMap,
		Values: values,
	}, nil
}

func labelsMapFromSearchResults(results []search.Result) (keyValuesMap map[string]*v1.DeploymentLabelsResponse_LabelValues, values []string) {
	labelFieldPath := mappings.OptionsMap.MustGet(search.Label.String()).GetFieldPath()
	keyFieldPath := blevesearch.ToMapKeyPath(labelFieldPath)
	valueFieldPath := blevesearch.ToMapValuePath(labelFieldPath)

	tempSet := make(map[string]set.StringSet)
	globalValueSet := set.NewStringSet()

	for _, r := range results {
		keyMatches, valueMatches := r.Matches[keyFieldPath], r.Matches[valueFieldPath]
		if len(keyMatches) != len(valueMatches) {
			errorhelpers.PanicOnDevelopmentf("Mismatch between key and value matches: %d != %d", len(keyMatches), len(valueMatches))
			continue
		}
		for i, keyMatch := range keyMatches {
			valueMatch := valueMatches[i]
			valSet, ok := tempSet[keyMatch]
			if !ok {
				valSet = set.NewStringSet()
				tempSet[keyMatch] = valSet
			}
			valSet.Add(valueMatch)
			globalValueSet.Add(valueMatch)
		}
	}

	keyValuesMap = make(map[string]*v1.DeploymentLabelsResponse_LabelValues)
	for k, valSet := range tempSet {
		keyValuesMap[k] = &v1.DeploymentLabelsResponse_LabelValues{
			Values: make([]string, 0, valSet.Cardinality()),
		}

		keyValuesMap[k].Values = append(keyValuesMap[k].Values, valSet.AsSlice()...)
		sort.Strings(keyValuesMap[k].Values)
	}
	values = globalValueSet.AsSlice()
	sort.Strings(values)

	return
}

func (s *serviceImpl) getQuery(ctx context.Context, q *v1.Query) *v1.Query {
	deploymentQuery := s.createDeploymentQuery(s.createDeploymentSortQuery(q))
	riskQuery := s.createRiskQuery(s.createRiskSortQuery(q))
	convertedDeploymentQuery := s.convertToDeploymentQuery(ctx, deploymentQuery, riskQuery)

	if convertedDeploymentQuery == nil {
		return search.EmptyQuery()
	}
	return convertedDeploymentQuery
}

func (s *serviceImpl) convertToDeploymentQuery(ctx context.Context, deploymentQuery *v1.Query, riskQuery *v1.Query) *v1.Query {
	if riskQuery == nil && deploymentQuery == nil {
		return nil
	}
	if riskQuery == nil {
		return deploymentQuery
	}

	var docIDs []string
	deploymentTypeQuery := search.NewQueryBuilder().AddStrings(
		search.RiskSubjectType, storage.RiskSubjectType_DEPLOYMENT.String()).ProtoQuery()
	risks, err := s.risks.SearchRawRisks(ctx, search.NewConjunctionQuery(riskQuery, deploymentTypeQuery))
	if err != nil {
		return nil
	}
	for _, risk := range risks {
		docIDs = append(docIDs, risk.GetSubject().GetId())
	}

	var docIDQuery *v1.Query
	if len(docIDs) != 0 {
		docIDQuery = search.NewQueryBuilder().AddDocIDs(docIDs...).ProtoQuery()
	}

	if docIDQuery == nil && deploymentQuery == nil {
		return nil
	}
	if deploymentQuery == nil {
		return docIDQuery
	}
	if docIDQuery == nil {
		return deploymentQuery
	}
	return search.NewConjunctionQuery(deploymentQuery, docIDQuery)
}

func (s *serviceImpl) createDeploymentSortQuery(q *v1.Query) *v1.Query {
	cloneQuery := proto.Clone(q).(*v1.Query)

	if q.GetPagination().GetSortOption().GetField() == search.Priority.String() {
		cloneQuery.Pagination.SortOption.Field = ""
	}
	return cloneQuery
}

func (s *serviceImpl) createDeploymentQuery(q *v1.Query) *v1.Query {
	newQuery, _ := search.FilterQuery(q, func(bq *v1.BaseQuery) bool {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return false
		}
		return matchFieldQuery.MatchFieldQuery.GetField() != search.Priority.String()
	})

	return newQuery
}

func (s *serviceImpl) createRiskSortQuery(q *v1.Query) *v1.Query {
	cloneQuery := proto.Clone(q).(*v1.Query)
	if q.GetPagination().GetSortOption().GetField() == search.Priority.String() {
		paginated.FillPagination(cloneQuery, q.GetPagination(), q.GetPagination().GetLimit())
		cloneQuery.Pagination.SortOption.Field = search.RiskScore.String()
		cloneQuery.Pagination.SortOption.Reversed = !q.GetPagination().GetSortOption().GetReversed()
	}
	return cloneQuery
}

func (s *serviceImpl) createRiskQuery(q *v1.Query) *v1.Query {
	var err error
	ranker := ranking.DeploymentRanker()

	newQuery, _ := search.FilterQuery(q, func(bq *v1.BaseQuery) bool {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return false
		}
		return matchFieldQuery.MatchFieldQuery.GetField() == search.Priority.String()
	})
	if newQuery == nil {
		return nil
	}

	search.ApplyFnToAllBaseQueries(newQuery, func(bq *v1.BaseQuery) {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}

		// Any Priority query, we want to change to be a risk query.
		// Parse the numeric query so we can swap it to a risk score query.
		var numericValue blevesearch.NumericQueryValue
		numericValue, err = blevesearch.ParseNumericQueryValue(matchFieldQuery.MatchFieldQuery.GetValue())
		if err == nil {
			return
		}

		// Go from priority space to risk score space by inverting comparison.
		numericValue.Comparator = priorityComparatorToRiskScoreComparator(numericValue.Comparator)
		numericValue.Value = float64(ranker.GetScoreForRank(int64(numericValue.Value)))

		// Set the query to the new value.
		matchFieldQuery.MatchFieldQuery.Field = search.RiskScore.String()
		matchFieldQuery.MatchFieldQuery.Value = blevesearch.PrintNumericQueryValue(numericValue)

	})
	if err != nil {
		log.Error(err)
		return nil
	}
	return newQuery
}

func priorityComparatorToRiskScoreComparator(comparator storage.Comparator) storage.Comparator {
	switch comparator {
	case storage.Comparator_LESS_THAN_OR_EQUALS:
		return storage.Comparator_GREATER_THAN
	case storage.Comparator_LESS_THAN:
		return storage.Comparator_GREATER_THAN_OR_EQUALS
	case storage.Comparator_GREATER_THAN_OR_EQUALS:
		return storage.Comparator_LESS_THAN
	case storage.Comparator_GREATER_THAN:
		return storage.Comparator_LESS_THAN_OR_EQUALS
	default: // storage.Comparator_EQUALS:
		return storage.Comparator_EQUALS
	}
}
