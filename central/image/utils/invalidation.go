package utils

import (
	"context"
	"fmt"
	"time"

	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	imageV2Datastore "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"google.golang.org/protobuf/proto"
)

const perConnectionTimeout = 5 * time.Second

var log = logging.LoggerForModule()

// UpdateImageCaches resolves clusters deploying the given image and sends
// UpdatedImage + AC-only InvalidateImageCache to each. Clusters are resolved
// by image name (tag-based) when a tag is available, falling back to
// ImageIDV2 (V2 mode) or SHA digest (V1 mode) for digest-only references.
// Intended to be called in a goroutine after image enrichment persists.
func UpdateImageCaches(
	connMgr connection.Manager,
	deploymentDS deploymentDatastore.DataStore,
	imagesV2DS imageV2Datastore.DataStore,
	image *storage.Image,
	key *central.ImageKey,
) {
	if key == nil {
		return
	}

	ctx := sac.WithAllAccess(context.Background())

	clusterIDs, err := resolveClusterIDs(ctx, deploymentDS, image.GetName(), key)
	if err != nil {
		log.Warnw("Failed to resolve clusters for image cache invalidation",
			logging.Err(err),
		)
		return
	}
	if len(clusterIDs) == 0 {
		log.Debugw("No clusters found deploying image, skipping cache update",
			logging.String("image_id", key.GetImageId()),
			logging.String("image_id_v2", key.GetImageIdV2()),
			logging.String("image_full_name", key.GetImageFullName()),
		)
		return
	}

	imageSHA := key.GetImageId()
	imageToSend := proto.Clone(image).(*storage.Image)
	if features.FlattenImageData.Enabled() && !connMgr.AllSensorsHaveCapability(centralsensor.FlattenImageData) {
		if imageSHA != "" {
			allNames, err := imagesV2DS.GetImageNames(ctx, imageSHA)
			if err != nil {
				log.Warnw("Failed to fetch all names for image digest",
					logging.String("image_sha", imageSHA),
					logging.Err(err),
				)
			} else if len(allNames) > 0 {
				imageToSend.Names = sliceutils.Unique(append(imageToSend.Names, allNames...))
			}
		}
	}

	// UpdatedImage is sent first to warm Sensor's image cache with the
	// freshly enriched image. InvalidateImageCache is sent second with
	// AdmissionControllerOnly=true so the AC re-evaluates using the data
	// Sensor already has cached — without evicting Sensor's own image
	// cache, which would cause redundant re-fetches.
	updateMsg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_UpdatedImage{UpdatedImage: imageToSend},
	}
	invalidateMsg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_InvalidateImageCache{
			InvalidateImageCache: &central.InvalidateImageCache{
				ImageKeys:               []*central.ImageKey{key},
				AdmissionControllerOnly: true,
			},
		},
	}

	for _, clusterID := range clusterIDs {
		conn := connMgr.GetConnection(clusterID)
		if conn == nil || !conn.HasCapability(centralsensor.TargetedImageCacheInvalidation) {
			log.Debugw("Skipping cluster: no connection or missing capability",
				logging.String("dst_cluster", clusterID),
				logging.String("image_sha", imageSHA),
			)
			continue
		}

		sendCtx, cancel := context.WithTimeout(ctx, perConnectionTimeout)
		if err := conn.InjectMessage(sendCtx, updateMsg); err != nil {
			cancel()
			log.Warnw("Failed to send UpdatedImage for cache warming",
				logging.String("dst_cluster", clusterID),
				logging.String("image_sha", imageSHA),
				logging.Err(err),
			)
			continue
		}
		cancel()

		sendCtx, cancel = context.WithTimeout(ctx, perConnectionTimeout)
		if err := conn.InjectMessage(sendCtx, invalidateMsg); err != nil {
			log.Warnw("Failed to send AC-only InvalidateImageCache",
				logging.String("dst_cluster", clusterID),
				logging.String("image_sha", imageSHA),
				logging.Err(err),
			)
		} else {
			log.Debugw("Sent UpdatedImage + AC-only InvalidateImageCache",
				logging.String("dst_cluster", clusterID),
				logging.String("image_sha", imageSHA),
			)
		}
		cancel()
	}
}

// resolveClusterIDs finds clusters deploying the given image. It prefers
// searching by image name (registry/remote:tag) via the deployment datastore,
// which avoids manifest-vs-config digest mismatches. For digest-only
// references (no tag), it falls back to ID-based search.
func resolveClusterIDs(
	ctx context.Context,
	deploymentDS deploymentDatastore.DataStore,
	imageName *storage.ImageName,
	key *central.ImageKey,
) ([]string, error) {
	if tag := imageName.GetTag(); tag != "" {
		searchName := fmt.Sprintf("%s/%s:%s", imageName.GetRegistry(), imageName.GetRemote(), tag)
		return resolveClusterIDsByImageName(ctx, deploymentDS, searchName)
	}
	// Digest-only reference: fall back to ID-based search.
	if features.FlattenImageData.Enabled() {
		return resolveClusterIDsByImageID(ctx, deploymentDS, search.ImageID, key.GetImageIdV2())
	}
	return resolveClusterIDsByImageID(ctx, deploymentDS, search.ImageSHA, key.GetImageId())
}

func resolveClusterIDsByImageName(
	ctx context.Context,
	deploymentDS deploymentDatastore.DataStore,
	imageFullName string,
) ([]string, error) {
	if imageFullName == "" {
		return nil, nil
	}
	q := search.NewQueryBuilder().
		AddExactMatches(search.ImageName, imageFullName).
		ProtoQuery()

	results, err := deploymentDS.GetContainerImageViews(ctx, q)
	if err != nil {
		return nil, err
	}

	clusterIDs := set.NewStringSet()
	for _, result := range results {
		clusterIDs.AddAll(result.GetClusterIDs()...)
	}
	return clusterIDs.AsSlice(), nil
}

func resolveClusterIDsByImageID(
	ctx context.Context,
	deploymentDS deploymentDatastore.DataStore,
	field search.FieldLabel,
	id string,
) ([]string, error) {
	if id == "" {
		return nil, nil
	}
	q := search.NewQueryBuilder().
		AddExactMatches(field, id).
		ProtoQuery()

	results, err := deploymentDS.GetContainerImageViews(ctx, q)
	if err != nil {
		return nil, err
	}

	clusterIDs := set.NewStringSet()
	for _, result := range results {
		clusterIDs.AddAll(result.GetClusterIDs()...)
	}
	return clusterIDs.AsSlice(), nil
}
