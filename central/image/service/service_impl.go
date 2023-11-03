package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	clusterUtil "github.com/stackrox/rox/central/cluster/util"
	"github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/sachelper"
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
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/timestamp"
	pkgUtils "github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/waiter"
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
		or.SensorOr(idcheck.AdmissionControlOnly()): {
			"/v1.ImageService/ScanImageInternal",
		},
		idcheck.SensorsOnly(): {
			"/v1.ImageService/GetImageVulnerabilitiesInternal",
			"/v1.ImageService/EnrichLocalImageInternal",
			"/v1.ImageService/UpdateLocalScanStatusInternal",
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

	delegateScanPermissions = []string{"Image"}
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	v1.UnimplementedImageServiceServer

	datastore   datastore.DataStore
	riskManager manager.Manager

	metadataCache expiringcache.Cache

	connManager connection.Manager

	enricher enricher.ImageEnricher

	watchedImages watchedImageDataStore.DataStore

	internalScanSemaphore *semaphore.Weighted

	scanWaiterManager waiter.Manager[*storage.Image]

	clusterSACHelper sachelper.ClusterSacHelper
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
	paginated.FillPagination(parsedQuery, request.GetPagination(), maxImagesReturned)

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
		log.Errorw("Error upserting image", logging.ImageName(img.GetName().GetFullName()), logging.Err(err))
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

	var (
		img       *storage.Image
		fetchOpt  enricher.FetchOption
		imgExists bool
	)

	imgID := request.GetImage().GetId()
	// Always pull the image from the store if the ID != "". Central will manage the reprocessing over the images.
	if imgID != "" {
		existingImg, exists, err := s.datastore.GetImage(ctx, imgID)
		if err != nil {
			return nil, err
		}

		// If the image exists and the image name from the request matches at least one stored image name(/reference),
		// then we returned the stored image.
		// Otherwise, we run the enrichment pipeline using the existing image with the requests image being added to it.
		if exists {
			if protoutils.SliceContains(request.GetImage().GetName(), existingImg.GetNames()) {
				return internalScanRespFromImage(existingImg), nil
			}
			existingImg.Names = append(existingImg.Names, request.GetImage().GetName())
			img = existingImg
			// We only want to force re-fetching of signatures and verification data, the additional image name has no
			// impact on image scan data.
			fetchOpt = enricher.ForceRefetchSignaturesOnly
			imgExists = true
		}
	}

	if img == nil {
		fetchOpt = enricher.UseCachesIfPossible
		if request.GetCachedOnly() {
			fetchOpt = enricher.NoExternalMetadata
		} else if imgID == "" { // If no ID, then don't use caches as they could return stale data.
			fetchOpt = enricher.ForceRefetch
		}
		img = types.ToImage(request.GetImage())
	}

	if err := s.enrichImage(ctx, img, fetchOpt, request.GetSource()); err != nil && imgExists {
		// In case we hit an error during enriching, and the image previously existed, we will _not_ upsert it in
		// central, since it could lead to us overriding an enriched image with a non-enriched image.
		return internalScanRespFromImage(img), nil
	}
	// Due to discrepancies in digests retrieved from metadata pulls and k8s, only upsert if the request
	// contained a digest.
	if imgID != "" {
		_ = s.saveImage(img)
	}

	return internalScanRespFromImage(img), nil
}

