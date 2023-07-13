package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	networkPolicyDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/backgroundtasks"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/networkpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	mitreDS "github.com/stackrox/rox/pkg/mitre/datastore"
	mitreUtils "github.com/stackrox/rox/pkg/mitre/utils"
	"github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate/basematchers"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log = logging.LoggerForModule()

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.WorkflowAdministration)): {
			"/v1.PolicyService/GetPolicy",
			"/v1.PolicyService/ListPolicies",
			"/v1.PolicyService/ReassessPolicies",
			"/v1.PolicyService/GetPolicyCategories",
			"/v1.PolicyService/QueryDryRunJobStatus",
			"/v1.PolicyService/ExportPolicies",
			"/v1.PolicyService/PolicyFromSearch",
			"/v1.PolicyService/GetPolicyMitreVectors",
		},
		user.With(permissions.Modify(resources.WorkflowAdministration)): {
			"/v1.PolicyService/PostPolicy",
			"/v1.PolicyService/PutPolicy",
			"/v1.PolicyService/PatchPolicy",
			"/v1.PolicyService/DeletePolicy",
			"/v1.PolicyService/DryRunPolicy",
			"/v1.PolicyService/SubmitDryRunPolicyJob",
			"/v1.PolicyService/CancelDryRunJob",
			"/v1.PolicyService/EnableDisablePolicyNotification",
			"/v1.PolicyService/ImportPolicies",
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
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))

	partialListPolicyGroups = set.NewStringSet(fieldnames.ImageComponent, fieldnames.DockerfileLine, fieldnames.EnvironmentVariable)
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	v1.UnimplementedPolicyServiceServer

	policies          datastore.DataStore
	clusters          clusterDataStore.DataStore
	deployments       deploymentDataStore.DataStore
	networkPolicies   networkPolicyDS.DataStore
	notifiers         notifierDataStore.DataStore
	mitreStore        mitreDS.AttackReadOnlyDataStore
	reprocessor       reprocessor.Loop
	connectionManager connection.Manager

	buildTimePolicies detection.PolicySet
	lifecycleManager  lifecycle.Manager
	processor         notifier.Processor
	metadataCache     expiringcache.Cache

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
	return s.getPolicy(ctx, request.GetId())
}

func (s *serviceImpl) getPolicy(ctx context.Context, id string) (*storage.Policy, error) {
	if id == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Policy ID must be provided")
	}
	policy, exists, err := s.policies.GetPolicy(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "policy with ID '%s' does not exist", id)
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
			EventSource:     p.GetEventSource(),
			IsDefault:       p.GetIsDefault(),
		})
	}
	return listPolicies
}

// ListPolicies retrieves all policies in ListPolicy form according to the request.
func (s *serviceImpl) ListPolicies(ctx context.Context, request *v1.RawQuery) (*v1.ListPoliciesResponse, error) {
	resp := new(v1.ListPoliciesResponse)
	if request.GetQuery() == "" {
		policies, err := s.policies.GetAllPolicies(ctx)
		if err != nil {
			return nil, err
		}
		resp.Policies = convertPoliciesToListPolicies(policies)
	} else {
		parsedQuery, err := search.ParseQuery(request.GetQuery())
		if err != nil {
			return nil, errors.Wrap(errox.InvalidArgs, err.Error())
		}
		policies, err := s.policies.SearchRawPolicies(ctx, parsedQuery)
		if err != nil {
			return nil, err
		}
		resp.Policies = convertPoliciesToListPolicies(policies)
	}

	return resp, nil
}

func (s *serviceImpl) convertAndValidate(ctx context.Context, p *storage.Policy, options ...booleanpolicy.ValidateOption) error {
	if err := policyversion.EnsureConvertedToLatest(p); err != nil {
		return errors.Wrapf(errox.InvalidArgs, "Could not ensure policy format: %v", err.Error())
	}

	if err := s.validator.validate(ctx, p, options...); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	return nil
}

