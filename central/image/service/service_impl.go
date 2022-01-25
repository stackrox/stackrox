package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	"github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/connection"
	watchedImageDataStore "github.com/stackrox/rox/central/watchedimage/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	pkgUtils "github.com/stackrox/rox/pkg/utils"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxImagesReturned = 1000

	maxSemaphoreWaitTime = 5 * time.Second
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Image)): {
			"/v1.ImageService/GetImage",
			"/v1.ImageService/CountImages",
			"/v1.ImageService/ListImages",
		},
		or.Or(idcheck.SensorsOnly(), idcheck.AdmissionControlOnly()): {
			"/v1.ImageService/ScanImageInternal",
		},
		user.With(permissions.Modify(permissions.WithLegacyAuthForSAC(resources.Image, true))): {
			"/v1.ImageService/DeleteImages",
			"/v1.ImageService/ScanImage",
		},
		user.With(permissions.View(permissions.WithLegacyAuthForSAC(resources.Image, true))): {
			"/v1.ImageService/InvalidateScanAndRegistryCaches",
		},
		user.With(permissions.View(resources.WatchedImage)): {
			"/v1.ImageService/GetWatchedImages",
		},
		user.With(permissions.Modify(resources.WatchedImage)): {
			"/v1.ImageService/WatchImage",
			"/v1.ImageService/UnwatchImage",
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	datastore    datastore.DataStore
	cveDatastore cveDataStore.DataStore
	riskManager  manager.Manager

	metadataCache expiringcache.Cache
	scanCache     expiringcache.Cache

	connManager connection.Manager

	enricher enricher.ImageEnricher

	watchedImages watchedImageDataStore.DataStore

	internalScanSemaphore *semaphore.Weighted
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterImageServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterImageServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetImage returns an image with given sha if it exists.
func (s *serviceImpl) GetImage(ctx context.Context, request *v1.GetImageRequest) (*storage.Image, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errorhelpers.ErrInvalidArgs, "id must be specified")
	}
	request.Id = types.NewDigest(request.Id).Digest()

	image, exists, err := s.datastore.GetImage(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errorhelpers.ErrNotFound, "image with id %q does not exist", request.GetId())
	}

	if !request.GetIncludeSnoozed() {
		// This modifies the image object
		utils.FilterSuppressedCVEsNoClone(image)
	}
	return image, nil
}

// CountImages counts the number of images that match the input query.
func (s *serviceImpl) CountImages(ctx context.Context, request *v1.RawQuery) (*v1.CountImagesResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errorhelpers.ErrInvalidArgs, err.Error())
	}

	numImages, err := s.datastore.Count(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}
	return &v1.CountImagesResponse{Count: int32(numImages)}, nil
}

// ListImages retrieves all images in minimal form.
func (s *serviceImpl) ListImages(ctx context.Context, request *v1.RawQuery) (*v1.ListImagesResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errorhelpers.ErrInvalidArgs, err.Error())
	}

	// Fill in pagination.
	paginated.FillPagination(parsedQuery, request.Pagination, maxImagesReturned)

	images, err := s.datastore.SearchListImages(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}

	return &v1.ListImagesResponse{
		Images: images,
	}, nil
}

// InvalidateScanAndRegistryCaches invalidates the image scan caches
func (s *serviceImpl) InvalidateScanAndRegistryCaches(context.Context, *v1.Empty) (*v1.Empty, error) {
	s.metadataCache.RemoveAll()
	s.scanCache.RemoveAll()
	return &v1.Empty{}, nil
}

func scanImageInternalResponseFromImage(img *storage.Image) *v1.ScanImageInternalResponse {
	utils.FilterSuppressedCVEsNoClone(img)
	utils.StripCVEDescriptionsNoClone(img)
	return &v1.ScanImageInternalResponse{
		Image: img,
	}
}

// ScanImageInternal handles an image request from Sensor
func (s *serviceImpl) ScanImageInternal(ctx context.Context, request *v1.ScanImageInternalRequest) (*v1.ScanImageInternalResponse, error) {
	if err := s.internalScanSemaphore.Acquire(concurrency.AsContext(concurrency.Timeout(maxSemaphoreWaitTime)), 1); err != nil {
		s, err := status.New(codes.Unavailable, err.Error()).WithDetails(&v1.ScanImageInternalResponseDetails_TooManyParallelScans{})
		if pkgUtils.Should(err) == nil {
			return nil, s.Err()
		}
	}
	defer s.internalScanSemaphore.Release(1)

	// Always pull the image from the store if the ID != "". Central will manage the reprocessing over the
	// images
	if request.GetImage().GetId() != "" {
		img, exists, err := s.datastore.GetImage(ctx, request.GetImage().GetId())
		if err != nil {
			return nil, err
		}
		// If the scan exists and it is less than the reprocessing interval then return the scan. Otherwise, fetch it from the DB
		if exists {
			return scanImageInternalResponseFromImage(img), nil
		}
	}

	// If no ID, then don't use caches as they could return stale data
	fetchOpt := enricher.UseCachesIfPossible
	if request.GetCachedOnly() {
		fetchOpt = enricher.NoExternalMetadata
	} else if request.GetImage().GetId() == "" {
		fetchOpt = enricher.ForceRefetch
	}

	img := types.ToImage(request.GetImage())
	if _, err := s.enricher.EnrichImage(enricher.EnrichmentContext{FetchOpt: fetchOpt, Internal: true}, img); err != nil {
		log.Errorf("error enriching image %q: %v", request.GetImage().GetName().GetFullName(), err)
		// purposefully, don't return here because we still need to save it into the DB so there is a reference
		// even if we weren't able to enrich it
	}

	img.Id = utils.GetImageID(img)
	if img.GetId() != "" {
		if err := s.riskManager.CalculateRiskAndUpsertImage(img); err != nil {
			log.Errorf("error upserting image %q: %v", img.GetName().GetFullName(), err)
		}
	}

	// This modifies the image object
	return scanImageInternalResponseFromImage(img), nil
}

