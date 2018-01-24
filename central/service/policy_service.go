package service

import (
	"errors"
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/detection"
	"bitbucket.org/stack-rox/apollo/central/detection/matcher"
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
func (s *PolicyService) GetPolicy(ctx context.Context, request *v1.ResourceByID) (*v1.Policy, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Policy id must be provided")
	}
	policy, exists, err := s.storage.GetPolicy(request.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "policy with id '%s' does not exist", request.GetId())
	}
	return policy, nil
}

// GetPolicies retrieves all policies according to the request.
func (s *PolicyService) GetPolicies(ctx context.Context, request *v1.GetPoliciesRequest) (*v1.PoliciesResponse, error) {
	policies, err := s.storage.GetPolicies(request)
	return &v1.PoliciesResponse{Policies: policies}, err
}

// PostPolicy inserts a new policy into the system.
func (s *PolicyService) PostPolicy(ctx context.Context, request *v1.Policy) (*v1.Policy, error) {
	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Id field should be empty when posting a new policy")
	}
	policy, err := validatePolicy(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	id, err := s.storage.AddPolicy(request)
	if err != nil {
		return nil, err
	}
	request.Id = id
	policy.Id = id
	s.detector.UpdatePolicy(policy)
	return request, nil
}

func validatePolicy(policy *v1.Policy) (*matcher.Policy, error) {
	if policy.GetName() == "" {
		return nil, errors.New("policy must have a set name")
	}
	if policy.GetSeverity() == v1.Severity_UNSET_SEVERITY {
		return nil, errors.New("policy must have a set severity")
	}
	if len(policy.GetCategories()) == 0 {
		return nil, errors.New("policy must have at least one category")
	}
	matcherPolicy, err := matcher.New(policy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Policy could not be edited due to: %+v", err))
	}
	return matcherPolicy, nil
}

// PutPolicy updates a current policy in the system.
func (s *PolicyService) PutPolicy(ctx context.Context, request *v1.Policy) (*empty.Empty, error) {
	policy, err := validatePolicy(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	s.detector.UpdatePolicy(policy)
	if err := s.storage.UpdatePolicy(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

// DeletePolicy deletes an policy from the system.
func (s *PolicyService) DeletePolicy(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "A policy id must be specified to delete a Policy")
	}
	if err := s.storage.RemovePolicy(request.GetId()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	s.detector.RemovePolicy(request.GetId())
	return &empty.Empty{}, nil
}
