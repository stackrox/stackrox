package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	"github.com/stackrox/rox/central/processbaseline/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.DeploymentExtension)): {
			v1.ProcessBaselineService_GetProcessBaseline_FullMethodName,
		},
		user.With(permissions.Modify(resources.DeploymentExtension)): {
			v1.ProcessBaselineService_UpdateProcessBaselines_FullMethodName,
			v1.ProcessBaselineService_LockProcessBaselines_FullMethodName,
			v1.ProcessBaselineService_BulkLockProcessBaselines_FullMethodName,
			v1.ProcessBaselineService_BulkUnlockProcessBaselines_FullMethodName,
			v1.ProcessBaselineService_DeleteProcessBaselines_FullMethodName,
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedProcessBaselineServiceServer

	dataStore        datastore.DataStore
	reprocessor      reprocessor.Loop
	deployments      deploymentStore.DataStore
	lifecycleManager lifecycle.Manager
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterProcessBaselineServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterProcessBaselineServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func validateKeyNotEmpty(key *storage.ProcessBaselineKey) error {
	if stringutils.AtLeastOneEmpty(
		key.GetDeploymentId(),
		key.GetContainerName(),
		key.GetClusterId(),
		key.GetNamespace(),
	) {
		return errors.New("invalid key: must specify both deployment id and container name")
	}
	return nil
}

func (s *serviceImpl) GetProcessBaseline(ctx context.Context, request *v1.GetProcessBaselineRequest) (*storage.ProcessBaseline, error) {
	if err := validateKeyNotEmpty(request.GetKey()); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	baseline, exists, err := s.dataStore.GetProcessBaseline(ctx, request.GetKey())
	if err != nil {
		return nil, err
	}
	if !exists {
		// Make sure the deployment still exists before we try to build a baseline.
		_, deploymentExists, err := s.deployments.GetDeployment(ctx, request.GetKey().GetDeploymentId())
		if err != nil {
			return nil, err
		}
		if !deploymentExists {
			return nil, errors.Wrapf(errox.NotFound, "deployment with id '%q' does not exist", request.GetKey().GetDeploymentId())
		}

		// Build an unlocked baseline
		baseline, err = s.dataStore.CreateUnlockedProcessBaseline(ctx, request.GetKey())
		if err != nil {
			return nil, err
		}
		if baseline == nil {
			return nil, errors.Wrapf(errox.NotFound, "No process baseline with key %+v found", request.GetKey())
		}
	}
	return baseline, nil
}

func bulkUpdate(keys []*storage.ProcessBaselineKey, parallelFunc func(*storage.ProcessBaselineKey) (*storage.ProcessBaseline, error)) *v1.UpdateProcessBaselinesResponse {
	chanLen := len(keys)
	baselines := make([]*storage.ProcessBaseline, 0, chanLen)
	errorList := make([]*v1.ProcessBaselineUpdateError, 0, chanLen)
	for _, key := range keys {
		baseline, err := parallelFunc(key)
		if err != nil {
			errorList = append(errorList, &v1.ProcessBaselineUpdateError{Error: err.Error(), Key: key})
		} else {
			baselines = append(baselines, baseline)
		}
	}
	response := &v1.UpdateProcessBaselinesResponse{
		Baselines: baselines,
		Errors:    errorList,
	}
	return response
}

func (s *serviceImpl) reprocessDeploymentRisks(keys []*storage.ProcessBaselineKey) {
	deploymentIDs := set.NewStringSet()
	for _, key := range keys {
		deploymentIDs.Add(key.GetDeploymentId())
	}
	s.reprocessor.ReprocessRiskForDeployments(deploymentIDs.AsSlice()...)
}

func (s *serviceImpl) UpdateProcessBaselines(ctx context.Context, request *v1.UpdateProcessBaselinesRequest) (*v1.UpdateProcessBaselinesResponse, error) {
	// Make sure only the baselines that were updated are reprocessed afterwards.
	var resp *v1.UpdateProcessBaselinesResponse
	defer s.reprocessUpdatedBaselines(&resp)

	updateFunc := func(key *storage.ProcessBaselineKey) (*storage.ProcessBaseline, error) {
		return s.dataStore.UpdateProcessBaselineElements(ctx, key, request.GetAddElements(), request.GetRemoveElements(), false)
	}
	resp = bulkUpdate(request.GetKeys(), updateFunc)

	for _, w := range resp.GetBaselines() {
		err := s.lifecycleManager.SendBaselineToSensor(w)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

func (s *serviceImpl) LockProcessBaselines(ctx context.Context, request *v1.LockProcessBaselinesRequest) (*v1.UpdateProcessBaselinesResponse, error) {
	var resp *v1.UpdateProcessBaselinesResponse
	defer s.reprocessUpdatedBaselines(&resp)

	updateFunc := func(key *storage.ProcessBaselineKey) (*storage.ProcessBaseline, error) {
		return s.dataStore.UserLockProcessBaseline(ctx, key, request.GetLocked())
	}
	resp = bulkUpdate(request.GetKeys(), updateFunc)
	for _, w := range resp.GetBaselines() {
		err := s.lifecycleManager.SendBaselineToSensor(w)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

func (s *serviceImpl) getKeys(ctx context.Context, clusterId string, namespaces []string) ([]*storage.ProcessBaselineKey, error) {
	queryBuilder := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterId)

	if len(namespaces) > 0 {
		queryBuilder = queryBuilder.AddExactMatches(search.Namespace, namespaces...)
	}

	query := queryBuilder.ProtoQuery()

	baselines, err := s.dataStore.SearchRawProcessBaselines(ctx, query)

	if err != nil {
		return nil, err
	}

	keys := make([]*storage.ProcessBaselineKey, len(baselines))

	for i := range baselines {
		keys[i] = baselines[i].GetKey()
	}

	return keys, nil
}

func (s *serviceImpl) bulkLockOrUnlockProcessBaselines(ctx context.Context, request *v1.BulkProcessBaselinesRequest, methodName string, lock bool) (*v1.BulkUpdateProcessBaselinesResponse, error) {
	var resp *v1.UpdateProcessBaselinesResponse
	defer s.reprocessUpdatedBaselines(&resp)

	clusterId := request.GetClusterId()

	if clusterId == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "no cluster ID specified")
	}

	keys, err := s.getKeys(ctx, clusterId, request.GetNamespaces())
	if err != nil {
		return nil, err
	}

	updateFunc := func(key *storage.ProcessBaselineKey) (*storage.ProcessBaseline, error) {
		return s.dataStore.UserLockProcessBaseline(ctx, key, lock)
	}

	resp = bulkUpdate(keys, updateFunc)

	for _, w := range resp.GetBaselines() {
		err := s.lifecycleManager.SendBaselineToSensor(w)
		if err != nil {
			return nil, err
		}
	}

	success := &v1.BulkUpdateProcessBaselinesResponse{
		Success: true,
	}

	return success, nil
}

func (s *serviceImpl) BulkLockProcessBaselines(ctx context.Context, request *v1.BulkProcessBaselinesRequest) (*v1.BulkUpdateProcessBaselinesResponse, error) {
	methodName := "BulkLockProcessBaselines"
	return s.bulkLockOrUnlockProcessBaselines(ctx, request, methodName, true)
}

func (s *serviceImpl) BulkUnlockProcessBaselines(ctx context.Context, request *v1.BulkProcessBaselinesRequest) (*v1.BulkUpdateProcessBaselinesResponse, error) {
	methodName := "BulkUnlockProcessBaselines"
	return s.bulkLockOrUnlockProcessBaselines(ctx, request, methodName, false)
}

func (s *serviceImpl) DeleteProcessBaselines(ctx context.Context, request *v1.DeleteProcessBaselinesRequest) (*v1.DeleteProcessBaselinesResponse, error) {
	if request.GetQuery() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "query string must be nonempty")
	}

	q, err := search.ParseQuery(request.GetQuery())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	results, err := s.dataStore.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	response := &v1.DeleteProcessBaselinesResponse{
		DryRun:     !request.GetConfirm(),
		NumDeleted: int32(len(results)),
	}

	if !request.GetConfirm() {
		return response, nil
	}

	toClear := make([]string, 0, len(results))
	var toDelete []string
	// go through list of IDs returned from the search results; clear the baseline and remove deployments from observation.
	for _, r := range results {
		key, _ := datastore.IDToKey(r.ID)

		// make sure the deployment still exists, if not that process will take care fo the baseline
		_, exists, err := s.deployments.GetDeployment(ctx, key.GetDeploymentId())

		if exists && err == nil {
			toClear = append(toClear, r.ID)
		}
		if !exists {
			toDelete = append(toDelete, r.ID)
		}

		// Remove the deployment from observation so everything forward is processed to prevent us from re-processing
		// indicators from the past.  This operation is a no-op if the deployment has been deleted.
		s.lifecycleManager.RemoveDeploymentFromObservation(key.GetDeploymentId())
	}

	// Clear the contents of the baseline
	if len(toClear) > 0 {
		err = s.dataStore.ClearProcessBaselines(ctx, toClear)
		// ClearProcessBaselines returns an error if the baseline does not exist.
		if err != nil {
			return nil, err
		}
	}

	// We have a key whose deployment does not exist, so we probably have orphaned data, we should
	// clean it up while we are here.
	if len(toDelete) > 0 {
		err = s.dataStore.RemoveProcessBaselinesByIDs(ctx, toDelete)

		if err != nil {
			return nil, err
		}
	}

	return response, nil
}

func (s *serviceImpl) reprocessUpdatedBaselines(resp **v1.UpdateProcessBaselinesResponse) {
	var keys []*storage.ProcessBaselineKey
	for _, pb := range (*resp).GetBaselines() {
		keys = append(keys, pb.GetKey())
	}
	s.reprocessDeploymentRisks(keys)
}
