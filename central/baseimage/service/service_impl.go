package service

import (
	"context"
	"fmt"
	"path"

	"github.com/distribution/reference"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/central/baseimage/datastore/repository"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/delegatedregistry"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/images/integration"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.ImageAdministration)): {
			v2.BaseImageServiceV2_GetBaseImageReferences_FullMethodName,
		},
		user.With(permissions.Modify(resources.ImageAdministration)): {
			v2.BaseImageServiceV2_CreateBaseImageReference_FullMethodName,
			v2.BaseImageServiceV2_UpdateBaseImageTagPattern_FullMethodName,
			v2.BaseImageServiceV2_DeleteBaseImageReference_FullMethodName,
		},
	})
)

// serviceImpl provides APIs for base image references.
type serviceImpl struct {
	v2.UnimplementedBaseImageServiceV2Server

	datastore      repository.DataStore
	integrationSet integration.Set
	delegator      delegatedregistry.Delegator
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
	if err := s.validateBaseImageRepository(ctx, req.GetBaseImageRepoPath()); err != nil {
		return nil, err
	}

	if valid, err := isValidTagPattern(req.GetBaseImageTagPattern()); !valid {
		return nil, status.Errorf(codes.InvalidArgument, "invalid base image tag pattern: %v", err)
	}

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

	// Update the repository
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
	user := &v2.SlimUser{
		Id:   repo.GetCreatedBy().GetId(),
		Name: repo.GetCreatedBy().GetName(),
	}
	return &v2.BaseImageReference{
		Id:                  repo.GetId(),
		BaseImageRepoPath:   repo.GetRepositoryPath(),
		BaseImageTagPattern: repo.GetTagPattern(),
		User:                user,
	}
}

// validateBaseImageRepository validate the provided base image repository path,
// Returns an error if the repository path is malformed or no matching image integration is found.
func (s *serviceImpl) validateBaseImageRepository(ctx context.Context, repoPath string) error {
	imageName, ref, err := imgUtils.GenerateImageNameFromString(repoPath)
	if err != nil {
		return errox.InvalidArgs.Newf("invalid base image repository path '%s'", repoPath).CausedBy(err)
	}
	// Reject references that include tags - these should be in separate fields
	if imageName.GetTag() != "" {
		return errox.InvalidArgs.Newf("repository path '%s' must not include tag - please put tag in the tag pattern field", repoPath)
	}
	// Reject references that include digests
	if _, ok := ref.(reference.Digested); ok {
		return errox.InvalidArgs.Newf("repository path '%s' must not include digest", repoPath)
	}

	// Check delegated registry config (only if feature is enabled).
	var shouldDelegate bool
	if features.DelegatedBaseImageScanning.Enabled() {
		_, shouldDelegate, _ = s.delegator.GetDelegateClusterID(ctx, imageName)
	}
	if !shouldDelegate {
		// Check central registry config.
		if !s.integrationSet.RegistrySet().Match(imageName) {
			return errox.InvalidArgs.Newf("no matching image integration found: please add an image integration for '%s'", repoPath)
		}
	}

	return nil
}

// isValidTagPattern checks if the given string is a valid [path.Match] glob pattern.
func isValidTagPattern(tagPattern string) (bool, error) {
	// Reject empty tag patterns, should use * instead
	if tagPattern == "" {
		return false, errox.InvalidArgs.New("tag pattern cannot be empty")
	}
	// Validate the pattern by attempting to match it against an empty string.
	// If the pattern is malformed, path.Match will return an error.
	_, err := path.Match(tagPattern, "")
	if err != nil {
		return false, errox.InvalidArgs.CausedBy(fmt.Errorf("invalid tag pattern: %w", err))
	}

	return true, nil
}
