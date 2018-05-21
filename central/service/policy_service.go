package service

import (
	"errors"
	"fmt"
	"regexp"
	"sort"

	"bitbucket.org/stack-rox/apollo/central/datastore"
	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/detection"
	"bitbucket.org/stack-rox/apollo/central/detection/matcher"
	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/errorhelpers"
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
func NewPolicyService(policies datastore.PolicyDataStore, clusters datastore.ClusterDataStore, deployments datastore.DeploymentDataStore, notifiers db.NotifierStorage, detector *detection.Detector) *PolicyService {
	return &PolicyService{
		policies:    policies,
		clusters:    clusters,
		deployments: deployments,

		notifiers: notifiers,
		detector:  detector,

		validator: newPolicyValidator(notifiers, clusters),
	}
}

// PolicyService is the struct that manages Policies API
type PolicyService struct {
	policies    datastore.PolicyDataStore
	clusters    datastore.ClusterDataStore
	deployments datastore.DeploymentDataStore

	notifiers db.NotifierStorage
	detector  *detection.Detector

	validator *policyValidator
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

// GetPolicies retrieves all policies according to the request.
func (s *PolicyService) GetPolicies(ctx context.Context, request *v1.RawQuery) (*v1.PoliciesResponse, error) {
	resp := new(v1.PoliciesResponse)
	if request.GetQuery() == "" {
		policies, err := s.policies.GetPolicies()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		resp.Policies = policies
	} else {
		parsedQuery, err := search.ParseRawQuery(request.GetQuery())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		policies, err := s.policies.SearchRawPolicies(parsedQuery)
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
	if err := s.validator.validate(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	policy, err := matcher.New(request)
	if err != nil {
		return nil, fmt.Errorf("Policy could not be edited due to: %+v", err)
	}

	id, err := s.policies.AddPolicy(request)
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
	if err := s.validator.validate(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	policy, err := matcher.New(request)
	if err != nil {
		return nil, fmt.Errorf("Policy could not be edited due to: %+v", err)
	}

	s.detector.UpdatePolicy(policy)
	if err := s.policies.UpdatePolicy(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

// DeletePolicy deletes an policy from the system.
func (s *PolicyService) DeletePolicy(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "A policy id must be specified to delete a Policy")
	}
	if err := s.policies.RemovePolicy(request.GetId()); err != nil {
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
	if err := s.validator.validate(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	policy, err := matcher.New(request)
	if err != nil {
		return nil, fmt.Errorf("Policy could not be edited due to: %+v", err)
	}

	var resp v1.DryRunResponse
	deployments, err := s.deployments.GetDeployments()
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

	if err := s.policies.RenamePolicyCategory(request); err != nil {
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

	if err := s.policies.DeletePolicyCategory(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &empty.Empty{}, nil
}

func (s *PolicyService) getPolicyCategorySet() (map[string]struct{}, error) {
	policies, err := s.policies.GetPolicies()
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

// Below is the validation to be run on new policies.
/////////////////////////////////////////////////////

func newPolicyValidator(notifierStorage db.NotifierStorage, clusterStorage db.ClusterStorage) *policyValidator {
	return &policyValidator{
		notifierStorage:      notifierStorage,
		clusterStorage:       clusterStorage,
		nameValidator:        regexp.MustCompile(`^[^\n\r\$]{5,64}$`),
		descriptionValidator: regexp.MustCompile(`^[^\$]{1,256}$`),
	}
}

// policyValidator validates the incoming policy.
type policyValidator struct {
	notifierStorage      db.NotifierStorage
	clusterStorage       db.ClusterStorage
	nameValidator        *regexp.Regexp
	descriptionValidator *regexp.Regexp
}

func (s *policyValidator) validate(policy *v1.Policy) error {
	errors := make([]error, 0)
	if err := s.validateName(policy); err != nil {
		errors = append(errors, err)
	}
	if err := s.validateDescription(policy); err != nil {
		errors = append(errors, err)
	}
	if err := s.validateSeverity(policy); err != nil {
		errors = append(errors, err)
	}
	if err := s.validateImagePolicy(policy); err != nil {
		errors = append(errors, err)
	}
	if err := s.validateCategories(policy); err != nil {
		errors = append(errors, err)
	}
	if err := s.validateScopes(policy); err != nil {
		errors = append(errors, err)
	}
	if err := s.validateWhitelists(policy); err != nil {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		return errorhelpers.FormatErrors("policy invalid", errors)
	}
	return nil
}

func (s *policyValidator) validateName(policy *v1.Policy) error {
	if policy.GetName() == "" || !s.nameValidator.MatchString(policy.GetName()) {
		return errors.New("policy must have a name, at least 5 chars long, and contain no punctuation or special characters")
	}
	return nil
}

func (s *policyValidator) validateDescription(policy *v1.Policy) error {
	if policy.GetDescription() != "" && !s.descriptionValidator.MatchString(policy.GetDescription()) {
		return errors.New("description, when present, should be of sentence form, and not contain more than 200 characters")
	}
	return nil
}

func (s *policyValidator) validateSeverity(policy *v1.Policy) error {
	if policy.GetSeverity() == v1.Severity_UNSET_SEVERITY {
		return errors.New("a policy must have a severity")
	}
	return nil
}

func (s *policyValidator) validateImagePolicy(policy *v1.Policy) error {
	if policy.GetImagePolicy() == nil && policy.GetConfigurationPolicy() == nil && policy.GetPrivilegePolicy() == nil {
		return errors.New("a policy must have at least one segment configured")
	}
	return nil
}

func (s *policyValidator) validateCategories(policy *v1.Policy) error {
	if len(policy.GetCategories()) == 0 {
		return errors.New("a policy must have one of Image Policy, Configuration Policy, or Privilege Policy")
	}
	categorySet := make(map[string]struct{})
	for _, c := range policy.GetCategories() {
		categorySet[c] = struct{}{}
	}
	if len(categorySet) != len(policy.GetCategories()) {
		return errors.New("a policy cannot contain duplicate categories")
	}
	return nil
}

func (s *policyValidator) validateNotifiers(policy *v1.Policy) error {
	for _, n := range policy.GetNotifiers() {
		_, exists, err := s.notifierStorage.GetNotifier(n)
		if err != nil {
			return fmt.Errorf("error checking if notifier %s is valid", n)
		}
		if !exists {
			return fmt.Errorf("notifier %s does not exist", n)
		}
	}
	return nil
}

func (s *policyValidator) validateScopes(policy *v1.Policy) error {
	for _, scope := range policy.GetScope() {
		if err := s.validateScope(scope); err != nil {
			return err
		}
	}
	return nil
}

func (s *policyValidator) validateWhitelists(policy *v1.Policy) error {
	for _, whitelist := range policy.GetWhitelists() {
		if err := s.validateWhitelist(whitelist); err != nil {
			return err
		}
	}
	return nil
}

func (s *policyValidator) validateWhitelist(whitelist *v1.Whitelist) error {
	// TODO(cgorman) once we have real whitelist support in UI, add validation for whitelist name
	if whitelist.GetContainer() == nil && whitelist.GetDeployment() == nil {
		return errors.New("all whitelists must have some criteria to match on")
	}
	if whitelist.GetContainer() != nil {
		if err := s.validateContainerWhitelist(whitelist); err != nil {
			return err
		}
	}
	if whitelist.GetDeployment() != nil {
		if err := s.validateDeploymentWhitelist(whitelist); err != nil {
			return err
		}
	}
	return nil
}

func (s *policyValidator) validateContainerWhitelist(whitelist *v1.Whitelist) error {
	imageName := whitelist.GetContainer().GetImageName()
	if imageName == nil {
		return errors.New("if container whitelist is defined, then image name must also be defined")
	}
	if imageName.GetSha() == "" && imageName.GetRegistry() == "" && imageName.GetRemote() == "" && imageName.GetTag() == "" {
		return errors.New("at least one field of image name must be populated (sha, registry, remote, tag)")
	}
	return nil
}

func (s *policyValidator) validateDeploymentWhitelist(whitelist *v1.Whitelist) error {
	deployment := whitelist.GetDeployment()
	if deployment.GetScope() == nil && deployment.GetName() == "" {
		return errors.New("at least one field of deployment whitelist must be defined")
	}
	if deployment.GetScope() != nil {
		if err := s.validateScope(deployment.GetScope()); err != nil {
			return err
		}
	}
	return nil
}

func (s *policyValidator) validateScope(scope *v1.Scope) error {
	if scope.GetCluster() == "" {
		return nil
	}
	_, exists, err := s.clusterStorage.GetCluster(scope.GetCluster())
	if err != nil {
		return fmt.Errorf("unable to get cluster id %s: %s", scope.GetCluster(), err)
	}
	if !exists {
		return fmt.Errorf("cluster %s does not exist", scope.GetCluster())
	}
	return nil
}
