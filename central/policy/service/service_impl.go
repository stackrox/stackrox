package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/backgroundtasks"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/searchbasedpolicies/matcher"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
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
			"/v1.PolicyService/QueryDryRunJobStatus",
		},
		user.With(permissions.Modify(resources.Policy)): {
			"/v1.PolicyService/PostPolicy",
			"/v1.PolicyService/PutPolicy",
			"/v1.PolicyService/PatchPolicy",
			"/v1.PolicyService/DeletePolicy",
			"/v1.PolicyService/DryRunPolicy",
			"/v1.PolicyService/SubmitDryRunPolicyJob",
			"/v1.PolicyService/CancelDryRunJob",
			"/v1.PolicyService/RenamePolicyCategory",
			"/v1.PolicyService/DeletePolicyCategory",
			"/v1.PolicyService/EnableDisablePolicyNotification",
		},
	})
)

const (
	uncategorizedCategory = `Uncategorized`
	dryRunParallelism     = 8
	identityUIDKey        = "identityUID"
)

var (
	policySyncReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Policy)))
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	policies          datastore.DataStore
	clusters          clusterDataStore.DataStore
	deployments       deploymentDataStore.DataStore
	notifiers         notifierDataStore.DataStore
	reprocessor       reprocessor.Loop
	connectionManager connection.Manager

	buildTimePolicies detection.PolicySet
	testMatchBuilder  matcher.Builder
	lifecycleManager  lifecycle.Manager
	processor         notifierProcessor.Processor
	metadataCache     expiringcache.Cache
	scanCache         expiringcache.Cache

	validator *policyValidator

	dryRunPolicyJobManager backgroundtasks.Manager
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
	policy, exists, err := s.policies.GetPolicy(ctx, request.GetId())
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
			Notifiers:       p.GetNotifiers(),
			LastUpdated:     p.GetLastUpdated(),
		})
	}
	return listPolicies
}

// ListPolicies retrieves all policies in ListPolicy form according to the request.
func (s *serviceImpl) ListPolicies(ctx context.Context, request *v1.RawQuery) (*v1.ListPoliciesResponse, error) {
	resp := new(v1.ListPoliciesResponse)
	if request.GetQuery() == "" {
		policies, err := s.policies.GetPolicies(ctx)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		resp.Policies = convertPoliciesToListPolicies(policies)
	} else {
		parsedQuery, err := search.ParseQuery(request.GetQuery())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		policies, err := s.policies.SearchRawPolicies(ctx, parsedQuery)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		resp.Policies = convertPoliciesToListPolicies(policies)
	}

	return resp, nil
}

// PostPolicy inserts a new policy into the system.
func (s *serviceImpl) PostPolicy(ctx context.Context, request *storage.Policy) (*storage.Policy, error) {
	request.LastUpdated = protoconv.ConvertTimeToTimestamp(time.Now())

	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Id field should be empty when posting a new policy")
	}
	if err := s.validator.validate(ctx, request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	id, err := s.policies.AddPolicy(ctx, request)
	if err != nil {
		return nil, err
	}
	request.Id = id

	if err := s.addActivePolicy(request); err != nil {
		return nil, errors.Wrap(err, "Policy could not be edited due to")
	}

	if err := s.syncPoliciesWithSensors(); err != nil {
		return nil, err
	}

	return request, nil
}

