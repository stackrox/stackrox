package filter

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sliceutils"
)

func newImageLayerFilter(skip storage.SkipImageLayers) *EvaluationFilter {
	if skip == storage.SkipImageLayers_SKIP_NONE {
		return nil
	}
	return &EvaluationFilter{
		isNonDefault: func() bool { return true },
		apply: func(deployment *storage.Deployment, images []*storage.Image) (*storage.Deployment, []*storage.Image) {
			result := make([]*storage.Image, len(images))
			for i, img := range images {
				result[i] = filterSingleImageByLayer(img, skip)
			}
			return deployment, result
		},
	}
}

func filterSingleImageByLayer(img *storage.Image, skip storage.SkipImageLayers) *storage.Image {
	if img == nil {
		return nil
	}
	baseInfos := img.GetBaseImageInfo()
	if len(baseInfos) == 0 {
		return img
	}
	maxBaseLayerIdx := baseInfos[0].GetMaxLayerIndex()
	layerFilter := func(idx int32) bool {
		switch skip {
		case storage.SkipImageLayers_SKIP_BASE:
			return idx > maxBaseLayerIdx
		case storage.SkipImageLayers_SKIP_APP:
			return idx <= maxBaseLayerIdx
		default:
			return true
		}
	}
	// Scan components (CVEs) are filtered only when scan data is present; the layer
	// (Dockerfile line) filtering below runs independently, since the base/app boundary
	// comes from BaseImageInfo, not the scan. This lets image-layer filtering work on
	// images that have layer metadata + base-image info but no vulnerability scan.
	scan := img.GetScan()
	var filteredComponents []*storage.EmbeddedImageScanComponent
	componentsChanged := false
	if scan != nil {
		filteredComponents = sliceutils.Filter(scan.GetComponents(), func(c *storage.EmbeddedImageScanComponent) bool {
			li, hasIdx := c.GetHasLayerIndex().(*storage.EmbeddedImageScanComponent_LayerIndex)
			if !hasIdx {
				return true
			}
			return layerFilter(li.LayerIndex)
		})
		componentsChanged = len(filteredComponents) != len(scan.GetComponents())
	}
	// Component LayerIndex values are kept as-is: they identify the original layer
	// position and are deliberately NOT rewritten to match the compacted slice built
	// below. filteredLayers packs the kept layers into consecutive positions, so the
	// returned Metadata.V1.Layers is intentionally not positionally aligned with
	// component LayerIndex -- consumers must not index Layers by a component's LayerIndex.
	layers := img.GetMetadata().GetV1().GetLayers()
	var filteredLayers []*storage.ImageLayer
	if len(layers) > 0 {
		for i, l := range layers {
			if layerFilter(int32(i)) {
				filteredLayers = append(filteredLayers, l)
			}
		}
	}
	layersChanged := len(filteredLayers) != len(layers)
	if !componentsChanged && !layersChanged {
		return img
	}
	cloned := img.CloneVT()
	if componentsChanged {
		cloned.Scan.Components = filteredComponents
	}
	if layersChanged && cloned.GetMetadata().GetV1() != nil {
		cloned.Metadata.V1.Layers = filteredLayers
	}
	return cloned
}
