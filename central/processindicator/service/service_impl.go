package service

import (
	"context"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/processbaseline"
	baselineStore "github.com/stackrox/rox/central/processbaseline/datastore"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	processBaselinePkg "github.com/stackrox/rox/pkg/processbaseline"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.DeploymentExtension)): {
			"/v1.ProcessService/CountProcesses",
			"/v1.ProcessService/GetProcessesByDeployment",
			"/v1.ProcessService/GetGroupedProcessByDeployment",
			"/v1.ProcessService/GetGroupedProcessByDeploymentAndContainer",
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedProcessServiceServer

	processIndicators processIndicatorStore.DataStore
	deployments       deploymentStore.DataStore
	baselines         baselineStore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterProcessServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterProcessServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// CountProcesses counts the number of processes that match the input query.
func (s *serviceImpl) CountProcesses(ctx context.Context, request *v1.RawQuery) (*v1.CountProcessesResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	numProcesses, err := s.processIndicators.Count(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}
	return &v1.CountProcessesResponse{Count: int32(numProcesses)}, nil
}

// GetDeployment returns the deployment with given id.
func (s *serviceImpl) GetProcessesByDeployment(ctx context.Context, req *v1.GetProcessesByDeploymentRequest) (*v1.GetProcessesResponse, error) {
	if req.GetDeploymentId() == "" {
		return nil, errors.New("Deployment ID must be specified when retrieving processes")
	}
	_, exists, err := s.deployments.GetDeployment(ctx, req.GetDeploymentId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "deployment with id '%s' does not exist", req.GetDeploymentId())
	}
	indicators, err := s.processIndicators.SearchRawProcessIndicators(ctx,
		search.NewQueryBuilder().
			AddExactMatches(search.DeploymentID, req.GetDeploymentId()).
			ProtoQuery(),
	)
	if err != nil {
		return nil, err
	}
	return &v1.GetProcessesResponse{
		Processes: indicators,
	}, nil
}

func sortIndicators(indicators []*storage.ProcessIndicator) {
	sort.SliceStable(indicators, func(i, j int) bool {
		return indicators[i].GetSignal().GetTime().Compare(indicators[j].GetSignal().GetTime()) == -1
	})
}

func (s *serviceImpl) setSuspicious(ctx context.Context, groupedIndicators []*v1.ProcessNameAndContainerNameGroup, deploymentID string) error {
	baselines := make(map[string]*set.StringSet)
	for _, group := range groupedIndicators {
		elementSet, ok := baselines[group.GetContainerName()]
		if !ok {
			var err error
			elementSet, err = s.getElementSet(ctx, deploymentID, group.GetContainerName())
			if err != nil {
				return err
			}
			baselines[group.GetContainerName()] = elementSet
		}
		group.Suspicious = elementSet != nil && !elementSet.Contains(group.Name)
	}
	return nil
}

func (s *serviceImpl) getElementSet(ctx context.Context, deploymentID string, containerName string) (*set.StringSet, error) {
	deployment, exists, err := s.deployments.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "deployment with id '%s' does not exist", deploymentID)
	}

	key := &storage.ProcessBaselineKey{
		ClusterId:     deployment.GetClusterId(),
		Namespace:     deployment.GetNamespace(),
		DeploymentId:  deploymentID,
		ContainerName: containerName,
	}
	baseline, exists, err := s.baselines.GetProcessBaseline(ctx, key)
	if !exists || err != nil {
		return nil, err
	}
	return processbaseline.Processes(baseline, processbaseline.RoxOrUserLocked), nil
}

