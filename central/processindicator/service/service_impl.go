package service

import (
	"context"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Deployment)): {
			"/v1.ProcessService/GetProcessesByDeployment",
			"/v1.ProcessService/GetGroupedProcessByDeployment",
		},
	})
)

type serviceImpl struct {
	processIndicators processIndicatorStore.DataStore
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

// GetDeployment returns the deployment with given id.
func (s *serviceImpl) GetProcessesByDeployment(_ context.Context, req *v1.GetProcessesByDeploymentRequest) (*v1.GetProcessesResponse, error) {
	if req.GetDeploymentId() == "" {
		return nil, status.Error(codes.Internal, "Deployment ID must be specified when retrieving processes")
	}
	indicators, err := s.processIndicators.SearchRawProcessIndicators(
		search.NewQueryBuilder().
			AddStrings(search.DeploymentID, req.GetDeploymentId()).
			ProtoQuery(),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.GetProcessesResponse{
		Processes: indicators,
	}, nil
}

func sortIndicators(indicators []*v1.ProcessIndicator) {
	sort.SliceStable(indicators, func(i, j int) bool {
		i1, i2 := indicators[i], indicators[j]
		return protoconv.CompareProtoTimestamps(i1.GetSignal().GetTime(), i2.GetSignal().GetTime()) == -1
	})
}

func indicatorsToGroupedResponses(indicators []*v1.ProcessIndicator) []*v1.ProcessNameGroup {
	processGroups := make(map[string]map[string][]*v1.ProcessIndicator)
	processNameToContainers := make(map[string]map[string]struct{})
	for _, i := range indicators {
		fullProcessName := i.GetSignal().GetExecFilePath()
		nameMap, ok := processGroups[fullProcessName]
		if !ok {
			nameMap = make(map[string][]*v1.ProcessIndicator)
			processGroups[fullProcessName] = nameMap
			processNameToContainers[fullProcessName] = make(map[string]struct{})
		}
		nameMap[i.GetSignal().GetArgs()] = append(nameMap[i.GetSignal().GetArgs()], i)
		processNameToContainers[fullProcessName][i.GetSignal().GetContainerId()] = struct{}{}
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
			TimesExecuted: uint32(len(processNameToContainers[name])),
		})
	}
	sort.SliceStable(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	return groups
}

func (s *serviceImpl) GetGroupedProcessByDeployment(_ context.Context, req *v1.GetProcessesByDeploymentRequest) (*v1.GetGroupedProcessesResponse, error) {
	if req.GetDeploymentId() == "" {
		return nil, status.Error(codes.Internal, "Deployment ID must be specified when retrieving processes")
	}
	indicators, err := s.processIndicators.SearchRawProcessIndicators(
		search.NewQueryBuilder().
			AddStrings(search.DeploymentID, req.GetDeploymentId()).
			ProtoQuery(),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.GetGroupedProcessesResponse{
		Groups: indicatorsToGroupedResponses(indicators),
	}, nil
}
