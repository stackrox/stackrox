// Package filtercompilers registers domain-specific FilterPlugins with the
// booleanpolicy engine. Each plugin owns one field of EvaluationFilter and
// produces a ValueFilterFactory for the matcher kinds it supports.
//
// Importing this package (typically via a blank import) is sufficient to register
// all plugins; no explicit initialization is required.
package filtercompilers

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

func init() {
	booleanpolicy.RegisterFilterPlugin[storage.SkipImageLayers](
		imageLayerPlugin,
		map[booleanpolicy.MatcherKind]booleanpolicy.ContextExtractor{
			booleanpolicy.DeploymentKind:  extractImagesFromDeployment,
			booleanpolicy.ProcessKind:     extractImagesFromDeployment,
			booleanpolicy.NetworkFlowKind: extractImagesFromDeployment,
			booleanpolicy.FileAccessKind:  extractImagesFromDeployment,
			booleanpolicy.ImageKind:       extractStandaloneImage,
		},
	)
}

// extractImagesFromDeployment extracts the images slice from an EnhancedDeployment context.
// Used by all deployment-context matchers (Deployment, Process, NetworkFlow, FileAccess).
func extractImagesFromDeployment(matchData interface{}) interface{} {
	return matchData.(booleanpolicy.EnhancedDeployment).Images
}

// extractStandaloneImage wraps the single *storage.Image from MatchImage into a slice
// so the factory always receives []*storage.Image regardless of the matcher kind.
func extractStandaloneImage(matchData interface{}) interface{} {
	return []*storage.Image{matchData.(*storage.Image)}
}

// imageLayerPlugin owns the EvaluationFilter.skip_image_layers field.
// Returns nil when skip_image_layers is SKIP_NONE — no filtering needed.
// The returned factory receives []*storage.Image already extracted by the ContextExtractor.
func imageLayerPlugin(f *storage.EvaluationFilter) booleanpolicy.ValueFilterFactory {
	skip := f.GetSkipImageLayers()
	if skip == storage.SkipImageLayers_SKIP_NONE {
		return nil
	}
	log.Infof("imageLayerPlugin: compiling filter with skip_image_layers=%v", skip)
	return func(data interface{}) pathutil.ValueFilter {
		images, ok := data.([]*storage.Image)
		if !ok || len(images) == 0 {
			return nil
		}
		return imageLayerValueFilter(images, skip)
	}
}

// imageLayerValueFilter returns a per-invocation ValueFilter that pre-terminates
// slice elements that fall outside the requested layer boundary.
//
// Two values are cached per slice traversal: the image (via imageForPath) and the
// resolved maxLayerIndex. Since the same AugmentedValue is passed for every i within
// one takeSliceMetaStep loop, both lookups run once per slice rather than per element.
// The closure is fresh per Match* call so the cache is never shared across concurrent calls.
func imageLayerValueFilter(images []*storage.Image, skip storage.SkipImageLayers) pathutil.ValueFilter {
	var cachedSlice pathutil.AugmentedValue
	var cachedMaxLayer int32
	var cachedHasMax bool

	return func(v pathutil.AugmentedValue, i int) bool {
		if v != cachedSlice {
			cachedSlice = v
			if img, ok := imageForPath(images, v.PathFromRoot()); ok {
				cachedMaxLayer, cachedHasMax = maxLayerIndex(img)
			} else {
				cachedMaxLayer, cachedHasMax = 0, false
			}
		}
		elem := v.Underlying().Index(i).Interface()
		switch e := elem.(type) {
		case *storage.EmbeddedImageScanComponent:
			return keepComponent(e, cachedMaxLayer, cachedHasMax, skip)
		case *storage.ImageLayer:
			return keepLayerByIndex(i, cachedMaxLayer, cachedHasMax, skip)
		default:
			// Not a filtered element type — pass through unchanged.
			return true
		}
	}
}

// keepComponent returns false when the component should be excluded.
// SKIP_BASE: exclude base-image layers, keep application layers.
// SKIP_APP:  exclude application layers, keep base-image layers.
// REQ-12: when hasMax is false the boundary is unknown; isBaseComponent returns false (treat as app).
func keepComponent(comp *storage.EmbeddedImageScanComponent, maxLayer int32, hasMax bool, skip storage.SkipImageLayers) bool {
	base := isBaseComponent(comp, maxLayer, hasMax)
	if skip == storage.SkipImageLayers_SKIP_APP {
		return base
	}
	return !base // SKIP_BASE
}

// keepLayerByIndex returns false when the Dockerfile layer at position i should be excluded.
// The slice position i equals the Dockerfile layer index j directly.
// REQ-12: when hasMax is false the boundary is unknown; isBaseLayer returns false (treat as app).
func keepLayerByIndex(i int, maxLayer int32, hasMax bool, skip storage.SkipImageLayers) bool {
	base := isBaseLayer(i, maxLayer, hasMax)
	if skip == storage.SkipImageLayers_SKIP_APP {
		return base
	}
	return !base // SKIP_BASE
}

// isBaseComponent returns true when the component belongs to the base image.
// Returns false when hasMax is false, or LayerIndex is absent (REQ-12).
func isBaseComponent(comp *storage.EmbeddedImageScanComponent, maxLayer int32, hasMax bool) bool {
	if !hasMax || comp.GetHasLayerIndex() == nil {
		return false
	}
	return comp.GetLayerIndex() <= maxLayer
}

// isBaseLayer returns true when the Dockerfile layer at position i belongs to the base image.
// Returns false when hasMax is false (REQ-12).
func isBaseLayer(i int, maxLayer int32, hasMax bool) bool {
	return hasMax && int32(i) <= maxLayer
}

// imageForPath resolves the image for the current traversal position.
// Deployment paths: extracts container index from Containers/[j] steps.
// Standalone image (MatchImage): falls back to images[0] when len == 1.
func imageForPath(images []*storage.Image, path *pathutil.Path) (*storage.Image, bool) {
	steps := path.Steps()
	for k, s := range steps {
		if s.Field() == "Containers" && k+1 < len(steps) {
			if idx := steps[k+1].Index(); idx >= 0 && idx < len(images) {
				return images[idx], true
			}
		}
	}
	if len(images) == 1 {
		return images[0], true
	}
	return nil, false
}

// maxLayerIndex returns the MaxLayerIndex from the image BaseImageInfo.
// MaxLayerIndex is the same across all BaseImageInfo entries; index 0 is used.
func maxLayerIndex(img *storage.Image) (int32, bool) {
	infos := img.GetBaseImageInfo()
	if len(infos) == 0 {
		return 0, false
	}
	return infos[0].GetMaxLayerIndex(), true
}
