package services

import (
	"context"
	"crypto/sha512"
	"fmt"
	"net/url"
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/quay/claircore"
	"github.com/quay/zlog"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/scanner/indexer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type indexerService struct {
	v4.UnimplementedIndexerServer
	indexer indexer.Indexer
}

// NewIndexerService creates a new indexer service.
func NewIndexerService(indexer indexer.Indexer) *indexerService {
	return &indexerService{
		indexer: indexer,
	}
}

func (s *indexerService) CreateIndexReport(ctx context.Context, req *v4.CreateIndexReportRequest) (*v4.IndexReport, error) {
	// TODO We currently only support container images, hence we assume the resource
	//      is of that type. When introducing nodes and other resources, this should
	//      evolve.
	resourceType := "containerimage"
	if err := validateContainerImageRequest(req); err != nil {
		return nil, err
	}
	// Create a claircore.Digest using the full Hash ID, including resource type.
	hashIDSum := sha512.Sum512([]byte(req.GetHashId()))
	manifestDigest, err := claircore.NewDigest(claircore.SHA512, hashIDSum[:])
	if err != nil {
		return nil, fmt.Errorf("internal error: creating container image manifest digest: %w", err)
	}

	ctx = zlog.ContextWithValues(ctx,
		"component", "scanner/service/indexer",
		"resource_type", resourceType,
		"manifest_digest", manifestDigest.String())

	// Setup authentication.
	var opts []indexer.Option
	hasAuth := req.GetContainerImage().GetUsername() != ""
	if hasAuth {
		opts = append(opts, indexer.WithAuth(&authn.Basic{
			Username: req.GetContainerImage().GetUsername(),
			Password: req.GetContainerImage().GetPassword(),
		}))
	}

	// Create index report.
	zlog.Info(ctx).
		Str("image_url", req.GetContainerImage().GetUrl()).
		Bool("has_auth", hasAuth).
		Msg("creating index report for container image")
	clairReport, err := s.indexer.IndexContainerImage(
		ctx,
		manifestDigest,
		req.GetContainerImage().GetUrl(),
		opts...)
	if err != nil {
		zlog.Error(ctx).Err(err).Send()
		return nil, err
	}
	indexReport := convertToIndexReport(clairReport)
	indexReport.HashId = req.GetHashId()
	// TODO Define behavior for indexReport.Err != "".
	return indexReport, nil
}

// validateContainerImageRequest validates a container image request.
func validateContainerImageRequest(req *v4.CreateIndexReportRequest) error {
	if req == nil {
		return errox.InvalidArgs.New("empty request")
	}
	if !strings.HasPrefix(req.GetHashId(), "/v4/containerimage/") {
		return errox.InvalidArgs.Newf("invalid hash id: %q", req.GetHashId())
	}
	if req.GetContainerImage() == nil {
		return errox.InvalidArgs.New("invalid resource locator for container image")
	}
	// Validate container image URL.
	imgURL := req.GetContainerImage().GetUrl()
	if imgURL == "" {
		return errox.InvalidArgs.New("missing image URL")
	}
	u, err := url.Parse(imgURL)
	if err != nil {
		return errox.InvalidArgs.Newf("invalid image URL: %q", imgURL).CausedBy(err)
	}
	switch u.Scheme {
	case "http", "https":
	default:
		return errox.InvalidArgs.New("image URL does not start with http:// or https://")
	}
	imageRef := strings.TrimPrefix(imgURL, u.Scheme+"://")
	_, err = name.ParseReference(imageRef, name.StrictValidation)
	if err != nil {
		return errox.InvalidArgs.CausedBy(err)
	}
	return nil
}

func (s *indexerService) GetIndexReport(_ context.Context, _ *v4.GetIndexReportRequest) (*v4.IndexReport, error) {
	return nil, status.Error(codes.Unimplemented, "method GetIndexReport not implemented")
}

func (s *indexerService) HasIndexReport(_ context.Context, _ *v4.HasIndexReportRequest) (*types.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "method HasIndexReport not implemented")
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *indexerService) RegisterServiceServer(grpcServer *grpc.Server) {
	v4.RegisterIndexerServer(grpcServer, s)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *indexerService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	// TODO: Setup permissions for indexer.
	return ctx, allow.Anonymous().Authorized(ctx, fullMethodName)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *indexerService) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	// Currently we do not set up gRPC gateway for the matcher.
	return nil
}
