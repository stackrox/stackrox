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
			filteredImages := make([]*storage.Image, len(images))
			for i, img := range images {
				filteredImages[i] = filterSingleImageByLayer(img, skip)
			}
			return deployment, filteredImages
		},
	}
}

func filterSingleImageByLayer(img *storage.Image, skip storage.SkipImageLayers) *storage.Image {
	if img == nil {
		return nil
	}
	// Without base-image info the boundary is unknown, so every layer is kept.
	// Components use max_layer_index: the manifest index of the last base content
	// blob, shared by every matched base. Dockerfile lines use max_history_index,
	// which also covers empty instructions such as USER. Those two indexes differ
	// when empty lines sit after the last base content blob.
	baseInfos := img.GetBaseImageInfo()
	if len(baseInfos) == 0 {
		return img
	}
	keepComponent := indexKeep(skip, baseInfos[0].GetMaxLayerIndex())
	// Zero is a valid history cut, so presence is the oneof, not the integer.
	// Several matched bases can carry different cuts; the greatest one still
	// includes base lines such as USER. An unset oneof on another base does not
	// cancel a set cut. When none are set, Dockerfile lines are not cut:
	// max_layer_index cannot classify those empty instructions.
	keepLine := func(int32) bool { return true }
	if cut, ok := greatestHistoryIndex(baseInfos); ok {
		keepLine = indexKeep(skip, cut)
	}
	// Scan components (CVEs) are filtered only when scan data is present; layer
	// filtering runs independently, since the boundary comes from BaseImageInfo, not the
	// scan.
	scan := img.GetScan()
	var filteredComponents []*storage.EmbeddedImageScanComponent
	componentsChanged := false
	if scan != nil {
		filteredComponents = sliceutils.Filter(scan.GetComponents(), func(c *storage.EmbeddedImageScanComponent) bool {
			li, hasIdx := c.GetHasLayerIndex().(*storage.EmbeddedImageScanComponent_LayerIndex)
			if !hasIdx {
				return true
			}
			return keepComponent(li.LayerIndex)
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
	for i, l := range layers {
		if keepLine(int32(i)) {
			filteredLayers = append(filteredLayers, l)
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

// indexKeep reports whether a layer index survives the base/app split at boundary.
// SKIP_BASE keeps application lines, which are strictly after the boundary.
// SKIP_APP keeps base lines, which are at the boundary and before it.
func indexKeep(skip storage.SkipImageLayers, boundary int32) func(int32) bool {
	return func(idx int32) bool {
		switch skip {
		case storage.SkipImageLayers_SKIP_BASE:
			return idx > boundary
		case storage.SkipImageLayers_SKIP_APP:
			return idx <= boundary
		default:
			return true
		}
	}
}

// greatestHistoryIndex returns the greatest set max_history_index.
// A base whose oneof is unset is skipped.
func greatestHistoryIndex(infos []*storage.BaseImageInfo) (int32, bool) {
	var cut int32
	found := false
	for _, info := range infos {
		if info.GetSetMaxHistoryIndex() == nil {
			continue
		}
		idx := info.GetMaxHistoryIndex()
		if !found {
			cut = idx
			found = true
			continue
		}
		cut = max(cut, idx)
	}
	return cut, found
}