// ScanImage scans an image and returns the result
func (s *serviceImpl) ScanImage(ctx context.Context, request *v1.ScanImageRequest) (*storage.Image, error) {
	enrichmentCtx := enricher.EnrichmentContext{
		FetchOpt: enricher.IgnoreExistingImages,
	}
	if request.GetForce() {
		enrichmentCtx.FetchOpt = enricher.ForceRefetch
	}
	img, err := enricher.EnrichImageByName(s.enricher, enrichmentCtx, request.GetImageName())
	if err != nil {
		return nil, err
	}

	// Save the image
	img.Id = utils.GetImageID(img)
	if img.GetId() != "" {
		if err := s.riskManager.CalculateRiskAndUpsertImage(img); err != nil {
			return nil, err
		}
	}
	if !request.GetIncludeSnoozed() {
		utils.FilterSuppressedCVEsNoClone(img)
	}
	return img, nil
}

// DeleteImages deletes images based on query
func (s *serviceImpl) DeleteImages(ctx context.Context, request *v1.DeleteImagesRequest) (*v1.DeleteImagesResponse, error) {
	if request.GetQuery() == nil {
		return nil, errors.New("a scoping query is required")
	}

	query, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errorhelpers.ErrInvalidArgs, "error parsing query: %v", err)
	}
	paginated.FillPagination(query, request.GetQuery().GetPagination(), math.MaxInt32)

	results, err := s.datastore.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	response := &v1.DeleteImagesResponse{
		NumDeleted: uint32(len(results)),
		DryRun:     !request.GetConfirm(),
	}

	if !request.GetConfirm() {
		return response, nil
	}

	idSlice := search.ResultsToIDs(results)
	if err := s.datastore.DeleteImages(ctx, idSlice...); err != nil {
		return nil, err
	}

	keys := make([]*central.InvalidateImageCache_ImageKey, 0, len(idSlice))
	for _, id := range idSlice {
		keys = append(keys, &central.InvalidateImageCache_ImageKey{
			ImageId: id,
		})
	}

	s.connManager.BroadcastMessage(&central.MsgToSensor{
		Msg: &central.MsgToSensor_InvalidateImageCache{
			InvalidateImageCache: &central.InvalidateImageCache{
				ImageKeys: keys,
			},
		},
	})

	return response, nil
}

func (s *serviceImpl) WatchImage(ctx context.Context, request *v1.WatchImageRequest) (*v1.WatchImageResponse, error) {
	if request.GetName() == "" {
		return &v1.WatchImageResponse{
			ErrorMessage: "no image name specified",
			ErrorType:    v1.WatchImageResponse_INVALID_IMAGE_NAME,
		}, nil
	}
	containerImage, err := utils.GenerateImageFromString(request.GetName())
	if err != nil {
		return &v1.WatchImageResponse{
			ErrorMessage: fmt.Sprintf("failed to parse name: %v", err),
			ErrorType:    v1.WatchImageResponse_INVALID_IMAGE_NAME,
		}, nil
	}
	if containerImage.Id != "" {
		return &v1.WatchImageResponse{
			ErrorMessage: fmt.Sprintf("name %s contains a digest, but watch does not handle images with digests", request.GetName()),
			ErrorType:    v1.WatchImageResponse_INVALID_IMAGE_NAME,
		}, nil
	}

	img := types.ToImage(containerImage)

	enrichmentResult, err := s.enricher.EnrichImage(enricher.EnrichmentContext{FetchOpt: enricher.IgnoreExistingImages}, img)
	if err != nil {
		return &v1.WatchImageResponse{
			ErrorMessage: fmt.Sprintf("failed to scan image: %v", err),
			ErrorType:    v1.WatchImageResponse_SCAN_FAILED,
		}, nil
	}

	if !enrichmentResult.ImageUpdated || (enrichmentResult.ScanResult != enricher.ScanSucceeded) {
		return &v1.WatchImageResponse{
			ErrorMessage: "scan could not be completed, due to no applicable registry/scanner integration",
			ErrorType:    v1.WatchImageResponse_NO_VALID_INTEGRATION,
		}, nil
	}

	// Save the image
	img.Id = utils.GetImageID(img)
	if img.GetId() == "" {
		return &v1.WatchImageResponse{
			ErrorType:    v1.WatchImageResponse_SCAN_FAILED,
			ErrorMessage: "could not get SHA after scanning image",
		}, nil
	}

	if err := s.riskManager.CalculateRiskAndUpsertImage(img); err != nil {
		return nil, errors.Errorf("failed to store image: %v", err)
	}

	normalizedName := img.GetName().GetFullName()
	if err := s.watchedImages.UpsertWatchedImage(ctx, normalizedName); err != nil {
		return nil, errors.Errorf("failed to upsert watched image: %v", err)
	}
	return &v1.WatchImageResponse{NormalizedName: normalizedName}, nil
}

func (s *serviceImpl) UnwatchImage(ctx context.Context, request *v1.UnwatchImageRequest) (*v1.Empty, error) {
	if err := s.watchedImages.UnwatchImage(ctx, request.GetName()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) GetWatchedImages(ctx context.Context, empty *v1.Empty) (*v1.GetWatchedImagesResponse, error) {
	watchedImgs, err := s.watchedImages.GetAllWatchedImages(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetWatchedImagesResponse{WatchedImages: watchedImgs}, nil
}
