package service

import (
	"context"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/compliance/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Compliance)): {
			"/v1.ComplianceManagementService/GetRecentRuns",
			"/v1.ComplianceManagementService/GetRunStatuses",
		},
		user.With(permissions.Modify(resources.Compliance)): {
			"/v1.ComplianceManagementService/TriggerRun",
			"/v1.ComplianceManagementService/TriggerRuns",
		},
	})
)

type service struct {
	v1.UnimplementedComplianceManagementServiceServer

	manager manager.ComplianceManager
}

func newService(manager manager.ComplianceManager) *service {
	return &service{
		manager: manager,
	}
}

func (s *service) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterComplianceManagementServiceServer(server, s)
}

func (s *service) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterComplianceManagementServiceHandler(ctx, mux, conn)
}

func (s *service) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *service) GetRecentRuns(ctx context.Context, req *v1.GetRecentComplianceRunsRequest) (*v1.GetRecentComplianceRunsResponse, error) {
	runs, err := s.manager.GetRecentRuns(ctx, req)
	if err != nil {
		return nil, err
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartTime.Compare(runs[j].StartTime) < 0
	})

	return &v1.GetRecentComplianceRunsResponse{
		ComplianceRuns: runs,
	}, nil
}

func (s *service) TriggerRuns(ctx context.Context, req *v1.TriggerComplianceRunsRequest) (*v1.TriggerComplianceRunsResponse, error) {
	expanded, err := s.manager.ExpandSelection(ctx, req.GetSelection().GetClusterId(), req.GetSelection().GetStandardId())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "could not expand cluster/standard selection: %v", err)
	}

	runs, err := s.manager.TriggerRuns(ctx, expanded...)
	if err != nil {
		return nil, err
	}
	return &v1.TriggerComplianceRunsResponse{
		StartedRuns: runs,
	}, nil
}

func (s *service) GetRunStatuses(ctx context.Context, req *v1.GetComplianceRunStatusesRequest) (*v1.GetComplianceRunStatusesResponse, error) {
	if req.GetLatest() && len(req.GetRunIds()) != 0 {
		return nil, errox.InvalidArgs.New("both latest and run ids cannot be specified")
	}

	if req.GetLatest() {
		runs, err := s.manager.GetLatestRunStatuses(ctx)
		if err != nil {
			return nil, err
		}
		return &v1.GetComplianceRunStatusesResponse{
			Runs: runs,
		}, nil
	}

	runs, err := s.manager.GetRunStatuses(ctx, req.GetRunIds()...)
	if err != nil {
		return nil, err
	}

	allRunIds := set.NewStringSet(req.GetRunIds()...)
	for _, run := range runs {
		allRunIds.Remove(run.GetId())
	}
	return &v1.GetComplianceRunStatusesResponse{
		InvalidRunIds: allRunIds.AsSlice(),
		Runs:          runs,
	}, nil
}
