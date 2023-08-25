package service

import (
	"context"
	"math"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/deployment/datastore"
	processBaselineStore "github.com/stackrox/rox/central/processbaseline/datastore"
	processBaselineResultsStore "github.com/stackrox/rox/central/processbaselineresults/datastore"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/options/deployments"
	"github.com/stackrox/rox/pkg/search/paginated"
	pgsearch "github.com/stackrox/rox/pkg/search/postgres/query"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
)

const (
	maxDeploymentsReturned = 1000
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Deployment)): {
			"/v1.DeploymentService/GetDeployment",
			"/v1.DeploymentService/GetDeploymentWithRisk",
			"/v1.DeploymentService/CountDeployments",
			"/v1.DeploymentService/ListDeployments",
			"/v1.DeploymentService/GetLabels",
			"/v1.DeploymentService/ListDeploymentsWithProcessInfo",
		},
	})
	deploymentExtensionAuth = user.With(permissions.View(resources.DeploymentExtension))
)

// serviceImpl provides APIs for deployments.
type serviceImpl struct {
	v1.UnimplementedDeploymentServiceServer

	datastore              datastore.DataStore
	processBaselines       processBaselineStore.DataStore
	processIndicators      processIndicatorStore.DataStore
	processBaselineResults processBaselineResultsStore.DataStore
	risks                  riskDataStore.DataStore
	manager                manager.Manager
}

func (s *serviceImpl) baselineResultsForDeployment(ctx context.Context, deployment *storage.ListDeployment) (*storage.ProcessBaselineResults, error) {
	baselineResults, err := s.processBaselineResults.GetBaselineResults(ctx, deployment.GetId())
	if err != nil {
		return nil, err
	}
	return baselineResults, nil
}

func (s *serviceImpl) fillBaselineResults(ctx context.Context, resp *v1.ListDeploymentsWithProcessInfoResponse) error {
	if err := deploymentExtensionAuth.Authorized(ctx, ""); err == nil {
		for _, depWithProc := range resp.Deployments {
			baselineResults, err := s.baselineResultsForDeployment(ctx, depWithProc.GetDeployment())
			if err != nil {
				return err
			}
			depWithProc.BaselineStatuses = baselineResults.GetBaselineStatuses()
		}
	}
	return nil
}

func (s *serviceImpl) ListDeploymentsWithProcessInfo(ctx context.Context, rawQuery *v1.RawQuery) (*v1.ListDeploymentsWithProcessInfoResponse, error) {
	deployments, err := s.ListDeployments(ctx, rawQuery)
	if err != nil {
		return nil, err
	}

	resp := &v1.ListDeploymentsWithProcessInfoResponse{}
	for _, deployment := range deployments.Deployments {
		resp.Deployments = append(resp.Deployments,
			&v1.ListDeploymentsWithProcessInfoResponse_DeploymentWithProcessInfo{
				Deployment: deployment,
			},
		)
	}
	if err := s.fillBaselineResults(ctx, resp); err != nil {
		return nil, err
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
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "deployment with id '%s' does not exist", request.GetId())
	}
	return deployment, nil
}

// GetDeploymentWithRisk returns the deployment and its risk with given id.
func (s *serviceImpl) GetDeploymentWithRisk(ctx context.Context, request *v1.ResourceByID) (*v1.GetDeploymentWithRiskResponse, error) {
	deployment, exists, err := s.datastore.GetDeployment(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "deployment with id '%s' does not exist", request.GetId())
	}

	risk, _, err := s.risks.GetRiskForDeployment(ctx, deployment)
	if err != nil {
		return nil, err
	}

	return &v1.GetDeploymentWithRiskResponse{
		Deployment: deployment,
		Risk:       risk,
	}, nil
}

// CountDeployments counts the number of deployments that match the input query.
func (s *serviceImpl) CountDeployments(ctx context.Context, request *v1.RawQuery) (*v1.CountDeploymentsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	numDeployments, err := s.datastore.Count(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}
	return &v1.CountDeploymentsResponse{Count: int32(numDeployments)}, nil
}

// ListDeployments returns ListDeployments according to the request.
func (s *serviceImpl) ListDeployments(ctx context.Context, request *v1.RawQuery) (*v1.ListDeploymentsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	// Fill in pagination.
	paginated.FillPagination(parsedQuery, request.GetPagination(), maxDeploymentsReturned)

	deployments, err := s.datastore.SearchListDeployments(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}

	return &v1.ListDeploymentsResponse{
		Deployments: deployments,
	}, nil
}

func queryForLabels() *v1.Query {
	q := search.NewQueryBuilder().AddStringsHighlighted(search.DeploymentLabel, search.WildcardString).ProtoQuery()
	q.Pagination = &v1.QueryPagination{
		Limit: math.MaxInt32,
	}
	return q
}

// GetLabels returns label keys and values for current deployments.
func (s *serviceImpl) GetLabels(ctx context.Context, _ *v1.Empty) (*v1.DeploymentLabelsResponse, error) {
	q := queryForLabels()
	searchRes, err := s.datastore.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	labelsMap, values := labelsMapFromSearchResults(searchRes)

	return &v1.DeploymentLabelsResponse{
		Labels: labelsMap,
		Values: values,
	}, nil
}

func labelsMapFromSearchResults(results []search.Result) (map[string]*v1.DeploymentLabelsResponse_LabelValues, []string) {
	labelField, ok := deployments.OptionsMap.Get(search.DeploymentLabel.String())
	if !ok {
		utils.Should(errors.Errorf("could not find label %q in options map", search.DeploymentLabel.String()))
		return nil, nil
	}
	labelFieldPath := labelField.GetFieldPath()
	tempSet := make(map[string]set.StringSet)
	globalValueSet := set.NewStringSet()

	setUpdater := func(key, value string) {
		valSet, ok := tempSet[key]
		if !ok {
			valSet = set.NewStringSet()
			tempSet[key] = valSet
		}
		valSet.Add(value)
		globalValueSet.Add(value)
	}

	for _, r := range results {
		// In postgres, map key and values are returned as one `k=v`.
		for _, match := range r.Matches[labelFieldPath] {
			key, value, hasEquals := pgsearch.ParseMapQuery(match)
			if !hasEquals {
				utils.Should(errors.Errorf("cannot handle label %s", match))
				continue
			}
			setUpdater(key, value)
		}
	}

	keyValuesMap := make(map[string]*v1.DeploymentLabelsResponse_LabelValues, len(tempSet))
	var values []string
	for k, valSet := range tempSet {
		keyValuesMap[k] = &v1.DeploymentLabelsResponse_LabelValues{
			Values: valSet.AsSlice(),
		}
		sort.Strings(keyValuesMap[k].Values)
	}
	values = globalValueSet.AsSlice()
	sort.Strings(values)

	return keyValuesMap, values
}