// PutPolicy updates a current policy in the system.
func (s *serviceImpl) PutPolicy(ctx context.Context, request *storage.Policy) (*v1.Empty, error) {
	request.LastUpdated = protoconv.ConvertTimeToTimestamp(time.Now())

	if err := s.validator.validate(ctx, request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := s.policies.UpdatePolicy(ctx, request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := s.addActivePolicy(request); err != nil {
		return nil, errors.Wrap(err, "Policy could not be edited due to")
	}

	if err := s.syncPoliciesWithSensors(); err != nil {
		return nil, err
	}

	return &v1.Empty{}, nil
}

// PatchPolicy patches a current policy in the system.
func (s *serviceImpl) PatchPolicy(ctx context.Context, request *v1.PatchPolicyRequest) (*v1.Empty, error) {
	policy, exists, err := s.policies.GetPolicy(ctx, request.GetId())
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

	policy, exists, err := s.policies.GetPolicy(ctx, request.GetId())
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Policy with id '%s' not found", request.GetId()))
	}

	if err := s.policies.RemovePolicy(ctx, request.GetId()); err != nil {
		return nil, err
	}

	if err := s.removeActivePolicy(policy); err != nil {
		return nil, err
	}

	if err := s.syncPoliciesWithSensors(); err != nil {
		return nil, err
	}

	return &v1.Empty{}, nil
}

// ReassessPolicies manually triggers enrichment of all deployments, and re-assesses policies if there's updated data.
func (s *serviceImpl) ReassessPolicies(context.Context, *v1.Empty) (*v1.Empty, error) {
	// Invalidate scan and metadata caches
	s.metadataCache.RemoveAll()
	s.scanCache.RemoveAll()

	go s.reprocessor.ShortCircuit()
	return &v1.Empty{}, nil
}

func (s *serviceImpl) SubmitDryRunPolicyJob(ctx context.Context, request *storage.Policy) (*v1.JobId, error) {
	if err := s.validator.validate(ctx, request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	t := func(c concurrency.ErrorWaitable, res *backgroundtasks.ExecutionResult) error {
		resp, err := s.predicateBasedDryRunPolicy(ctx, c, request)
		if err != nil {
			return err
		}

		res.Result = resp
		return nil
	}

	metadata := map[string]interface{}{identityUIDKey: authn.IdentityFromContext(ctx).UID()}
	id, err := s.dryRunPolicyJobManager.AddTask(metadata, t)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add dry-run job: %v", err)
	}

	return &v1.JobId{
		JobId: id,
	}, nil
}

func (s *serviceImpl) QueryDryRunJobStatus(ctx context.Context, jobid *v1.JobId) (*v1.DryRunJobStatusResponse, error) {
	metadata, res, completed, err := s.dryRunPolicyJobManager.GetTaskStatusAndMetadata(jobid.JobId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := checkIdentityFromMetadata(ctx, metadata); err != nil {
		return nil, err
	}

	resp := &v1.DryRunJobStatusResponse{
		Pending: !completed,
	}

	if completed {
		resp.Result, _ = res.(*v1.DryRunResponse)
		if resp.Result == nil {
			return nil, status.Error(codes.Internal, "Invalid response.")
		}
	}

	return resp, nil
}

func (s *serviceImpl) CancelDryRunJob(ctx context.Context, jobid *v1.JobId) (*v1.Empty, error) {
	metadata, _, _, err := s.dryRunPolicyJobManager.GetTaskStatusAndMetadata(jobid.JobId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := checkIdentityFromMetadata(ctx, metadata); err != nil {
		return nil, err
	}

	if err := s.dryRunPolicyJobManager.CancelTask(jobid.JobId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &v1.Empty{}, nil
}

func (s *serviceImpl) predicateBasedDryRunPolicy(ctx context.Context, cancelCtx concurrency.ErrorWaitable, request *storage.Policy) (*v1.DryRunResponse, error) {
	if err := s.validator.validate(ctx, request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var resp v1.DryRunResponse
	// Dry runs do not apply to policies with whitelists because they are evaluated through the process indicator pipeline
	if request.GetFields().GetWhitelistEnabled() {
		return &resp, nil
	}

	searchBasedMatcher, err := s.testMatchBuilder.ForPolicy(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("couldn't construct matcher: %s", err))
	}

	compiledPolicy, err := detection.NewCompiledPolicy(request, searchBasedMatcher)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid policy: %v", err)
	}

	deploymentIds, err := s.deployments.GetDeploymentIDs()
	if err != nil {
		return nil, err
	}

	pChan := make(chan struct{}, dryRunParallelism)
	alertChan := make(chan *v1.DryRunResponse_Alert)
	var wg sync.WaitGroup
	go func() {
		for {
			select {
			case alert, ok := <-alertChan:
				// channel is closed
				if !ok {
					return
				}
				resp.Alerts = append(resp.Alerts, alert)
			case <-cancelCtx.Done():
				// context canceled or expired
				return
			}
		}
	}()

	for _, id := range deploymentIds {
		if err := cancelCtx.Err(); err != nil {
			return nil, err
		}

		pChan <- struct{}{}
		wg.Add(1)
		go func(depId string) {
			defer func() {
				wg.Done()
				<-pChan
			}()

			deployment, exists, err := s.deployments.GetDeployment(ctx, depId)
			if !exists || err != nil {
				return
			}

			images, err := s.deployments.GetImagesForDeployment(ctx, deployment)
			if err != nil {
				return
			}

			violations, err := searchBasedMatcher.MatchOne(ctx, deployment, images, nil)
			if err != nil {
				log.Errorf("failed policy matching: %s", err.Error())
				return
			}

			if !compiledPolicy.AppliesTo(deployment) {
				return
			}

			// Collect the violation messages as strings for the output.
			convertedViolations := make([]string, 0, len(violations.AlertViolations))
			for _, violation := range violations.AlertViolations {
				convertedViolations = append(convertedViolations, violation.GetMessage())
			}
			if violations.ProcessViolation != nil {
				convertedViolations = append(convertedViolations, violations.ProcessViolation.GetMessage())
			}

			alertChan <- &v1.DryRunResponse_Alert{Deployment: deployment.GetName(), Violations: convertedViolations}
		}(id)
	}

	wg.Wait()
	close(alertChan)
	return &resp, nil
}

// DryRunPolicy runs a dry run of the policy and determines what deployments would violate it
func (s *serviceImpl) DryRunPolicy(ctx context.Context, request *storage.Policy) (*v1.DryRunResponse, error) {
	if features.DryRunPolicyJobMechanism.Enabled() {
		return s.predicateBasedDryRunPolicy(ctx, ctx, request)
	}

	if err := s.validator.validate(ctx, request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var resp v1.DryRunResponse
	// Dry runs do not apply to policies with whitelists because they are evaluated through the process indicator pipeline
	if request.GetFields().GetWhitelistEnabled() {
		return &resp, nil
	}

	searchBasedMatcher, err := s.testMatchBuilder.ForPolicy(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("couldn't construct matcher: %s", err))
	}

	compiledPolicy, err := detection.NewCompiledPolicy(request, searchBasedMatcher)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid policy: %v", err)
	}

	violationsPerDeployment, err := searchBasedMatcher.Match(ctx, s.deployments)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed policy matching: %s", err))
	}

	for deploymentID, violations := range violationsPerDeployment {
		if len(violations.AlertViolations) == 0 && violations.ProcessViolation == nil {
			continue
		}
		deployment, exists, err := s.deployments.GetDeployment(ctx, deploymentID)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("retrieving deployment '%s': %s", deploymentID, err))
		}
		if !exists {
			// Maybe the deployment was deleted around the time of the dry-run.
			continue
		}
		if !compiledPolicy.AppliesTo(deployment) {
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
func (s *serviceImpl) GetPolicyCategories(ctx context.Context, _ *v1.Empty) (*v1.PolicyCategoriesResponse, error) {
	categorySet, err := s.getPolicyCategorySet(ctx)
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

	if err := s.policies.RenamePolicyCategory(ctx, request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := s.syncPoliciesWithSensors(); err != nil {
		return nil, err
	}

	return &v1.Empty{}, nil
}

// DeletePolicyCategory removes all usage of the category in policies. Policies may end up with no configured category.
func (s *serviceImpl) DeletePolicyCategory(ctx context.Context, request *v1.DeletePolicyCategoryRequest) (*v1.Empty, error) {
	categorySet, err := s.getPolicyCategorySet(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !categorySet.Contains(request.GetCategory()) {
		return nil, status.Errorf(codes.NotFound, "Policy Category %s does not exist", request.GetCategory())
	}

	if err := s.policies.DeletePolicyCategory(ctx, request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := s.syncPoliciesWithSensors(); err != nil {
		return nil, err
	}

	return &v1.Empty{}, nil
}

func (s *serviceImpl) getPolicyCategorySet(ctx context.Context) (categorySet set.StringSet, err error) {
	policies, err := s.policies.GetPolicies(ctx)
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
	errorList := errorhelpers.NewErrorList("error removing policy from detection: ")
	errorList.AddError(s.buildTimePolicies.RemovePolicy(policy.GetId()))
	errorList.AddError(s.lifecycleManager.RemovePolicy(policy.GetId()))
	return errorList.ToError()
}

func (s *serviceImpl) EnableDisablePolicyNotification(ctx context.Context, request *v1.EnableDisablePolicyNotificationRequest) (*v1.Empty, error) {
	if request.GetPolicyId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Policy ID must be specified")
	}
	var err error
	if request.GetDisable() {
		err = s.disablePolicyNotification(ctx, request.GetPolicyId(), request.GetNotifierIds())
	} else {
		err = s.enablePolicyNotification(ctx, request.GetPolicyId(), request.GetNotifierIds())
	}

	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) enablePolicyNotification(ctx context.Context, policyID string, notifierIDs []string) error {
	if len(notifierIDs) == 0 {
		return status.Error(codes.InvalidArgument, "Notifier IDs must be specified")
	}

	policy, exists, err := s.policies.GetPolicy(ctx, policyID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to retrieve policy: %v", err)
	}
	if !exists {
		return status.Errorf(codes.NotFound, "Policy %q not found", policyID)
	}
	notifierSet := set.NewStringSet(policy.Notifiers...)
	errorList := errorhelpers.NewErrorList("unable to use all requested notifiers")
	for _, notifierID := range notifierIDs {
		_, exists, err := s.notifiers.GetNotifier(ctx, notifierID)
		if err != nil {
			errorList.AddError(err)
			continue
		}
		if !exists {
			errorList.AddStringf("notifier with id: %s not found", notifierID)
			continue
		} else {
			if notifierSet.Contains(notifierID) {
				continue
			}
			policy.Notifiers = append(policy.Notifiers, notifierID)
		}
	}

	_, err = s.PutPolicy(ctx, policy)
	if err != nil {
		errorList.AddStringf("policy could not be updated with notifier %v", err)
	}

	err = errorList.ToError()
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	return nil
}

func (s *serviceImpl) syncPoliciesWithSensors() error {
	policies, err := s.policies.GetPolicies(policySyncReadCtx)
	if err != nil {
		return errors.Wrap(err, "error reading policies from store")
	}
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_PolicySync{
			PolicySync: &central.PolicySync{
				Policies: policies,
			},
		},
	}
	s.connectionManager.BroadcastMessage(msg)
	return nil
}

func (s *serviceImpl) disablePolicyNotification(ctx context.Context, policyID string, notifierIDs []string) error {
	policy, exists, err := s.policies.GetPolicy(ctx, policyID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to retrieve policy: %v", err)
	}
	if !exists {
		return status.Errorf(codes.NotFound, "Policy %q not found", policyID)
	}
	notifierSet := set.NewStringSet(policy.Notifiers...)
	if notifierSet.Cardinality() == 0 {
		return nil
	}
	errorList := errorhelpers.NewErrorList("unable to delete all requested notifiers")
	for _, notifierID := range notifierIDs {
		if !notifierSet.Contains(notifierID) {
			continue
		}
		notifierSet.Remove(notifierID)
	}

	policy.Notifiers = notifierSet.AsSlice()
	_, err = s.PutPolicy(ctx, policy)
	if err != nil {
		errorList.AddStringf("policy could not be updated with notifier %v", err)
	}

	err = errorList.ToError()
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}

func checkIdentityFromMetadata(ctx context.Context, metadata map[string]interface{}) error {
	identityUID, ok := metadata[identityUIDKey].(string)
	if !ok {
		return status.Error(codes.Internal, "Invalid job.")
	}

	if identityUID != authn.IdentityFromContext(ctx).UID() {
		return status.Error(codes.PermissionDenied, "Unauthorized access.")
	}

	return nil
}
