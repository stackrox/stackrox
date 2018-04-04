package service

import (
	"errors"
	"fmt"
	"sort"

	"bitbucket.org/stack-rox/apollo/central/datastore"
	"bitbucket.org/stack-rox/apollo/central/detection"
	"bitbucket.org/stack-rox/apollo/central/detection/matcher"
	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	uncategorizedCategory = `Uncategorized`
)

var (
	log = logging.LoggerForModule()
)

// NewPolicyService returns the PolicyService API.
func NewPolicyService(storage *datastore.DataStore, detector *detection.Detector) *PolicyService {
	return &PolicyService{
		datastore: storage,
		detector:  detector,
	}
}

// PolicyService is the struct that manages Policies API
type PolicyService struct {
	datastore *datastore.DataStore
	detector  *detection.Detector
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *PolicyService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterPolicyServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *PolicyService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterPolicyServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *PolicyService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, returnErrorCode(user.Any().Authorized(ctx))
}

// GetPolicy returns a policy by name.
func (s *PolicyService) GetPolicy(ctx context.Context, request *v1.ResourceByID) (*v1.Policy, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Policy id must be provided")
	}
	policy, exists, err := s.datastore.GetPolicy(request.GetId())
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

// GetPolicies retrieves all policies according to the request.
func (s *PolicyService) GetPolicies(ctx context.Context, request *v1.RawQuery) (*v1.PoliciesResponse, error) {
	resp := new(v1.PoliciesResponse)
	if request.GetQuery() == "" {
		policies, err := s.datastore.GetPolicies()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		resp.Policies = policies
	} else {
		parsedQuery, err := search.ParseRawQuery(request.GetQuery())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		policies, err := s.datastore.SearchRawPolicies(parsedQuery)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		resp.Policies = policies
	}
	sort.SliceStable(resp.Policies, func(i, j int) bool { return resp.Policies[i].GetName() < resp.Policies[j].GetName() })
	return resp, nil
}

// PostPolicy inserts a new policy into the system.
func (s *PolicyService) PostPolicy(ctx context.Context, request *v1.Policy) (*v1.Policy, error) {
	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Id field should be empty when posting a new policy")
	}
	policy, err := s.validatePolicy(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	id, err := s.datastore.AddPolicy(request)
	if err != nil {
		return nil, err
	}
	request.Id = id
	policy.Id = id
	s.detector.UpdatePolicy(policy)
	return request, nil
}