func (s *serviceImpl) addOrUpdatePolicy(ctx context.Context, request *storage.Policy, extraValidateFunc func(*storage.Policy) error, updateFunc func(context.Context, *storage.Policy) error, options ...booleanpolicy.ValidateOption) (*storage.Policy, error) {
	if extraValidateFunc != nil {
		if err := extraValidateFunc(request); err != nil {
			return nil, err
		}
	}

	options = append(options, booleanpolicy.ValidateNoFromInDockerfileLine())

	if err := s.convertAndValidate(ctx, request, options...); err != nil {
		return nil, err
	}

	request.LastUpdated = protoconv.ConvertTimeToTimestamp(time.Now())
	if err := updateFunc(ctx, request); err != nil {
		return nil, err
	}

	if err := s.addActivePolicy(request); err != nil {
		return nil, errors.Wrap(err, "Policy could not be edited due to")
	}

	if err := s.syncPoliciesWithSensors(); err != nil {
		return nil, err
	}

	return request, nil
}

func ensureIDEmpty(p *storage.Policy) error {
	if p.GetId() != "" {
		return errors.Wrap(errox.InvalidArgs, "Id field should be empty when posting a new policy")
	}
	return nil
}

func (s *serviceImpl) addPolicyToStoreAndSetID(ctx context.Context, p *storage.Policy) error {
	id, err := s.policies.AddPolicy(ctx, p)
	if err != nil {
		return err
	}
	p.Id = id
	return nil
}

// GetPolicyMitreVectors returns a policy's MITRE ATT&CK vectors.
func (s *serviceImpl) GetPolicyMitreVectors(ctx context.Context, request *v1.GetPolicyMitreVectorsRequest) (*v1.GetPolicyMitreVectorsResponse, error) {
	policy, err := s.getPolicy(ctx, request.GetId())
	if err != nil {
		return nil, err
	}

	fullVectors, err := mitreUtils.GetFullMitreAttackVectors(s.mitreStore, policy)
	if err != nil {
		return nil, errors.Wrapf(err, "fetching MITRE ATT&CK vectors for policy %q", request.GetId())
	}

	resp := &v1.GetPolicyMitreVectorsResponse{
		Vectors: fullVectors,
	}

	if !request.GetOptions().GetExcludePolicy() {
		resp.Policy = policy
	}

	return resp, nil
}

// PostPolicy inserts a new policy into the system.
func (s *serviceImpl) PostPolicy(ctx context.Context, request *v1.PostPolicyRequest) (*storage.Policy, error) {
	options := []booleanpolicy.ValidateOption{}

	if request.GetEnableStrictValidation() {
		options = append(options, booleanpolicy.ValidateEnvVarSourceRestrictions())
	}
	return s.addOrUpdatePolicy(ctx, request.GetPolicy(), ensureIDEmpty, s.addPolicyToStoreAndSetID, options...)
}

// PutPolicy updates a current policy in the system.
func (s *serviceImpl) PutPolicy(ctx context.Context, request *storage.Policy) (*v1.Empty, error) {
	_, err := s.addOrUpdatePolicy(ctx, request, nil, s.policies.UpdatePolicy)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

// PatchPolicy patches a current policy in the system.
func (s *serviceImpl) PatchPolicy(ctx context.Context, request *v1.PatchPolicyRequest) (*v1.Empty, error) {
	policy, exists, err := s.policies.GetPolicy(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "Policy with id '%s' not found", request.GetId())
	}
	if request.SetDisabled != nil {
		policy.Disabled = request.GetDisabled()
	}

	return s.PutPolicy(ctx, policy)
}

// DeletePolicy deletes an policy from the system.
func (s *serviceImpl) DeletePolicy(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "A policy id must be specified to delete a Policy")
	}

	if err := s.policies.RemovePolicy(ctx, request.GetId()); err != nil {
		return nil, err
	}

	if err := s.removeActivePolicy(request.GetId()); err != nil {
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

	s.reprocessor.ShortCircuit()
	return &v1.Empty{}, nil
}

func (s *serviceImpl) SubmitDryRunPolicyJob(ctx context.Context, request *storage.Policy) (*v1.JobId, error) {
	if err := s.convertAndValidate(ctx, request); err != nil {
		return nil, err
	}

	t := func(c concurrency.ErrorWaitable) (interface{}, error) {
		ctx := contextutil.WithValuesFrom(context.Background(), ctx)
		return s.predicateBasedDryRunPolicy(ctx, c, request)
	}

	identity, err := authn.IdentityFromContext(ctx)
	if err != nil {
		return nil, err
	}
	metadata := map[string]interface{}{identityUIDKey: identity.UID()}
	id, err := s.dryRunPolicyJobManager.AddTask(metadata, t)
	if err != nil {
		return nil, errors.Errorf("failed to add dry-run job: %v", err)
	}

	return &v1.JobId{
		JobId: id,
	}, nil
}

