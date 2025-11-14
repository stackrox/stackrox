package enrichment

import (
	"context"

	"github.com/stackrox/rox/central/image/datastore"
	imageV2Datastore "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	getImageContext = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Image)))
)

// enricherImpl enriches images with data from registries and scanners.
type enricherImpl struct {
	imageEnricher   enricher.ImageEnricher
	imageEnricherV2 enricher.ImageEnricherV2

	images   datastore.DataStore
	imagesV2 imageV2Datastore.DataStore
}

// EnrichDeployment enriches a deployment with data from registries and scanners.
func (e *enricherImpl) EnrichDeployment(ctx context.Context, enrichCtx enricher.EnrichmentContext, deployment *storage.Deployment) (images []*storage.Image, updatedIndices []int, pendingEnrichment bool, err error) {
	for i, c := range deployment.GetContainers() {
		var imgToProcess *storage.Image
		if (enrichCtx.FetchOnlyIfMetadataEmpty() || enrichCtx.FetchOnlyIfScanEmpty()) && c.GetImage().GetId() != "" {
			var img *storage.Image
			img, _, err = e.images.GetImage(getImageContext, c.GetImage().GetId())
			if err != nil {
				return
			}
			imgToProcess = img
		}
		if imgToProcess == nil {
			imgToProcess = types.ToImage(c.GetImage())
		}
		images = append(images, imgToProcess)
		// If an ID was found and the image is not pullable, then don't try to get metadata because it won't
		// be available
		if imgToProcess.GetId() != "" && imgToProcess.GetNotPullable() {
			continue
		}
		enrichmentResult, err := e.imageEnricher.EnrichImage(ctx, enrichCtx, imgToProcess)
		if err != nil {
			if env.AdministrationEventsAdHocScans.BooleanSetting() {
				log.Errorw("Enriching image",
					logging.ImageName(imgToProcess.GetName().GetFullName()),
					logging.Err(err),
					logging.Bool("ad_hoc", true),
				)
			} else {
				log.Error(err)
			}
		}
		if enrichmentResult.ImageUpdated {
			updatedIndices = append(updatedIndices, i)
		}
		if enrichmentResult.ScanResult == enricher.ScanTriggered {
			pendingEnrichment = true
		}
	}
	return
}

func (e *enricherImpl) EnrichDeploymentV2(ctx context.Context, enrichCtx enricher.EnrichmentContext, deployment *storage.Deployment) (images []*storage.ImageV2, updatedIndices []int, pendingEnrichment bool, err error) {
	for i, c := range deployment.GetContainers() {
		var imgToProcess *storage.ImageV2
		if (enrichCtx.FetchOnlyIfMetadataEmpty() || enrichCtx.FetchOnlyIfScanEmpty()) && c.GetImage().GetIdV2() != "" {
			var img *storage.ImageV2
			img, _, err = e.imagesV2.GetImage(getImageContext, c.GetImage().GetId())
			if err != nil {
				return
			}
			imgToProcess = img
		}
		if imgToProcess == nil {
			imgToProcess = types.ToImageV2(c.GetImage())
		}
		images = append(images, imgToProcess)
		// If an ID was found and the image is not pullable, then don't try to get metadata because it won't
		// be available
		if imgToProcess.GetId() != "" && imgToProcess.GetNotPullable() {
			continue
		}
		enrichmentResult, err := e.imageEnricherV2.EnrichImage(ctx, enrichCtx, imgToProcess)
		if err != nil {
			if env.AdministrationEventsAdHocScans.BooleanSetting() {
				log.Errorw("Enriching image",
					logging.ImageName(imgToProcess.GetName().GetFullName()),
					logging.Err(err),
					logging.Bool("ad_hoc", true),
				)
			} else {
				log.Error(err)
			}
		}
		if enrichmentResult.ImageUpdated {
			updatedIndices = append(updatedIndices, i)
		}
		if enrichmentResult.ScanResult == enricher.ScanTriggered {
			pendingEnrichment = true
		}
	}
	return
}
