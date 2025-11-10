package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/risk/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.Modify(resources.DeploymentExtension)): {
			"/v1.RiskService/ChangeDeploymentRiskPosition",
			"/v1.RiskService/ResetDeploymentRisk",
			"/v1.RiskService/ResetAllDeploymentRisks",
		},
	})
)

// serviceImpl provides APIs for risk ranking adjustments.
type serviceImpl struct {
	v1.UnimplementedRiskServiceServer

	manager manager.Manager
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterRiskServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterRiskServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the authorization for this service.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// ChangeDeploymentRiskPosition adjusts a deployment's risk ranking position.
func (s *serviceImpl) ChangeDeploymentRiskPosition(ctx context.Context, req *v1.RiskPositionChangeRequest) (*v1.RiskAdjustmentResponse, error) {
	if req.GetDeploymentId() == "" {
		return nil, errors.New("deployment_id is required")
	}

	// At least one neighbor must be specified
	if req.GetAboveDeploymentId() == "" && req.GetBelowDeploymentId() == "" {
		return nil, errors.New("at least one neighbor deployment must be specified")
	}

	deployment, err := s.manager.ChangeDeploymentRiskPosition(ctx, req.GetDeploymentId(), req.GetAboveDeploymentId(), req.GetBelowDeploymentId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to reposition deployment %s", req.GetDeploymentId())
	}

	message := "Deployment repositioned in risk ranking"
	return buildAdjustmentResponse(deployment, message), nil
}

// ResetDeploymentRisk removes user ranking adjustments.
func (s *serviceImpl) ResetDeploymentRisk(ctx context.Context, req *v1.RiskAdjustmentRequest) (*v1.RiskAdjustmentResponse, error) {
	if req.GetDeploymentId() == "" {
		return nil, errors.New("deployment_id is required")
	}

	deployment, err := s.manager.ResetDeploymentRisk(ctx, req.GetDeploymentId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to reset deployment %s", req.GetDeploymentId())
	}

	return buildAdjustmentResponse(deployment, "Deployment risk reset to original ML score"), nil
}

// ResetAllDeploymentRisks removes all user ranking adjustments.
func (s *serviceImpl) ResetAllDeploymentRisks(ctx context.Context, req *v1.ResetAllRisksRequest) (*v1.ResetAllRisksResponse, error) {
	count, err := s.manager.ResetAllDeploymentRisks(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to reset all deployment risks")
	}

	message := "All deployment risk adjustments have been reset to original ML scores"
	if count == 0 {
		message = "No deployments had risk adjustments to reset"
	}

	return &v1.ResetAllRisksResponse{
		Count:   int32(count),
		Message: message,
	}, nil
}

// buildAdjustmentResponse creates the response message from a deployment object.
func buildAdjustmentResponse(deployment *storage.Deployment, message string) *v1.RiskAdjustmentResponse {
	// Original score is the ML-calculated risk score
	originalScore := deployment.GetRiskScore()

	// Effective score is either the user-adjusted score or the ML score
	effectiveScore := originalScore
	if adj := deployment.GetUserRankingAdjustment(); adj != nil {
		effectiveScore = adj.GetEffectiveRiskScore()
	}

	return &v1.RiskAdjustmentResponse{
		Deployment:     deployment,
		OriginalScore:  originalScore,
		EffectiveScore: effectiveScore,
		Message:        message,
	}
}