// enrichImage will enrich the given image, additionally applying the request source and fetch option to the request.
// Any occurred error will be logged, and the given image will be modified, after execution it will contain the enriched
// image data (i.e. scan results, signature data etc.).
func (s *serviceImpl) enrichImage(ctx context.Context, img *storage.Image, fetchOpt enricher.FetchOption,
	requestSource *v1.ScanImageInternalRequest_Source) error {
	enrichmentContext := enricher.EnrichmentContext{
		FetchOpt: fetchOpt,
		Internal: true,
	}

	if features.SourcedAutogeneratedIntegrations.Enabled() && requestSource != nil {
		enrichmentContext.Source = &enricher.RequestSource{
			ClusterID:        requestSource.GetClusterId(),
			Namespace:        requestSource.GetNamespace(),
			ImagePullSecrets: set.NewStringSet(requestSource.GetImagePullSecrets()...),
		}
	}

	if _, err := s.enricher.EnrichImage(ctx, enrichmentContext, img); err != nil {
		log.Errorw("error enriching image", logging.ImageName(img.GetName().GetFullName()), logging.Err(err))
		// Purposefully, don't return here because we still need to save it into the DB so there is a reference
		// even if we weren't able to enrich it.
		return err
	}
	return nil
}

// ScanImage scans an image and returns the result
func (s *serviceImpl) ScanImage(ctx context.Context, request *v1.ScanImageRequest) (*storage.Image, error) {
	enrichmentCtx := enricher.EnrichmentContext{
		FetchOpt:  enricher.UseCachesIfPossible,
		Delegable: true,
	}
	if request.GetForce() {
		enrichmentCtx.FetchOpt = enricher.UseImageNamesRefetchCachedValues
	}

	if request.GetCluster() != "" {
		// The request indicates enrichment should be delegated to a specific cluster.
		clusterID, err := clusterUtil.GetClusterIDFromNameOrID(ctx, s.clusterSACHelper, request.GetCluster(), delegateScanPermissions)
		if err != nil {
			return nil, err
		}

		enrichmentCtx.ClusterID = clusterID
	}

	img, err := enricher.EnrichImageByName(ctx, s.enricher, enrichmentCtx, request.GetImageName())
	if err != nil {
		return nil, err
	}

	// Save the image
	img.Id = utils.GetSHA(img)
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
		// Aborted indicates the operation was aborted, typically due to a concurrency
		// issues.  Clients should retry by default on Aborted.
		s, err := status.New(codes.Aborted, err.Error()).WithDetails(&v1.ScanImageInternalResponseDetails_TooManyParallelScans{})
		if pkgUtils.ShouldErr(err) == nil {
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

	var hasErrors bool
	if request.Error != "" {
		// If errors occurred we continue processing so that the failed image scan may be saved in
		// the central datastore. Without this users would not have an indication that scans from
		// secured clusters are failing.
		hasErrors = true
		log.Warnw("Received image enrichment request with errors",
			logging.ImageName(request.GetImageName().GetFullName()), logging.Err(errors.New(request.GetError())))
	}

	var imgExists bool
	var existingImg *storage.Image
	forceSigVerificationUpdate := true
	forceScanUpdate := true
	imgID := request.GetImageId()
	// Always pull the image from the store if the ID != "" and rescan is not forced. Central will manage the reprocessing over the images.
	if imgID != "" && !request.GetForce() {
		existingImg, imgExists, err = s.datastore.GetImage(ctx, imgID)
		if err != nil {
			s.informScanWaiter(request.GetRequestId(), nil, err)
			return nil, err
		}
		// This is safe even if img is nil.
		scanTime := existingImg.GetScan().GetScanTime()

		// Check whether too much time has passed, if yes we have to do a signature verification update via the
		// enrichment pipeline to ensure we do not return stale data. Only do this when the image signature verification
		// feature is enabled. If no verification result is given, we can assume that the image doesn't have any
		// signatures associated with it.
		if imgExists && len(existingImg.GetSignatureVerificationData().GetResults()) > 0 {
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
		if imgExists && !forceScanUpdate && !forceSigVerificationUpdate {
			s.informScanWaiter(request.GetRequestId(), existingImg, nil)
			return internalScanRespFromImage(existingImg), nil
		}
	}

	img := &storage.Image{
		Id:   imgID,
		Name: request.GetImageName(),
		// 'Names' must be populated to enable cache hits in central AND sensor.
		Names:          buildNames(request.GetImageName(), request.GetMetadata()),
		Signature:      request.GetImageSignature(),
		Metadata:       request.GetMetadata(),
		Notes:          request.GetImageNotes(),
		Scan:           existingImg.GetScan(),
		IsClusterLocal: true,
	}

	if !hasErrors {
		if forceScanUpdate {
			if _, err := s.enricher.EnrichWithVulnerabilities(img, request.GetComponents(), request.GetNotes()); err != nil && imgExists {
				// In case we hit an error during enriching, and the image previously existed, we will _not_ upsert it in
				// central, since it could lead to us overriding an enriched image with a non-enriched image.
				s.informScanWaiter(request.GetRequestId(), nil, err)
				return nil, err
			}
		} else {
			// If we didn't update the scan, fill in the stats from existing image (if there is one).
			enricher.FillScanStats(img)
		}

		if forceSigVerificationUpdate {
			if _, err := s.enricher.EnrichWithSignatureVerificationData(ctx, img); err != nil && imgExists {
				s.informScanWaiter(request.GetRequestId(), nil, err)
				return nil, err
			}
		}
	}

	// Due to discrepancies in digests retrieved from metadata pulls and k8s, only upsert if the request
	// contained a digest. Do not upsert if a previous scan exists and there were errors with this scan
	// since it could lead to us overriding an enriched image with a non-enriched image.
	// Also do not upsert if there is a request id, this enables the caller to determine how to handle
	// the results (and also prevents multiple upserts for the same image).
	if imgID != "" && !(hasErrors && imgExists) && request.GetRequestId() == "" {
		_ = s.saveImage(img)
	}

	if hasErrors && request.GetRequestId() != "" {
		// Send an actual error to the waiter so that error handling can be done (ie: retry)
		// Without this a bare image is returned that will have notes such as MISSING_METADATA
		// which will be interpreted as a valid scan result.
		err = errors.New(request.GetError())
	}

	s.informScanWaiter(request.GetRequestId(), img, err)
	return internalScanRespFromImage(img), nil
}

// buildNames returns a slice containing the known image names from the various parameters.
func buildNames(srcImage *storage.ImageName, metadata *storage.ImageMetadata) []*storage.ImageName {
	names := []*storage.ImageName{srcImage}

	// Add a mirror name if exists.
	if mirror := metadata.GetDataSource().GetMirror(); mirror != "" {
		mirrorImg, err := utils.GenerateImageFromString(mirror)
		if err != nil {
			log.Warnw("Failed generating image from string",
				logging.String("mirror", mirror), logging.Err(err))
		} else {
			names = append(names, mirrorImg.GetName())
		}
	}

	return names
}

func (s *serviceImpl) informScanWaiter(reqID string, img *storage.Image, scanErr error) {
	if reqID == "" {
		// do nothing if request ID is missing (no waiter).
		return
	}

	if err := s.scanWaiterManager.Send(reqID, img, scanErr); err != nil {
		log.Errorw("Failed to send results to scan waiter",
			logging.String("request_id", reqID), logging.Err(err))
	}
}

func (s *serviceImpl) UpdateLocalScanStatusInternal(_ context.Context, req *v1.UpdateLocalScanStatusInternalRequest) (*v1.Empty, error) {
	log.Debugf("Received early delegated scan failure for %q: %q", req.GetRequestId(), req.GetError())

	s.informScanWaiter(req.GetRequestId(), nil, errors.New(req.GetError()))

	return &v1.Empty{}, nil
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

	enrichCtx := enricher.EnrichmentContext{
		FetchOpt:  enricher.IgnoreExistingImages,
		Delegable: true,
	}
	enrichmentResult, err := s.enricher.EnrichImage(ctx, enrichCtx, img)
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
	img.Id = utils.GetSHA(img)
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

func (s *serviceImpl) GetWatchedImages(ctx context.Context, _ *v1.Empty) (*v1.GetWatchedImagesResponse, error) {
	watchedImgs, err := s.watchedImages.GetAllWatchedImages(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetWatchedImagesResponse{WatchedImages: watchedImgs}, nil
}
