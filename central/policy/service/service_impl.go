package service

import (
	"fmt"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/deployment/index/mappings"
	imageDetection "github.com/stackrox/rox/central/detection/image"
	"github.com/stackrox/rox/central/detection/lifecycle"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/central/policy/datastore"
	processDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/searchbasedpolicies/matcher"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/compiledpolicies/deployment/predicate"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log = logging.LoggerForModule()

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Policy)): {
			"/v1.PolicyService/GetPolicy",
			"/v1.PolicyService/ListPolicies",
			"/v1.PolicyService/ReassessPolicies",
			"/v1.PolicyService/GetPolicyCategories",
		},
		user.With(permissions.Modify(resources.Policy)): {
			"/v1.PolicyService/PostPolicy",
			"/v1.PolicyService/PutPolicy",
			"/v1.PolicyService/PatchPolicy",
			"/v1.PolicyService/DeletePolicy",
			"/v1.PolicyService/DryRunPolicy",
			"/v1.PolicyService/RenamePolicyCategory",
			"/v1.PolicyService/DeletePolicyCategory",
		},
	})
)

const (
	uncategorizedCategory = `Uncategorized`
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	policies    datastore.DataStore
	clusters    clusterDataStore.DataStore
	deployments deploymentDataStore.DataStore
	processes   processDataStore.DataStore
	reprocessor reprocessor.Loop

	buildTimePolicies imageDetection.PolicySet
	lifecycleManager  lifecycle.Manager
	processor         notifierProcessor.Processor
	metadataCache     expiringcache.Cache
	scanCache         expiringcache.Cache

	validator *policyValidator
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterPolicyServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterPolicyServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetPolicy returns a policy by name.
func (s *serviceImpl) GetPolicy(ctx context.Context, request *v1.ResourceByID) (*storage.Policy, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Policy id must be provided")
	}
	policy, exists, err := s.policies.GetPolicy(request.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "policy with id '%s' does not exist", request.GetId())
	}
	if len(policy.GetCategories()) == 0 {
		policy.Categories = []string{uncategorizedCategory}
	}
	return policy, nil
}

func convertPoliciesToListPolicies(policies []*storage.Policy) []*storage.ListPolicy {
	listPolicies := make([]*storage.ListPolicy, 0, len(policies))
	for _, p := range policies {
		listPolicies = append(listPolicies, &storage.ListPolicy{
			Id:              p.GetId(),
			Name:            p.GetName(),
			Description:     p.GetDescription(),
			Severity:        p.GetSeverity(),
			Disabled:        p.GetDisabled(),
			LifecycleStages: p.GetLifecycleStages(),
		})
	}
	return listPolicies
}

// ListPolicies retrieves all policies in ListPolicy form according to the request.
func (s *serviceImpl) ListPolicies(ctx context.Context, request *v1.RawQuery) (*v1.ListPoliciesResponse, error) {
	resp := new(v1.ListPoliciesResponse)
	if request.GetQuery() == "" {
		policies, err := s.policies.GetPolicies()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		resp.Policies = convertPoliciesToListPolicies(policies)
	} else {
		parsedQuery, err := search.ParseRawQuery(request.GetQuery())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		policies, err := s.policies.SearchRawPolicies(parsedQuery)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		resp.Policies = convertPoliciesToListPolicies(policies)
	}
	sort.SliceStable(resp.Policies, func(i, j int) bool { return resp.Policies[i].GetName() < resp.Policies[j].GetName() })
	return resp, nil
}

// PostPolicy inserts a new policy into the system.
func (s *serviceImpl) PostPolicy(ctx context.Context, request *storage.Policy) (*storage.Policy, error) {
	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Id field should be empty when posting a new policy")
	}
	if err := s.validator.validate(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	id, err := s.policies.AddPolicy(request)
	if err != nil {
		return nil, err
	}
	request.Id = id

	if err := s.addActivePolicy(request); err != nil {
		return nil, errors.Wrap(err, "Policy could not be edited due to")
	}
	return request, nil
}

