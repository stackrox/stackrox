package service

import (
	"errors"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/detection"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log = logging.New("service")
)

// NewPolicyService returns the PolicyService API.
func NewPolicyService(storage db.PolicyStorage, detector *detection.Detector) *PolicyService {
	return &PolicyService{
		storage:  storage,
		detector: detector,
	}
}

// PolicyService is the struct that manages Policies API
type PolicyService struct {
	storage  db.PolicyStorage
	detector *detection.Detector
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *PolicyService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterPolicyServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *PolicyService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterPolicyServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// GetPolicy returns a policy by name.
func (s *PolicyService) GetPolicy(ctx context.Context, request *v1.GetPolicyRequest) (*v1.Policy, error) {
	policy, exists, err := s.storage.GetPolicy(request.GetName())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "policy with name '%s' does not exist", request.GetName())
	}

	return policy, nil
}

// GetPolicies retrieves all policies according to the request.
func (s *PolicyService) GetPolicies(ctx context.Context, request *v1.GetPoliciesRequest) (*v1.PoliciesResponse, error) {
	policies, err := s.storage.GetPolicies(request)
	return &v1.PoliciesResponse{Policies: policies}, err
}

// PostPolicy inserts a new policy into the system.
func (s *PolicyService) PostPolicy(ctx context.Context, request *v1.Policy) (*empty.Empty, error) {
	if err := validatePolicy(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.storage.AddPolicy(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if err := s.detector.UpdatePolicy(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

func validatePolicy(policy *v1.Policy) error {
	if policy.GetName() == "" {
		return errors.New("policy must have a set name")
	}
	if policy.GetSeverity() == v1.Severity_UNSET_SEVERITY {
		return errors.New("policy must have a set severity")
	}
	if len(policy.GetCategories()) == 0 {
		return errors.New("policy must have at least one category")
	}
	return nil
}

// PutPolicy updates a current policy in the system.
func (s *PolicyService) PutPolicy(ctx context.Context, request *v1.Policy) (*empty.Empty, error) {
	if err := validatePolicy(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.detector.UpdatePolicy(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if err := s.storage.UpdatePolicy(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

// DeletePolicy deletes an policy from the system.
func (s *PolicyService) DeletePolicy(ctx context.Context, request *v1.DeletePolicyRequest) (*empty.Empty, error) {
	if request.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "A policy name must be specified to delete a Policy")
	}
	if err := s.storage.RemovePolicy(request.GetName()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	s.detector.RemovePolicy(request.GetName())
	return &empty.Empty{}, nil
}