// PutPolicy updates a current policy in the system.
func (s *PolicyService) PutPolicy(ctx context.Context, request *v1.Policy) (*empty.Empty, error) {
	policy, err := s.validatePolicy(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	s.detector.UpdatePolicy(policy)
	if err := s.datastore.UpdatePolicy(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

// DeletePolicy deletes an policy from the system.
func (s *PolicyService) DeletePolicy(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "A policy id must be specified to delete a Policy")
	}
	if err := s.datastore.RemovePolicy(request.GetId()); err != nil {
		return nil, returnErrorCode(err)
	}
	s.detector.RemovePolicy(request.GetId())
	return &empty.Empty{}, nil
}

// ReassessPolicies manually triggers enrichment of all deployments, and re-assesses policies if there's updated data.
func (s *PolicyService) ReassessPolicies(context.Context, *empty.Empty) (*empty.Empty, error) {
	go s.detector.EnrichAndReprocess()

	return &empty.Empty{}, nil
}

// DryRunPolicy runs a dry run of the policy and determines what deployments would
func (s *PolicyService) DryRunPolicy(ctx context.Context, request *v1.Policy) (*v1.DryRunResponse, error) {
	policy, err := s.validatePolicy(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	var resp v1.DryRunResponse
	deployments, err := s.datastore.GetDeployments()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	for _, deployment := range deployments {
		alert, _, excluded := s.detector.Detect(detection.NewTask(deployment, v1.ResourceAction_DRYRUN_RESOURCE, policy))
		if alert != nil {
			violations := make([]string, 0, len(alert.GetViolations()))
			for _, v := range alert.GetViolations() {
				violations = append(violations, v.GetMessage())
			}
			resp.Alerts = append(resp.GetAlerts(), &v1.DryRunResponse_Alert{Deployment: deployment.GetName(), Violations: violations})
		} else if excluded != nil {
			resp.Excluded = append(resp.GetExcluded(), excluded)
		}
	}
	return &resp, nil
}

// GetPolicyCategories returns the categories of all policies.
func (s *PolicyService) GetPolicyCategories(context.Context, *empty.Empty) (*v1.PolicyCategoriesResponse, error) {
	categorySet, err := s.getPolicyCategorySet()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	response := new(v1.PolicyCategoriesResponse)
	response.Categories = make([]string, 0, len(categorySet))
	for c := range categorySet {
		response.Categories = append(response.Categories, c)
	}
	sort.Strings(response.Categories)

	return response, nil
}

// RenamePolicyCategory changes all usage of the category in policies to the requsted name.
func (s *PolicyService) RenamePolicyCategory(ctx context.Context, request *v1.RenamePolicyCategoryRequest) (*empty.Empty, error) {
	if request.GetOldCategory() == request.GetNewCategory() {
		return &empty.Empty{}, nil
	}

	if err := s.datastore.RenamePolicyCategory(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &empty.Empty{}, nil
}

// DeletePolicyCategory removes all usage of the category in policies. Policies may end up with no configured category.
func (s *PolicyService) DeletePolicyCategory(ctx context.Context, request *v1.DeletePolicyCategoryRequest) (*empty.Empty, error) {
	categorySet, err := s.getPolicyCategorySet()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if _, ok := categorySet[request.GetCategory()]; !ok {
		return nil, status.Errorf(codes.NotFound, "Policy Category %s does not exist", request.GetCategory())
	}

	if err := s.datastore.DeletePolicyCategory(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &empty.Empty{}, nil
}

func (s *PolicyService) validateScope(scope *v1.Scope) error {
	if scope.GetCluster() == "" {
		return nil
	}
	_, exists, err := s.datastore.GetCluster(scope.GetCluster())
	if err != nil {
		return fmt.Errorf("unable to get cluster id %s: %s", scope.GetCluster(), err)
	}
	if !exists {
		return fmt.Errorf("Cluster %s does not exist", scope.GetCluster())
	}
	return nil
}

func (s *PolicyService) validateWhitelist(whitelist *v1.Whitelist) error {
	// TODO(cgorman) once we have real whitelist support in UI, add validation for whitelist name
	if whitelist.GetContainer() == nil && whitelist.GetDeployment() == nil {
		return errors.New("All whitelists must have some criteria to match on")
	}
	if whitelist.GetContainer() != nil {
		imageName := whitelist.GetContainer().GetImageName()
		if imageName == nil {
			return errors.New("If container whitelist is defined, then image name must also be defined")
		}
		if imageName.GetSha() == "" && imageName.GetRegistry() == "" && imageName.GetRemote() == "" && imageName.GetTag() == "" {
			return errors.New("At least one field of image name must be populated (sha, registry, remote, tag)")
		}
	}
	if whitelist.GetDeployment() != nil {
		deployment := whitelist.GetDeployment()
		if deployment.GetScope() == nil && deployment.GetName() == "" {
			return errors.New("At least one field of deployment whitelist must be defined")
		}
		if deployment.GetScope() != nil {
			if err := s.validateScope(deployment.GetScope()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *PolicyService) validatePolicy(policy *v1.Policy) (*matcher.Policy, error) {
	if policy.GetName() == "" {
		return nil, errors.New("policy must have a name")
	}
	if policy.GetSeverity() == v1.Severity_UNSET_SEVERITY {
		return nil, errors.New("policy must have a severity")
	}
	if policy.GetImagePolicy() == nil && policy.GetConfigurationPolicy() == nil && policy.GetPrivilegePolicy() == nil {
		return nil, errors.New("policy must have at least one segment configured")
	}
	if len(policy.GetCategories()) == 0 {
		return nil, errors.New("policy must have at least one category configured")
	}
	categorySet := make(map[string]struct{})
	for _, c := range policy.GetCategories() {
		categorySet[c] = struct{}{}
	}
	if len(categorySet) != len(policy.GetCategories()) {
		return nil, errors.New("policy cannot contain duplicate categories")
	}

	for _, n := range policy.GetNotifiers() {
		_, exists, err := s.datastore.GetNotifier(n)
		if err != nil {
			return nil, fmt.Errorf("Error checking if notifier %v is valid", n)
		}
		if !exists {
			return nil, fmt.Errorf("Notifier %v does not exist", n)
		}
	}
	for _, scope := range policy.GetScope() {
		if err := s.validateScope(scope); err != nil {
			return nil, err
		}
	}

	for _, whitelist := range policy.GetWhitelists() {
		if err := s.validateWhitelist(whitelist); err != nil {
			return nil, err
		}
	}

	matcherPolicy, err := matcher.New(policy)
	if err != nil {
		return nil, fmt.Errorf("Policy could not be edited due to: %+v", err)
	}
	return matcherPolicy, nil
}

func (s *PolicyService) getPolicyCategorySet() (map[string]struct{}, error) {
	policies, err := s.datastore.GetPolicies()
	if err != nil {
		return nil, err
	}

	categorySet := make(map[string]struct{})
	for _, p := range policies {
		for _, c := range p.GetCategories() {
			categorySet[c] = struct{}{}
		}
	}

	return categorySet, nil
}
