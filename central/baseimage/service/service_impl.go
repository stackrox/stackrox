package service

import (
	"context"
	"regexp"

	"github.com/distribution/reference"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/baseimage/datastore/repository"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// TODO(ROX-32170): Review and finalize RBAC for base image repository operations.
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Administration)): {
			v2.BaseImageServiceV2_GetBaseImageReferences_FullMethodName,
		},
		user.With(permissions.Modify(resources.Administration)): {
			v2.BaseImageServiceV2_CreateBaseImageReference_FullMethodName,
			v2.BaseImageServiceV2_UpdateBaseImageTagPattern_FullMethodName,
			v2.BaseImageServiceV2_DeleteBaseImageReference_FullMethodName,
		},
	})
)

// serviceImpl provides APIs for base image references.
type serviceImpl struct {
	v2.UnimplementedBaseImageServiceV2Server

	datastore repository.DataStore
}

// GetBaseImageReferences returns all base image references.
func (s *serviceImpl) GetBaseImageReferences(ctx context.Context, _ *v2.Empty) (*v2.GetBaseImageReferenceResponse, error) {
	repos, err := s.datastore.ListRepositories(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get base image repositories: %v", err)
	}

	references := make([]*v2.BaseImageReference, 0, len(repos))
	for _, repo := range repos {
		references = append(references, convertStorageToAPI(repo))
	}

	return &v2.GetBaseImageReferenceResponse{
		BaseImageReferences: references,
	}, nil
}

// CreateBaseImageReference creates a new base image reference.
func (s *serviceImpl) CreateBaseImageReference(ctx context.Context, req *v2.CreateBaseImageReferenceRequest) (*v2.CreateBaseImageReferenceResponse, error) {
	if valid, err := isValidRepo(req.GetBaseImageRepoPath()); !valid {
		return nil, status.Errorf(codes.InvalidArgument, "invalid base image repo path: %v", err)
	}

	if valid, err := isValidTagPattern(req.GetBaseImageTagPattern()); !valid {
		return nil, status.Errorf(codes.InvalidArgument, "invalid base image tag pattern: %v", err)
	}

	// TODO(ROX-32170): Populate user information to the base image repository.
	repo := &storage.BaseImageRepository{
		RepositoryPath: req.GetBaseImageRepoPath(),
		TagPattern:     req.GetBaseImageTagPattern(),
	}

	created, err := s.datastore.UpsertRepository(ctx, repo)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create base image repository: %v", err)
	}

	return &v2.CreateBaseImageReferenceResponse{
		BaseImageReference: convertStorageToAPI(created),
	}, nil
}

// UpdateBaseImageTagPattern updates the tag pattern of an existing base image reference.
func (s *serviceImpl) UpdateBaseImageTagPattern(ctx context.Context, req *v2.UpdateBaseImageTagPatternRequest) (*v2.Empty, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "base image reference ID is required")
	}

	if valid, err := isValidTagPattern(req.GetBaseImageTagPattern()); !valid {
		return nil, status.Errorf(codes.InvalidArgument, "invalid base image tag pattern: %v", err)
	}

	// First get the existing repository
	existing, found, err := s.datastore.GetRepository(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get base image repository: %v", err)
	}
	if !found {
		return nil, status.Errorf(codes.NotFound, "base image repository with ID %q not found", req.GetId())
	}

	// Update the repository (only tag pattern can be updated)
	existing.TagPattern = req.GetBaseImageTagPattern()

	_, err = s.datastore.UpsertRepository(ctx, existing)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update base image repository: %v", err)
	}

	return &v2.Empty{}, nil
}

// DeleteBaseImageReference deletes a base image reference.
func (s *serviceImpl) DeleteBaseImageReference(ctx context.Context, req *v2.DeleteBaseImageReferenceRequest) (*v2.Empty, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "base image reference ID is required")
	}

	err := s.datastore.DeleteRepository(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete base image repository: %v", err)
	}

	return &v2.Empty{}, nil
}

// AuthFuncOverride applies the authorizer to the request.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// RegisterServiceServer registers this service with the given gRPC server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v2.RegisterBaseImageServiceV2Server(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC gateway.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterBaseImageServiceV2Handler(ctx, mux, conn)
}

// convertStorageToAPI converts a storage.BaseImageRepository to a v2.BaseImageReference.
func convertStorageToAPI(repo *storage.BaseImageRepository) *v2.BaseImageReference {
	return &v2.BaseImageReference{
		Id:                  repo.GetId(),
		BaseImageRepoPath:   repo.GetRepositoryPath(),
		BaseImageTagPattern: repo.GetTagPattern(),
	}
}

// isValidRepo checks if the given string is a valid repository reference.
// It uses distribution/reference.Parse to validate the input.
// Returns true only if the string represents a valid repository reference without tag or digest.
// Returns false with a specific error message for invalid references.
func isValidRepo(repo string) (bool, error) {
	ref, err := reference.Parse(repo)
	if err != nil {
		// Map distribution errors to more user-friendly messages
		switch err {
		case reference.ErrNameEmpty:
			return false, errox.InvalidArgs.New("repository cannot be empty")
		case reference.ErrNameContainsUppercase:
			return false, errox.InvalidArgs.New("repository path must be lowercase")
		case reference.ErrReferenceInvalidFormat:
			return false, errox.InvalidArgs.New("invalid repository path format")
		case reference.ErrNameTooLong:
			return false, errox.InvalidArgs.New("repository path must not be more than 255 characters")
		default:
			return false, errox.InvalidArgs.CausedBy(errors.Wrap(err, "invalid repository reference"))
		}
	}

	// Reject references that include digests - these should be in separate fields
	if _, ok := ref.(reference.Digested); ok {
		return false, errox.InvalidArgs.New("repository path must not include digest - please put tag in the tag pattern field")
	}
	// Reject references that include tags - these should be in separate fields
	if _, ok := ref.(reference.Tagged); ok {
		return false, errox.InvalidArgs.New("repository path must not include tag - please put tag in the tag pattern field")
	}

	return true, nil
}

// isValidTagPattern checks if the given string is a valid regex pattern.
// The tag pattern must be a valid regular expression.
func isValidTagPattern(tagPattern string) (bool, error) {
	if _, err := regexp.Compile(tagPattern); err != nil {
		return false, errox.InvalidArgs.CausedBy(errors.Wrap(err, "invalid tag pattern regex"))
	}

	return true, nil
}
