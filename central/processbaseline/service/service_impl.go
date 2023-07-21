package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	"github.com/stackrox/rox/central/processbaseline/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
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
			"/v1.ProcessBaselineService/GetProcessBaseline",
		},
		user.With(permissions.Modify(resources.DeploymentExtension)): {
			"/v1.ProcessBaselineService/UpdateProcessBaselines",
			"/v1.ProcessBaselineService/LockProcessBaselines",
			"/v1.ProcessBaselineService/DeleteProcessBaselines",
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedProcessBaselineServiceServer

	dataStore         datastore.DataStore
	reprocessor       reprocessor.Loop
	connectionManager connection.Manager
	deployments       deploymentStore.DataStore
	lifecycleManager  lifecycle.Manager
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

func (s *serviceImpl) sendBaselineToSensor(pw *storage.ProcessBaseline) {
	err := s.connectionManager.SendMessage(pw.GetKey().GetClusterId(), &central.MsgToSensor{
		Msg: &central.MsgToSensor_BaselineSync{
			BaselineSync: &central.BaselineSync{
				Baselines: []*storage.ProcessBaseline{pw},
			}},
	})
	if err != nil {
		log.Errorf("Error sending process baseline to cluster %q: %v", pw.GetKey().GetClusterId(), err)
	}
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
		s.sendBaselineToSensor(w)
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
		s.sendBaselineToSensor(w)
	}
	return resp, nil
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

	toClear := make([]string, len(results))
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
