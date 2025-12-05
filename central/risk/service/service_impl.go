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

	if req.GetDirection() == v1.RiskPositionDirection_RISK_POSITION_DIRECTION_UNSPECIFIED {
		return nil, errors.New("direction is required")
	}

	moveUp := req.GetDirection() == v1.RiskPositionDirection_RISK_POSITION_UP

	risk, err := s.manager.ChangeDeploymentRiskPosition(ctx, req.GetDeploymentId(), moveUp)
	if err != nil {
		direction := "up"
		if !moveUp {
			direction = "down"
		}
		return nil, errors.Wrapf(err, "failed to move deployment %s %s", req.GetDeploymentId(), direction)
	}

	message := "Deployment position moved up in ranking"
	if !moveUp {
		message = "Deployment position moved down in ranking"
	}

	return buildAdjustmentResponse(risk, message), nil
}

// ResetDeploymentRisk removes user ranking adjustments.
func (s *serviceImpl) ResetDeploymentRisk(ctx context.Context, req *v1.RiskAdjustmentRequest) (*v1.RiskAdjustmentResponse, error) {
	if req.GetDeploymentId() == "" {
		return nil, errors.New("deployment_id is required")
	}

	risk, err := s.manager.ResetDeploymentRisk(ctx, req.GetDeploymentId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to reset deployment %s", req.GetDeploymentId())
	}

	return buildAdjustmentResponse(risk, "Deployment risk reset to original ML score"), nil
}

// buildAdjustmentResponse creates the response message from a risk object.
func buildAdjustmentResponse(risk *storage.Risk, message string) *v1.RiskAdjustmentResponse {
	originalScore := risk.GetScore()
	effectiveScore := originalScore

	if adj := risk.GetUserRankingAdjustment(); adj != nil && adj.GetLastAdjusted() != nil {
		effectiveScore = adj.GetAdjustedScore()
	}

	return &v1.RiskAdjustmentResponse{
		Risk:           risk,
		OriginalScore:  originalScore,
		EffectiveScore: effectiveScore,
		Message:        message,
	}
}
