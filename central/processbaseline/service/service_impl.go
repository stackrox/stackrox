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
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.DeploymentExtension)): {
			v1.ProcessBaselineService_GetProcessBaseline_FullMethodName,
			v1.ProcessBaselineService_GetProcessBaselineBulk_FullMethodName,
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

func (s *serviceImpl) getProcessBaseline(ctx context.Context, key *storage.ProcessBaselineKey) (*storage.ProcessBaseline, error) {
	if err := validateKeyNotEmpty(key); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	baseline, exists, err := s.dataStore.GetProcessBaseline(ctx, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		// Make sure the deployment still exists before we try to build a baseline.
		_, deploymentExists, err := s.deployments.GetDeployment(ctx, key.GetDeploymentId())
		if err != nil {
			return nil, err
		}
		if !deploymentExists {
			return nil, errors.Wrapf(errox.NotFound, "deployment with id '%q' does not exist", key.GetDeploymentId())
		}

		// Build an unlocked baseline
		baseline, err = s.dataStore.CreateUnlockedProcessBaseline(ctx, key)
		if err != nil {
			return nil, err
		}
		if baseline == nil {
			return nil, errors.Wrapf(errox.NotFound, "No process baseline with key %+v found", key)
		}
	}

	return baseline, nil
}

func (s *serviceImpl) GetProcessBaseline(ctx context.Context, request *v1.GetProcessBaselineRequest) (*storage.ProcessBaseline, error) {
	return s.getProcessBaseline(ctx, request.GetKey())
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

func (s *serviceImpl) bulkLockOrUnlockProcessBaselines(ctx context.Context, request *v1.BulkProcessBaselinesRequest, lock bool) (*v1.BulkUpdateProcessBaselinesResponse, error) {
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
	return s.bulkLockOrUnlockProcessBaselines(ctx, request, true)
}

func (s *serviceImpl) BulkUnlockProcessBaselines(ctx context.Context, request *v1.BulkProcessBaselinesRequest) (*v1.BulkUpdateProcessBaselinesResponse, error) {
	return s.bulkLockOrUnlockProcessBaselines(ctx, request, false)
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

func (s *serviceImpl) GetProcessBaselineBulk(ctx context.Context, request *v1.GetProcessBaselinesBulkRequest) (*v1.GetProcessBaselinesBulkResponse, error) {
	query := request.GetQuery()
	if query == nil {
		return nil, errors.Wrap(errox.InvalidArgs, "Query must not be nil")
	}

	clusters := query.GetClusterIds()
	if len(clusters) == 0 {
		return nil, errors.Wrap(errox.InvalidArgs, "Clusters list cannot be empty. Set the list of clusters to *, if you want process baselines in all clusters")
	}

	// First we search for matching deployments and containers. Since process baselines are created lazily not process baselines
	// that we are interested exist and we may need to create them. The information that we need is in the deployments datastore
	// and might not be in the process baselines datastore. Also some fields that we are interested in such as image names are
	// availabe in the deployments datastore and not in process baselines datastore.
	deploymentQueryBuilder := search.NewQueryBuilder()

	if !(len(clusters) == 1 && clusters[0] == "*") {
		deploymentQueryBuilder = deploymentQueryBuilder.AddExactMatches(search.ClusterID, query.GetClusterIds()...)
	}
	if len(query.GetNamespaces()) > 0 {
		deploymentQueryBuilder = deploymentQueryBuilder.AddExactMatches(search.Namespace, query.GetNamespaces()...)
	}
	if len(query.GetDeploymentNames()) > 0 {
		deploymentQueryBuilder = deploymentQueryBuilder.AddExactMatches(search.DeploymentName, query.GetDeploymentNames()...)
	}
	if len(query.GetDeploymentIds()) > 0 {
		deploymentQueryBuilder = deploymentQueryBuilder.AddExactMatches(search.DeploymentID, query.GetDeploymentIds()...)
	}
	if len(query.GetImages()) > 0 {
		deploymentQueryBuilder = deploymentQueryBuilder.AddExactMatches(search.ImageName, query.GetImages()...)
	}

	if deploymentQueryBuilder.Query() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "At least one parameter must not be empty or a wild card, not counting container name")
	}

	deployments, err := s.deployments.SearchRawDeployments(ctx, deploymentQueryBuilder.ProtoQuery())
	if err != nil {
		return nil, err
	}

	// Get the process baseline keys from the deployments and filter for container name
	imageSet := set.NewStringSet(query.GetImages()...)
	containerSet := set.NewStringSet(query.GetContainerNames()...)
	baselineKeys := make([]*storage.ProcessBaselineKey, 0, len(deployments))
	for _, deployment := range deployments {
		for _, container := range deployment.GetContainers() {
			imageName := container.GetImage().GetName().GetFullName()
			containerName := container.GetName()
			// We need to check that the container has the correct image and container name. Searching for an image
			// will return all deployments that have that image, including containers that don't have that image.
			// Containers need to be filtered for the container name, since that is not searchable.
			if (imageSet.Contains(imageName) || len(imageSet) == 0) && (containerSet.Contains(containerName) || len(containerSet) == 0) {
				baselineKey := &storage.ProcessBaselineKey{
					DeploymentId:  deployment.GetId(),
					ContainerName: containerName,
					ClusterId:     deployment.GetClusterId(),
					Namespace:     deployment.GetNamespace(),
				}
				baselineKeys = append(baselineKeys, baselineKey)
			}
		}
	}

	totalCount := int32(len(baselineKeys))

	page := request.GetPagination()
	if page != nil {
		baselineKeys = paginated.PaginateSlice(int(page.GetOffset()), int(page.GetLimit()), baselineKeys)
	}

	baselines := make([]*storage.ProcessBaseline, 0, len(baselineKeys))
	for _, baselineKey := range baselineKeys {
		baseline, err := s.getProcessBaseline(ctx, baselineKey)
		if err == nil {
			baselines = append(baselines, baseline)
		} else {
			log.Errorf("Unable to get process baseline from process baseline key %+v: %v", baselineKey, err)
		}
	}

	response := &v1.GetProcessBaselinesBulkResponse{
		Baselines:  baselines,
		TotalCount: totalCount,
	}

	return response, nil
}