// indicatorsToGroupedResponsesWithContainer rearranges process indicator storage items into API process name/container
// name group items.
func indicatorsToGroupedResponsesWithContainer(indicators []*storage.ProcessIndicator) []*v1.ProcessNameAndContainerNameGroup {
	type groupKey struct {
		processName   string
		containerName string
	}
	processGroups := make(map[groupKey]map[string][]*storage.ProcessIndicator)
	processNameToContainers := make(map[groupKey]*set.StringSet)
	for _, i := range indicators {
		name := processBaselinePkg.BaselineItemFromProcess(i)
		if name == "" {
			continue
		}
		containerName := i.ContainerName
		groupKey := groupKey{name, containerName}
		groupMap, ok := processGroups[groupKey]
		if !ok {
			groupMap = make(map[string][]*storage.ProcessIndicator)
			processGroups[groupKey] = groupMap
			processNameToContainers[groupKey] = &set.StringSet{}
		}
		groupMap[i.GetSignal().GetArgs()] = append(groupMap[i.GetSignal().GetArgs()], i)
		processNameToContainers[groupKey].Add(i.GetSignal().GetContainerId())
	}

	groups := make([]*v1.ProcessNameAndContainerNameGroup, 0, len(processGroups))
	for groupKey, groupMap := range processGroups {
		processGroups := make([]*v1.ProcessGroup, 0, len(groupMap))
		for args, indicators := range groupMap {
			sortIndicators(indicators)
			processGroups = append(processGroups, &v1.ProcessGroup{Args: args, Signals: indicators})
		}
		sort.SliceStable(processGroups, func(i, j int) bool { return processGroups[i].GetArgs() < processGroups[j].GetArgs() })
		groups = append(groups, &v1.ProcessNameAndContainerNameGroup{
			Name:          groupKey.processName,
			ContainerName: groupKey.containerName,
			Groups:        processGroups,
			TimesExecuted: uint32(processNameToContainers[groupKey].Cardinality()),
			Suspicious:    false,
		})
	}
	sort.SliceStable(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	return groups
}

func (s *serviceImpl) GetGroupedProcessByDeploymentAndContainer(ctx context.Context, req *v1.GetProcessesByDeploymentRequest) (*v1.GetGroupedProcessesWithContainerResponse, error) {
	indicators, err := s.validateGetProcesses(ctx, req)
	if err != nil {
		return nil, err
	}

	groupedIndicators := indicatorsToGroupedResponsesWithContainer(indicators)
	err = s.setSuspicious(ctx, groupedIndicators, req.GetDeploymentId())
	if err != nil {
		return nil, err
	}
	return &v1.GetGroupedProcessesWithContainerResponse{Groups: groupedIndicators}, nil
}

// IndicatorsToGroupedResponses rearranges process indicator storage items into API process name group items.
func IndicatorsToGroupedResponses(indicators []*storage.ProcessIndicator) []*v1.ProcessNameGroup {
	processGroups := make(map[string]map[string][]*storage.ProcessIndicator)
	processNameToContainers := make(map[string]*set.StringSet)
	for _, i := range indicators {
		fullProcessName := i.GetSignal().GetExecFilePath()
		nameMap, ok := processGroups[fullProcessName]
		if !ok {
			nameMap = make(map[string][]*storage.ProcessIndicator)
			processGroups[fullProcessName] = nameMap
			processNameToContainers[fullProcessName] = &set.StringSet{}
		}
		nameMap[i.GetSignal().GetArgs()] = append(nameMap[i.GetSignal().GetArgs()], i)
		processNameToContainers[fullProcessName].Add(i.GetSignal().GetContainerId())
	}

	groups := make([]*v1.ProcessNameGroup, 0, len(processGroups))
	for name, nameMap := range processGroups {
		processGroups := make([]*v1.ProcessGroup, 0, len(nameMap))
		for args, indicators := range nameMap {
			sortIndicators(indicators)
			processGroups = append(processGroups, &v1.ProcessGroup{Args: args, Signals: indicators})
		}
		sort.SliceStable(processGroups, func(i, j int) bool { return processGroups[i].GetArgs() < processGroups[j].GetArgs() })
		groups = append(groups, &v1.ProcessNameGroup{
			Name:          name,
			Groups:        processGroups,
			TimesExecuted: uint32(processNameToContainers[name].Cardinality()),
		})
	}
	sort.SliceStable(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	return groups
}

func (s *serviceImpl) GetGroupedProcessByDeployment(ctx context.Context, req *v1.GetProcessesByDeploymentRequest) (*v1.GetGroupedProcessesResponse, error) {
	indicators, err := s.validateGetProcesses(ctx, req)
	if err != nil {
		return nil, err
	}

	return &v1.GetGroupedProcessesResponse{
		Groups: IndicatorsToGroupedResponses(indicators),
	}, nil
}

func (s *serviceImpl) validateGetProcesses(ctx context.Context, req *v1.GetProcessesByDeploymentRequest) ([]*storage.ProcessIndicator, error) {
	if req.GetDeploymentId() == "" {
		return nil, errors.New("Deployment ID must be specified when retrieving processes")
	}
	indicators, err := s.processIndicators.SearchRawProcessIndicators(ctx,
		search.NewQueryBuilder().
			AddExactMatches(search.DeploymentID, req.GetDeploymentId()).
			ProtoQuery(),
	)
	if err != nil {
		return nil, err
	}

	return indicators, nil
}