// PutPolicy updates a current policy in the system.
func (s *serviceImpl) PutPolicy(ctx context.Context, request *storage.Policy) (*v1.Empty, error) {
	if err := s.validator.validate(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := s.policies.UpdatePolicy(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := s.addActivePolicy(request); err != nil {
		return nil, errors.Wrap(err, "Policy could not be edited due to")
	}
	return &v1.Empty{}, nil
}

// PatchPolicy patches a current policy in the system.
func (s *serviceImpl) PatchPolicy(ctx context.Context, request *v1.PatchPolicyRequest) (*v1.Empty, error) {
	policy, exists, err := s.policies.GetPolicy(request.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Policy with id '%s' not found", request.GetId()))
	}
	if request.SetDisabled != nil {
		policy.Disabled = request.GetDisabled()
	}
	return s.PutPolicy(ctx, policy)
}

// DeletePolicy deletes an policy from the system.
func (s *serviceImpl) DeletePolicy(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "A policy id must be specified to delete a Policy")
	}

	policy, exists, err := s.policies.GetPolicy(request.GetId())
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Policy with id '%s' not found", request.GetId()))
	}

	if err := s.removeActivePolicy(policy); err != nil {
		return nil, err
	}
	if err := s.policies.RemovePolicy(request.GetId()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

// ReassessPolicies manually triggers enrichment of all deployments, and re-assesses policies if there's updated data.
func (s *serviceImpl) ReassessPolicies(context.Context, *v1.Empty) (*v1.Empty, error) {
	// Invalidate scan and metadata caches
	s.metadataCache.Purge()
	s.scanCache.Purge()

	go s.reprocessor.ShortCircuit()
	return &v1.Empty{}, nil
}

// DryRunPolicy runs a dry run of the policy and determines what deployments would violate it
func (s *serviceImpl) DryRunPolicy(ctx context.Context, request *storage.Policy) (*v1.DryRunResponse, error) {
	if err := s.validator.validate(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	shouldProcessFunc, err := predicate.Compile(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("policy predicate does not compile: %s", err))
	}

	var resp v1.DryRunResponse
	searchBasedMatcher, err := matcher.ForPolicy(request, mappings.OptionsMap, s.processes)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("couldn't construct matcher: %s", err))
	}

	violationsPerDeployment, err := searchBasedMatcher.Match(s.deployments)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed policy matching: %s", err))
	}

	for deploymentID, violations := range violationsPerDeployment {
		if len(violations.AlertViolations) == 0 && violations.ProcessViolation == nil {
			continue
		}
		deployment, exists, err := s.deployments.GetDeployment(deploymentID)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("retrieving deployment '%s': %s", deploymentID, err))
		}
		if !exists {
			// Maybe the deployment was deleted around the time of the dry-run.
			continue
		}
		if shouldProcessFunc != nil && !shouldProcessFunc(deployment) {
			continue
		}
		// Collect the violation messages as strings for the output.
		convertedViolations := make([]string, 0, len(violations.AlertViolations))
		for _, violation := range violations.AlertViolations {
			convertedViolations = append(convertedViolations, violation.GetMessage())
		}
		if violations.ProcessViolation != nil {
			convertedViolations = append(convertedViolations, violations.ProcessViolation.GetMessage())
		}
		resp.Alerts = append(resp.Alerts, &v1.DryRunResponse_Alert{Deployment: deployment.GetName(), Violations: convertedViolations})
	}
	return &resp, nil
}

// GetPolicyCategories returns the categories of all policies.
func (s *serviceImpl) GetPolicyCategories(context.Context, *v1.Empty) (*v1.PolicyCategoriesResponse, error) {
	categorySet, err := s.getPolicyCategorySet()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	response := new(v1.PolicyCategoriesResponse)
	response.Categories = categorySet.AsSlice()
	sort.Strings(response.Categories)

	return response, nil
}

// RenamePolicyCategory changes all usage of the category in policies to the requsted name.
func (s *serviceImpl) RenamePolicyCategory(ctx context.Context, request *v1.RenamePolicyCategoryRequest) (*v1.Empty, error) {
	if request.GetOldCategory() == request.GetNewCategory() {
		return &v1.Empty{}, nil
	}

	if err := s.policies.RenamePolicyCategory(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.Empty{}, nil
}

// DeletePolicyCategory removes all usage of the category in policies. Policies may end up with no configured category.
func (s *serviceImpl) DeletePolicyCategory(ctx context.Context, request *v1.DeletePolicyCategoryRequest) (*v1.Empty, error) {
	categorySet, err := s.getPolicyCategorySet()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !categorySet.Contains(request.GetCategory()) {
		return nil, status.Errorf(codes.NotFound, "Policy Category %s does not exist", request.GetCategory())
	}

	if err := s.policies.DeletePolicyCategory(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.Empty{}, nil
}

func (s *serviceImpl) getPolicyCategorySet() (categorySet set.StringSet, err error) {
	policies, err := s.policies.GetPolicies()
	if err != nil {
		return
	}

	categorySet = set.NewStringSet()
	for _, p := range policies {
		for _, c := range p.GetCategories() {
			categorySet.Add(c)
		}
	}
	return
}

func (s *serviceImpl) addActivePolicy(policy *storage.Policy) error {
	s.processor.UpdatePolicy(policy)

	errorList := errorhelpers.NewErrorList("error adding policy to detection caches: ")

	if policies.AppliesAtBuildTime(policy) {
		errorList.AddError(s.buildTimePolicies.UpsertPolicy(policy))
	} else {
		errorList.AddError(s.buildTimePolicies.RemovePolicy(policy.GetId()))
	}

	errorList.AddError(s.lifecycleManager.UpsertPolicy(policy))
	return errorList.ToError()
}

func (s *serviceImpl) removeActivePolicy(policy *storage.Policy) error {
	s.processor.RemovePolicy(policy)

	errorList := errorhelpers.NewErrorList("error removing policy from detection: ")
	errorList.AddError(s.buildTimePolicies.RemovePolicy(policy.GetId()))
	errorList.AddError(s.lifecycleManager.RemovePolicy(policy.GetId()))
	return errorList.ToError()
}
