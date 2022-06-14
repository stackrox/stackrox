package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	cveDataStore "github.com/stackrox/rox/central/cve/image/datastore"
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
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
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
	"github.com/stackrox/rox/pkg/timestamp"
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
		or.SensorOrAuthorizer(idcheck.AdmissionControlOnly()): {
			"/v1.ImageService/ScanImageInternal",
		},
		idcheck.SensorsOnly(): {
			"/v1.ImageService/GetImageVulnerabilitiesInternal",
			"/v1.ImageService/EnrichLocalImageInternal",
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

	reprocessInterval = env.ReprocessInterval.DurationSetting()
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	datastore    datastore.DataStore
	cveDatastore cveDataStore.DataStore
	riskManager  manager.Manager

	metadataCache expiringcache.Cache

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
		return nil, errors.Wrap(errox.InvalidArgs, "id must be specified")
	}

	id := types.NewDigest(request.GetId()).Digest()

	image, exists, err := s.datastore.GetImage(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "image with id %q does not exist", request.GetId())
	}

	if !request.GetIncludeSnoozed() {
		// This modifies the image object
		utils.FilterSuppressedCVEsNoClone(image)
	}
	if request.GetStripDescription() {
		// This modifies the image object
		utils.StripCVEDescriptionsNoClone(image)
	}

	return image, nil
}

// CountImages counts the number of images that match the input query.
func (s *serviceImpl) CountImages(ctx context.Context, request *v1.RawQuery) (*v1.CountImagesResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
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
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
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
	return &v1.Empty{}, nil
}

func internalScanRespFromImage(img *storage.Image) *v1.ScanImageInternalResponse {
	utils.FilterSuppressedCVEsNoClone(img)
	utils.StripCVEDescriptionsNoClone(img)
	return &v1.ScanImageInternalResponse{
		Image: img,
	}
}

func (s *serviceImpl) saveImage(img *storage.Image) error {
	if err := s.riskManager.CalculateRiskAndUpsertImage(img); err != nil {
		log.Errorf("error upserting image %q: %v", img.GetName().GetFullName(), err)
		return err
	}
	return nil
}

// ScanImageInternal handles an image request from Sensor and Admission Controller.
func (s *serviceImpl) ScanImageInternal(ctx context.Context, request *v1.ScanImageInternalRequest) (*v1.ScanImageInternalResponse, error) {
	err := s.acquireScanSemaphore()
	if err != nil {
		return nil, err
	}

	defer s.internalScanSemaphore.Release(1)

	imgID := request.GetImage().GetId()

	// Always pull the image from the store if the ID != "". Central will manage the reprocessing over the images.
	if imgID != "" {
		existingImg, exists, err := s.datastore.GetImage(ctx, imgID)
		if err != nil {
			return nil, err
		}
		// If the scan exists, return the scan.
		// Otherwise, run the enrichment pipeline.
		if exists {
			return internalScanRespFromImage(existingImg), nil
		}
	}

	// If no ID, then don't use caches as they could return stale data
	fetchOpt := enricher.UseCachesIfPossible
	if request.GetCachedOnly() {
		fetchOpt = enricher.NoExternalMetadata
	} else if imgID == "" {
		fetchOpt = enricher.ForceRefetch
	}

	img := types.ToImage(request.GetImage())
	if _, err := s.enricher.EnrichImage(ctx, enricher.EnrichmentContext{FetchOpt: fetchOpt, Internal: true}, img); err != nil {
		log.Errorf("error enriching image %q: %v", request.GetImage().GetName().GetFullName(), err)
		// purposefully, don't return here because we still need to save it into the DB so there is a reference
		// even if we weren't able to enrich it
	}

	// Due to discrepancies in digests retrieved from metadata pulls and k8s, only upsert if the request
	// contained a digest
	if imgID != "" {
		_ = s.saveImage(img)
	}

	return internalScanRespFromImage(img), nil
}

// ScanImage scans an image and returns the result
func (s *serviceImpl) ScanImage(ctx context.Context, request *v1.ScanImageRequest) (*storage.Image, error) {
	enrichmentCtx := enricher.EnrichmentContext{
		FetchOpt: enricher.IgnoreExistingImages,
	}
	if request.GetForce() {
		enrichmentCtx.FetchOpt = enricher.ForceRefetch
	}
	img, err := enricher.EnrichImageByName(ctx, s.enricher, enrichmentCtx, request.GetImageName())
	if err != nil {
		return nil, err
	}

	// Save the image
	img.Id = utils.GetImageID(img)
	if img.GetId() != "" {
		if err := s.saveImage(img); err != nil {
			return nil, err
		}
	}
	if !request.GetIncludeSnoozed() {
		utils.FilterSuppressedCVEsNoClone(img)
	}

	return img, nil
}

