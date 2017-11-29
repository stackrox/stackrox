package service

import (
	"errors"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/image_processor"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
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

// NewImagePolicyService returns the ImagePolicyService API.
func NewImagePolicyService(storage db.ImagePolicyStorage, processor *imageprocessor.ImageProcessor) *ImagePolicyService {
	return &ImagePolicyService{
		storage:        storage,
		imageProcessor: processor,
	}
}

// ImagePolicyService is the struct that manages Image Policies API
type ImagePolicyService struct {
	storage        db.ImagePolicyStorage
	imageProcessor *imageprocessor.ImageProcessor
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *ImagePolicyService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterImagePolicyServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *ImagePolicyService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterImagePolicyServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// GetImagePolicies retrieves all image policies.
func (s *ImagePolicyService) GetImagePolicies(ctx context.Context, request *v1.GetImagePoliciesRequest) (*v1.ImagePoliciesResponse, error) {
	policies, err := s.storage.GetImagePolicies(request)
	return &v1.ImagePoliciesResponse{Policies: policies}, err
}

// PostImagePolicy inserts a new image policy into the system.
func (s *ImagePolicyService) PostImagePolicy(ctx context.Context, request *v1.ImagePolicy) (*empty.Empty, error) {
	if err := validateImagePolicy(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.storage.AddImagePolicy(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if err := s.imageProcessor.UpdatePolicy(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

func validateImagePolicy(policy *v1.ImagePolicy) error {
	if policy.GetName() == "" {
		return errors.New("Image policy must have a set name")
	}
	if policy.GetSeverity() == v1.Severity_UNSET_SEVERITY {
		return errors.New("Image policy must have a set severity")
	}
	return nil
}

// PutImagePolicy updates a current image policy in the system.
func (s *ImagePolicyService) PutImagePolicy(ctx context.Context, request *v1.ImagePolicy) (*empty.Empty, error) {
	if err := validateImagePolicy(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.storage.UpdateImagePolicy(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if err := s.imageProcessor.UpdatePolicy(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

// DeleteImagePolicy deletes an image policy from the system.
func (s *ImagePolicyService) DeleteImagePolicy(ctx context.Context, request *v1.DeleteImagePolicyRequest) (*empty.Empty, error) {
	if request.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "A policy name must be specified to delete an Image Policy")
	}
	if err := s.storage.RemoveImagePolicy(request.GetName()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	s.imageProcessor.RemovePolicy(request.GetName())
	return &empty.Empty{}, nil
}