func (s *serviceImpl) QueryDryRunJobStatus(ctx context.Context, jobid *v1.JobId) (*v1.DryRunJobStatusResponse, error) {
	metadata, res, completed, err := s.dryRunPolicyJobManager.GetTaskStatusAndMetadata(jobid.JobId)
	if err != nil {
		return nil, err
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
			return nil, errors.New("Invalid response.")
		}
	}

	return resp, nil
}

func (s *serviceImpl) CancelDryRunJob(ctx context.Context, jobid *v1.JobId) (*v1.Empty, error) {
	metadata, _, _, err := s.dryRunPolicyJobManager.GetTaskStatusAndMetadata(jobid.JobId)
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	if err := checkIdentityFromMetadata(ctx, metadata); err != nil {
		return nil, err
	}

	if err := s.dryRunPolicyJobManager.CancelTask(jobid.JobId); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	return &v1.Empty{}, nil
}

func (s *serviceImpl) getNetworkPoliciesForDeployment(ctx context.Context, dep *storage.Deployment) (*augmentedobjs.NetworkPoliciesApplied, error) {
	storedPolicies, err := s.networkPolicies.GetNetworkPolicies(ctx, dep.GetClusterId(), dep.GetNamespace())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get network policies for clusterId %s on namespace %s", dep.GetClusterId(), dep.GetNamespace())
	}
	matchedNetworkPolicies := networkpolicy.FilterForDeployment(storedPolicies, dep)
	return networkpolicy.GenerateNetworkPoliciesAppliedObj(matchedNetworkPolicies), nil
}

func (s *serviceImpl) predicateBasedDryRunPolicy(ctx context.Context, cancelCtx concurrency.ErrorWaitable, request *storage.Policy) (*v1.DryRunResponse, error) {
	var resp v1.DryRunResponse

	// Dry runs do not apply to policies with excluded scopes or runtime lifecycle stage because they are evaluated
	// through the process indicator pipeline
	if policies.AppliesAtRunTime(request) {
		return &resp, nil
	}

	compiledPolicy, err := detection.CompilePolicy(request)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "invalid policy: %v", err)
	}

	deploymentIds, err := s.deployments.GetDeploymentIDs(ctx)
	if err != nil {
		return nil, err
	}

	pChan := make(chan struct{}, dryRunParallelism)
	alertChan := make(chan *v1.DryRunResponse_Alert)
	allAlertsProcessedSig := concurrency.NewSignal()
	go func() {
		for {
			select {
			case alert, ok := <-alertChan:
				// channel is closed
				if !ok {
					allAlertsProcessedSig.Signal()
					return
				}
				resp.Alerts = append(resp.Alerts, alert)
			case <-cancelCtx.Done():
				// context canceled or expired
				return
			}
		}
	}()

	var wg sync.WaitGroup
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

			if !compiledPolicy.AppliesTo(deployment) {
				return
			}

			images, err := s.deployments.GetImagesForDeployment(ctx, deployment)
			if err != nil {
				return
			}

			matched, err := s.getNetworkPoliciesForDeployment(ctx, deployment)
			if err != nil {
				log.Errorf("failed to fetch network policies for deployment: %s", err.Error())
				return
			}

			violations, err := compiledPolicy.MatchAgainstDeployment(nil, booleanpolicy.EnhancedDeployment{
				Deployment:             deployment,
				Images:                 images,
				NetworkPoliciesApplied: matched,
			})

			if err != nil {
				log.Errorf("failed policy matching: %s", err.Error())
				return
			}

			if len(violations.AlertViolations) == 0 && violations.ProcessViolation == nil {
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
	select {
	case <-allAlertsProcessedSig.Done():
		return &resp, nil
	case <-cancelCtx.Done():
		return nil, cancelCtx.Err()
	}
}

// DryRunPolicy runs a dry run of the policy and determines what deployments would violate it
func (s *serviceImpl) DryRunPolicy(ctx context.Context, request *storage.Policy) (*v1.DryRunResponse, error) {
	if err := s.convertAndValidate(ctx, request); err != nil {
		return nil, err
	}

	return s.predicateBasedDryRunPolicy(ctx, ctx, request)
}

