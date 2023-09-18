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
	"github.com/stackrox/rox/scanner/services/converters"
	"google.golang.org/grpc"
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
	manifestDigest, err := createManifestDigest(req.GetHashId())
	if err != nil {
		return nil, err
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
	indexReport := converters.ToProtoV4IndexReport(clairReport)
	indexReport.HashId = req.GetHashId()
	// TODO Define behavior for indexReport.Err != "".
	return indexReport, nil
}

func (s *indexerService) GetIndexReport(ctx context.Context, req *v4.GetIndexReportRequest) (*v4.IndexReport, error) {
	clairReport, err := s.getClairIndexReport(ctx, req.GetHashId())
	if err != nil {
		return nil, err
	}
	indexReport := converters.ToProtoV4IndexReport(clairReport)
	indexReport.HashId = req.GetHashId()
	return indexReport, nil
}

func (s *indexerService) HasIndexReport(ctx context.Context, req *v4.HasIndexReportRequest) (*types.Empty, error) {
	_, err := s.getClairIndexReport(ctx, req.GetHashId())
	if err != nil {
		return nil, err
	}
	return &types.Empty{}, nil
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

// createManifestDigest creates a unique claircore.Digest from a Scanner's manifest hash ID.
func createManifestDigest(hashID string) (claircore.Digest, error) {
	hashIDSum := sha512.Sum512([]byte(hashID))
	d, err := claircore.NewDigest(claircore.SHA512, hashIDSum[:])
	if err != nil {
		return claircore.Digest{}, fmt.Errorf("creating manifest digest: %w", err)
	}
	return d, nil
}

// getClairIndexReport query and return a claircore index report, return a "not
// found" error when the report does not exist.
func (s *indexerService) getClairIndexReport(ctx context.Context, hashID string) (*claircore.IndexReport, error) {
	manifestDigest, err := createManifestDigest(hashID)
	if err != nil {
		return nil, err
	}
	clairReport, ok, err := s.indexer.GetIndexReport(ctx, manifestDigest)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errox.NotFound.Newf("index report not found: %s", hashID)
	}
	return clairReport, nil
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
