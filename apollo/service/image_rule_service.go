package service

import (
	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/image_processor"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	log = logging.New("service")
)

// NewRuleService returns the NotifierConfigService API.
func NewRuleService(storage db.Storage, processor *imageprocessor.ImageProcessor) *RuleService {
	return &RuleService{
		storage:        storage,
		imageProcessor: processor,
	}
}

// RuleService is the struct that manages Rule API
type RuleService struct {
	storage        db.Storage
	imageProcessor *imageprocessor.ImageProcessor
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *RuleService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterImageRuleServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *RuleService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterImageRuleServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// GetImageRules retrieves all image rules
func (s *RuleService) GetImageRules(ctx context.Context, request *v1.GetImageRulesRequest) (*v1.ImageRules, error) {
	rules, err := s.storage.GetImageRules(request)
	return &v1.ImageRules{Rules: rules}, err
}

// PostImageRule inserts a new image rule into the system
func (s *RuleService) PostImageRule(ctx context.Context, request *v1.ImageRule) (*empty.Empty, error) {
	s.storage.AddImageRule(request)
	err := s.imageProcessor.UpdateRule(request)
	return &empty.Empty{}, err
}

// PutImageRule updates a current image rule into the system
func (s *RuleService) PutImageRule(ctx context.Context, request *v1.ImageRule) (*empty.Empty, error) {
	s.storage.UpdateImageRule(request)
	err := s.imageProcessor.UpdateRule(request)
	return &empty.Empty{}, err
}

// DeleteImageRule deletes an image rule from the system
func (s *RuleService) DeleteImageRule(ctx context.Context, request *v1.DeleteImageRuleRequest) (*empty.Empty, error) {
	s.storage.RemoveImageRule(request.Name)
	s.imageProcessor.RemoveRule(request.Name)
	return &empty.Empty{}, nil
}