// GetPolicyCategories returns the categories of all policies.
func (s *serviceImpl) GetPolicyCategories(ctx context.Context, _ *v1.Empty) (*v1.PolicyCategoriesResponse, error) {
	categorySet, err := s.getPolicyCategorySet(ctx)
	if err != nil {
		return nil, err
	}

	response := new(v1.PolicyCategoriesResponse)
	response.Categories = categorySet.AsSlice()
	sort.Strings(response.Categories)

	return response, nil
}

func (s *serviceImpl) getPolicyCategorySet(ctx context.Context) (categorySet set.StringSet, err error) {
	policies, err := s.policies.GetAllPolicies(ctx)
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
		s.buildTimePolicies.RemovePolicy(policy.GetId())
	}

	errorList.AddError(s.lifecycleManager.UpsertPolicy(policy))
	return errorList.ToError()
}

func (s *serviceImpl) removeActivePolicy(id string) error {
	errorList := errorhelpers.NewErrorList("error removing policy from detection: ")
	s.buildTimePolicies.RemovePolicy(id)
	errorList.AddError(s.lifecycleManager.RemovePolicy(id))
	return errorList.ToError()
}

func (s *serviceImpl) EnableDisablePolicyNotification(ctx context.Context, request *v1.EnableDisablePolicyNotificationRequest) (*v1.Empty, error) {
	if request.GetPolicyId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Policy ID must be specified")
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
		return errors.Wrap(errox.InvalidArgs, "Notifier IDs must be specified")
	}

	policy, exists, err := s.policies.GetPolicy(ctx, policyID)
	if err != nil {
		return errors.Errorf("failed to retrieve policy: %v", err)
	}
	if !exists {
		return errors.Wrapf(errox.NotFound, "Policy %q not found", policyID)
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
		}
		if notifierSet.Contains(notifierID) {
			continue
		}
		policy.Notifiers = append(policy.Notifiers, notifierID)
	}

	_, err = s.PutPolicy(ctx, policy)
	if err != nil {
		errorList.AddStringf("policy could not be updated with notifier %v", err)
	}

	err = errorList.ToError()
	if err != nil {
		return err
	}
	return nil
}

func (s *serviceImpl) syncPoliciesWithSensors() error {
	policies, err := s.policies.GetAllPolicies(policySyncReadCtx)
	if err != nil {
		return errors.Wrap(err, "error reading policies from store")
	}

	s.connectionManager.PreparePoliciesAndBroadcast(policies)
	return nil
}

func (s *serviceImpl) disablePolicyNotification(ctx context.Context, policyID string, notifierIDs []string) error {
	policy, exists, err := s.policies.GetPolicy(ctx, policyID)
	if err != nil {
		return errors.Errorf("failed to retrieve policy: %v", err)
	}
	if !exists {
		return errors.Wrapf(errox.NotFound, "Policy %q not found", policyID)
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
		return err
	}

	return nil
}

func checkIdentityFromMetadata(ctx context.Context, metadata map[string]interface{}) error {
	identityUID, ok := metadata[identityUIDKey].(string)
	if !ok {
		return errors.New("Invalid job.")
	}

	id, err := authn.IdentityFromContext(ctx)
	if err != nil {
		return err
	}
	if identityUID != id.UID() {
		return errox.NotAuthorized
	}

	return nil
}

func (s *serviceImpl) ExportPolicies(ctx context.Context, request *v1.ExportPoliciesRequest) (*storage.ExportPoliciesResponse, error) {
	// missingIndices and policyErrors should not overlap
	policyList, missingIndices, err := s.policies.GetPolicies(ctx, request.PolicyIds)
	if err != nil {
		return nil, err
	}
	errDetails := &v1.ExportPoliciesErrorList{}
	for _, missingIndex := range missingIndices {
		policyID := request.PolicyIds[missingIndex]
		errDetails.Errors = append(errDetails.Errors, &v1.ExportPolicyError{
			PolicyId: policyID,
			Error: &v1.PolicyError{
				Error: "not found",
			},
		})
		log.Warnf("A policy error occurred for id %s: not found", policyID)

	}
	if len(missingIndices) > 0 {
		statusMsg, err := status.New(codes.InvalidArgument, "Some policies could not be retrieved. Check the error details for a list of policies that could not be found").WithDetails(errDetails)
		if err != nil {
			return nil, utils.ShouldErr(errors.Errorf("unexpected error creating status proto: %v", err))
		}
		return nil, statusMsg.Err()
	}

	for _, policy := range policyList {
		removeInternal(policy)
	}
	return &storage.ExportPoliciesResponse{
		Policies: policyList,
	}, nil
}