// GetImageVulnerabilitiesInternal retrieves the vulnerabilities related to the image
// specified by the given components and scan notes.
// This is meant to be called by Sensor.
func (s *serviceImpl) GetImageVulnerabilitiesInternal(ctx context.Context, request *v1.GetImageVulnerabilitiesInternalRequest) (*v1.ScanImageInternalResponse, error) {
	err := s.acquireScanSemaphore()
	if err != nil {
		return nil, err
	}
	defer s.internalScanSemaphore.Release(1)

	imgID := request.GetImageId()

	// Always pull the image from the store if the ID != "". Central will manage the reprocessing over the images.
	if imgID != "" {
		existingImg, exists, err := s.datastore.GetImage(ctx, imgID)
		if err != nil {
			return nil, err
		}
		// This is safe even if img is nil.
		scanTime := existingImg.GetScan().GetScanTime()
		// If the scan exists, and reprocessing has not run since, return the scan.
		// Otherwise, run the enrichment pipeline to ensure we do not return stale data.
		if exists && timestamp.FromProtobuf(scanTime).Add(reprocessInterval).After(timestamp.Now()) {
			return internalScanRespFromImage(existingImg), nil
		}
	}

	img := &storage.Image{
		Id:             imgID,
		Name:           request.GetImageName(),
		Metadata:       request.GetMetadata(),
		IsClusterLocal: request.GetIsClusterLocal(),
	}
	_, err = s.enricher.EnrichWithVulnerabilities(img, request.GetComponents(), request.GetNotes())
	if err != nil {
		return nil, err
	}

	// Due to discrepancies in digests retrieved from metadata pulls and k8s, only upsert if the request
	// contained a digest
	if imgID != "" {
		_ = s.saveImage(img)
	}

	return internalScanRespFromImage(img), nil
}

func (s *serviceImpl) acquireScanSemaphore() error {
	if err := s.internalScanSemaphore.Acquire(concurrency.AsContext(concurrency.Timeout(maxSemaphoreWaitTime)), 1); err != nil {
		s, err := status.New(codes.Unavailable, err.Error()).WithDetails(&v1.ScanImageInternalResponseDetails_TooManyParallelScans{})
		if pkgUtils.Should(err) == nil {
			return s.Err()
		}
	}
	return nil
}

func (s *serviceImpl) EnrichLocalImageInternal(ctx context.Context, request *v1.EnrichLocalImageInternalRequest) (*v1.ScanImageInternalResponse, error) {
	err := s.acquireScanSemaphore()
	if err != nil {
		return nil, err
	}

	defer s.internalScanSemaphore.Release(1)

	forceSigVerificationUpdate := true
	forceScanUpdate := true
	imgID := request.GetImageId()
	// Always pull the image from the store if the ID != "". Central will manage the reprocessing over the images.
	if imgID != "" {
		existingImg, exists, err := s.datastore.GetImage(ctx, imgID)
		if err != nil {
			return nil, err
		}
		// This is safe even if img is nil.
		scanTime := existingImg.GetScan().GetScanTime()

		// Check whether too much time has passed, if yes we have to do a signature verification update via the
		// enrichment pipeline to ensure we do not return stale data. Only do this when the image signature verification
		// feature is enabled. If no verification result is given, we can assume that the image doesn't have any
		// signatures associated with it.
		if exists && features.ImageSignatureVerification.Enabled() &&
			len(existingImg.GetSignatureVerificationData().GetResults()) > 0 {
			// For now, all verification results within the signature verification data will have approximately the same
			// time, their margin being ns.
			verificationTime := existingImg.GetSignatureVerificationData().GetResults()[0].GetVerificationTime()
			forceSigVerificationUpdate = !timestamp.FromProtobuf(verificationTime).
				Add(reprocessInterval).After(timestamp.Now())
		}

		// If the scan exists and not too much time has passed, we don't need to update scans.
		forceScanUpdate = !timestamp.FromProtobuf(scanTime).Add(reprocessInterval).After(timestamp.Now())

		// If the image exists and scan / signature verification results do not need an update yet, return it.
		// Otherwise, reprocess the image.
		if exists && !forceScanUpdate && !forceSigVerificationUpdate {
			return internalScanRespFromImage(existingImg), nil
		}
	}

	img := &storage.Image{
		Id:             imgID,
		Name:           request.GetImageName(),
		Signature:      request.GetImageSignature(),
		Metadata:       request.GetMetadata(),
		IsClusterLocal: true,
	}

	if forceScanUpdate {
		if _, err := s.enricher.EnrichWithVulnerabilities(img, request.GetComponents(), request.GetNotes()); err != nil {
			return nil, err
		}
	}

	if features.ImageSignatureVerification.Enabled() && forceSigVerificationUpdate {
		if _, err := s.enricher.EnrichWithSignatureVerificationData(ctx, img); err != nil {
			return nil, err
		}
	}

	// Due to discrepancies in digests retrieved from metadata pulls and k8s, only upsert if the request
	// contained a digest
	if imgID != "" {
		_ = s.saveImage(img)
	}

	return internalScanRespFromImage(img), nil
}

// DeleteImages deletes images based on query
func (s *serviceImpl) DeleteImages(ctx context.Context, request *v1.DeleteImagesRequest) (*v1.DeleteImagesResponse, error) {
	if request.GetQuery() == nil {
		return nil, errors.New("a scoping query is required")
	}

	query, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "error parsing query: %v", err)
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

	enrichmentResult, err := s.enricher.EnrichImage(ctx, enricher.EnrichmentContext{FetchOpt: enricher.IgnoreExistingImages}, img)
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

	if err := s.saveImage(img); err != nil {
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
