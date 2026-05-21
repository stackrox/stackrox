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

	scan := img.GetScan()
	if scan == nil {
		return img
	}

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

	filteredComponents := sliceutils.Filter(scan.GetComponents(), func(c *storage.EmbeddedImageScanComponent) bool {
		li, hasIdx := c.GetHasLayerIndex().(*storage.EmbeddedImageScanComponent_LayerIndex)
		if !hasIdx {
			return true
		}
		return layerFilter(li.LayerIndex)
	})

	layers := img.GetMetadata().GetV1().GetLayers()
	var filteredLayers []*storage.ImageLayer
	if len(layers) > 0 {
		for i, l := range layers {
			if layerFilter(int32(i)) {
				filteredLayers = append(filteredLayers, l)
			}
		}
	}

	componentsChanged := len(filteredComponents) != len(scan.GetComponents())
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