func removeInternal(policy *storage.Policy) {
	if policy == nil {
		return
	}
	policy.SORTLifecycleStage = ""
	policy.SORTEnforcement = false
	policy.SORTName = ""
}

func (s *serviceImpl) convertAndValidateForImport(p *storage.Policy) error {
	if err := policyversion.EnsureConvertedToLatest(p); err != nil {
		return err
	}
	if err := s.validator.validateImport(p); err != nil {
		return err
	}

	return nil

}

func (s *serviceImpl) ImportPolicies(ctx context.Context, request *v1.ImportPoliciesRequest) (*v1.ImportPoliciesResponse, error) {
	responses := make([]*v1.ImportPolicyResponse, 0, len(request.Policies))
	allValidationSucceeded := true
	// Validate input policies
	validPolicyList := make([]*storage.Policy, 0, len(request.GetPolicies()))
	for _, policy := range request.GetPolicies() {
		err := s.convertAndValidateForImport(policy)
		if err != nil {
			allValidationSucceeded = false
			responses = append(responses, makeValidationError(policy, err))
			continue
		}
		validPolicyList = append(validPolicyList, policy)
	}

	// Import valid policies
	importResponses, allImportsSucceeded, err := s.policies.ImportPolicies(ctx, validPolicyList, request.GetMetadata().GetOverwrite())
	if err != nil {
		return nil, err
	}

	for _, importResponse := range importResponses {
		if importResponse.GetSucceeded() {
			if err := s.addActivePolicy(importResponse.GetPolicy()); err != nil {
				importResponse.Succeeded = false
				importResponse.Errors = append(importResponse.GetErrors(), &v1.ImportPolicyError{
					Message: errors.Wrap(err, "Policy could not be imported due to").Error(),
					Type:    policies.ErrImportUnknown,
				})
			}
		}
		// Clone here because this may be the same object stored by the DB
		importResponse.Policy = importResponse.GetPolicy().Clone()
		removeInternal(importResponse.Policy)
	}

	if err := s.syncPoliciesWithSensors(); err != nil {
		return nil, err
	}

	responses = append(responses, importResponses...)
	return &v1.ImportPoliciesResponse{
		Responses:    responses,
		AllSucceeded: allValidationSucceeded && allImportsSucceeded,
	}, nil
}

func makeValidationError(policy *storage.Policy, err error) *v1.ImportPolicyResponse {
	return &v1.ImportPolicyResponse{
		Succeeded: false,
		Policy:    policy,
		Errors: []*v1.ImportPolicyError{
			{
				Message: "Invalid policy",
				Type:    policies.ErrImportValidation,
				Metadata: &v1.ImportPolicyError_ValidationError{
					ValidationError: err.Error(),
				},
			},
		},
	}
}

func (s *serviceImpl) PolicyFromSearch(ctx context.Context, request *v1.PolicyFromSearchRequest) (*v1.PolicyFromSearchResponse, error) {
	policy, unconvertableCriteria, hasNestedFields, err := s.parsePolicy(ctx, request.GetSearchParams())
	if err != nil {
		return nil, errors.Wrap(err, "error creating policy from search string")
	}

	response := &v1.PolicyFromSearchResponse{
		Policy:             policy,
		HasNestedFields:    hasNestedFields,
		AlteredSearchTerms: make([]string, 0, len(unconvertableCriteria)),
	}
	for _, fieldName := range unconvertableCriteria {
		response.AlteredSearchTerms = append(response.AlteredSearchTerms, fieldName.String())
	}
	return response, nil
}

func (s *serviceImpl) parsePolicy(ctx context.Context, searchString string) (*storage.Policy, []search.FieldLabel, bool, error) {
	// Handle empty input query case.
	if len(searchString) == 0 {
		return nil, nil, false, errox.InvalidArgs.CausedBy("can not generate a policy from an empty query")
	}
	// Have a filled query, parse it.
	fieldMap, err := getFieldMapFromQueryString(searchString)
	if err != nil {
		return nil, nil, false, err
	}

	policy, unconvertable, err := s.makePolicyFromFieldMap(ctx, fieldMap)
	if err != nil {
		return nil, nil, false, err
	}

	return policy, unconvertable, false, err
}

func getFieldMapFromQueryString(searchString string) (map[search.FieldLabel][]string, error) {
	fieldMap, err := search.ParseFieldMap(searchString)
	if err != nil {
		return nil, err
	}
	for fieldLabel, fieldValues := range fieldMap {
		filteredV := fieldValues[:0]
		for _, value := range fieldValues {
			if value == "" {
				continue
			}
			filteredV = append(filteredV, value)
		}
		fieldMap[fieldLabel] = filteredV
	}
	return fieldMap, nil
}

func (s *serviceImpl) makePolicyFromFieldMap(ctx context.Context, fieldMap map[search.FieldLabel][]string) (*storage.Policy, []search.FieldLabel, error) {
	// Sort the FieldLabels by field value, to ensure consistency of output.
	fieldLabels := make([]search.FieldLabel, 0, len(fieldMap))
	for field := range fieldMap {
		fieldLabels = append(fieldLabels, field)
	}
	sortedFieldLabels := search.SortFieldLabels(fieldLabels)

	var unconvertableFields []search.FieldLabel
	policyGroupMap := make(map[string][]*storage.PolicyGroup)
	for _, field := range sortedFieldLabels {
		if field == search.Cluster || field == search.Namespace || field == search.DeploymentLabel {
			continue
		}
		searchTermPolicyGroup, fieldsDropped, converterExists := booleanpolicy.GetPolicyGroupFromSearchTerms(field, fieldMap[field])
		if !converterExists || searchTermPolicyGroup == nil {
			// Either we can't convert this search term or the translator generated no policy values
			unconvertableFields = append(unconvertableFields, field)
			continue
		}
		if fieldsDropped {
			// Some part of this search term was dropped during conversion but we still ended up with policy values.
			unconvertableFields = append(unconvertableFields, field)
		}
		policyGroupMap[searchTermPolicyGroup.GetFieldName()] = append(policyGroupMap[searchTermPolicyGroup.GetFieldName()], searchTermPolicyGroup)
	}

	scopes, err := s.makeScopes(ctx, fieldMap)
	if err != nil {
		return nil, nil, err
	}

	if len(policyGroupMap) == 0 && len(scopes) == 0 {
		return nil, nil, errors.New("after parsing there were no valid policy groups or scopes")
	}

	policyGroups := flattenPolicyGroupMap(policyGroupMap)

	policy := &storage.Policy{
		PolicyVersion: policyversion.CurrentVersion().String(),
	}
	if len(scopes) > 0 {
		policy.Scope = scopes
	}
	if len(policyGroups) > 0 {
		policy.PolicySections = []*storage.PolicySection{
			{
				PolicyGroups: policyGroups,
			},
		}
	}

	// We have to add and remove a policy name because the BPL validator requires a policy name for these checks
	policy.Name = "Policy from Search"

	for _, group := range policyGroups {
		// Only check for Deployment event fields since audit log fields are not searchable anyways.
		if booleanpolicy.FieldMetadataSingleton().IsDeploymentEventField(group.GetFieldName()) {
			policy.EventSource = storage.EventSource_DEPLOYMENT_EVENT
			break
		}
	}

	if lifecycleStages := s.validator.getAllowedLifecyclesForPolicy(policy); len(lifecycleStages) > 0 {
		policy.LifecycleStages = lifecycleStages
	}

	policy.Name = ""
	return policy, unconvertableFields, nil
}

func (s *serviceImpl) makeScopes(ctx context.Context, fieldMap map[search.FieldLabel][]string) ([]*storage.Scope, error) {
	clusters, clustersOk := fieldMap[search.Cluster]
	namespaces, namespacesOk := fieldMap[search.Namespace]
	if !namespacesOk {
		namespaces = []string{""}
	}
	labels, labelsOk := fieldMap[search.DeploymentLabel]
	if !labelsOk {
		labels = []string{""}
	}
	// If we have none of the above, we have no scopes
	if !clustersOk && !namespacesOk && !labelsOk {
		return nil, nil
	}
	// We need cluster IDs, not cluster names
	clusterIDs, err := s.getClusterIDs(ctx, clusters)
	if err != nil {
		return nil, err
	}
	if len(clusterIDs) == 0 {
		clusterIDs = []string{""}
	}

	// For each combination of label, cluster, and namespace create a Scope
	var scopes []*storage.Scope
	for _, label := range labels {
		labelKey, labelValue := stringutils.Split2(label, "=")
		var labelObject *storage.Scope_Label
		if labelKey != "" || labelValue != "" {
			labelObject = &storage.Scope_Label{
				Key:   labelKey,
				Value: labelValue,
			}
		}
		for _, clusterID := range clusterIDs {
			for _, namespace := range namespaces {
				scopes = append(scopes, &storage.Scope{
					Cluster:   clusterID,
					Namespace: namespace,
					Label:     labelObject,
				})
			}
		}
	}

	return scopes, nil
}

func (s *serviceImpl) getClusterIDs(ctx context.Context, clusterNames []string) ([]string, error) {
	if len(clusterNames) == 0 {
		return nil, nil
	}

	allClusters, err := s.clusters.GetClusters(ctx)
	if err != nil {
		return nil, err
	}

	clusterIDs := set.NewStringSet()
	for _, clusterName := range clusterNames {
		matcher, err := basematchers.ForString(clusterName)
		if err != nil {
			log.Errorf("could not create matcher for %s: %v", clusterName, err)
			continue
		}

		for _, cluster := range allClusters {
			if matcher(cluster.GetName()) {
				clusterIDs.Add(cluster.GetId())
			}
		}
	}

	return clusterIDs.AsSlice(), nil
}

func flattenPolicyGroupMap(policyGroupMap map[string][]*storage.PolicyGroup) []*storage.PolicyGroup {
	policyGroupList := make([]*storage.PolicyGroup, 0, len(policyGroupMap))
	for groupName, singleGroupList := range policyGroupMap {
		if !partialListPolicyGroups.Contains(groupName) {
			// This policy group can't be combined, use whichever one was generated first.  This will be consistent as
			// we sort the field names.  If there is more than one the later groups will be dropped.
			policyGroupList = append(policyGroupList, singleGroupList[0])
			continue
		}

		var policyValueLists []*storage.PolicyValue
		for _, policyGroup := range singleGroupList {
			// For now we don't care which search term a policy value came from because no two search terms can
			// generate a value for the same list index, and no search term can generate a value for more than one list
			// index.  Therefore it is safe to flatten the values and naively generate all possible combinations.
			policyValueLists = append(policyValueLists, policyGroup.GetValues()...)
		}
		combinedValues := combinePolicyValues(policyValueLists)

		policyGroupList = append(policyGroupList, &storage.PolicyGroup{
			FieldName: groupName,
			Values:    combinedValues,
		})
	}
	return policyGroupList
}

func combinePolicyValues(policyValues []*storage.PolicyValue) []*storage.PolicyValue {
	splitValueStringLists := make([][]string, 0, len(policyValues))
	for _, policyValue := range policyValues {
		splitValueStringLists = append(splitValueStringLists, strings.Split(policyValue.GetValue(), "="))
	}

	requiredLength := len(splitValueStringLists[0])
	partsToCombine := make([][]string, requiredLength)
	for _, splitValueList := range splitValueStringLists {
		for i, section := range splitValueList {
			if section != "" {
				partsToCombine[i] = append(partsToCombine[i], section)
			}
		}
	}

	for i, partsList := range partsToCombine {
		if len(partsList) == 0 {
			// If part of a list is empty we still want to generate the combinations of the other parts, leaving this part empty
			partsToCombine[i] = []string{""}
		}
	}

	combinations := combineStrings(partsToCombine)
	values := make([]*storage.PolicyValue, len(combinations))
	for i, combination := range combinations {
		values[i] = &storage.PolicyValue{
			Value: combination,
		}
	}

	return values
}

func combineStrings(toCombine [][]string) []string {
	indices := make([]int, len(toCombine))
	var combinations []string

	maxIterations := 1
	for _, category := range toCombine {
		maxIterations *= len(category)
	}
	for i := 0; i < maxIterations; i++ {
		combination := ""
		for i := 0; i < len(indices); i++ {
			if i > 0 {
				combination = combination + "="
			}
			combination = combination + toCombine[i][indices[i]]
		}
		combinations = append(combinations, combination)

		for index := len(indices) - 1; index >= 0; index-- {
			indices[index]++
			if indices[index] < len(toCombine[index]) {
				break
			}
			if index == 0 {
				return combinations
			}
			indices[index] = 0
		}
	}
	return nil
}
