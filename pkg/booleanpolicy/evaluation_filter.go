package booleanpolicy

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sliceutils"
)

// isNonDefaultFilter returns true if the filter specifies any non-default
// filtering (i.e. the policy should pre-filter containers or image layers).
func isNonDefaultFilter(filter *storage.EvaluationFilter) bool {
	if filter == nil {
		return false
	}
	return len(filter.GetSkipContainerTypes()) > 0 ||
		filter.GetSkipImageLayers() != storage.SkipImageLayers_SKIP_NONE
}

// skipContainerTypeSet builds a lookup set from the repeated enum values.
func skipContainerTypeSet(filter *storage.EvaluationFilter) map[storage.SkipContainerType]struct{} {
	s := make(map[storage.SkipContainerType]struct{}, len(filter.GetSkipContainerTypes()))
	for _, ct := range filter.GetSkipContainerTypes() {
		s[ct] = struct{}{}
	}
	return s
}

// containerTypeMatchesSkip returns true if a container's ContainerType should
// be skipped according to the SkipContainerType enum value.
func containerTypeMatchesSkip(ct storage.ContainerType, skip storage.SkipContainerType) bool {
	switch skip {
	case storage.SkipContainerType_SKIP_INIT:
		return ct == storage.ContainerType_INIT
	default:
		return false
	}
}

// filterContainersBySkipTypes returns a shallow copy of the deployment with
// containers (and their corresponding images) removed if their type matches
// any entry in skip_container_types. The original deployment is not modified.
// If no containers are filtered, the original slices are returned as-is.
func filterContainersBySkipTypes(
	deployment *storage.Deployment,
	images []*storage.Image,
	filter *storage.EvaluationFilter,
) (*storage.Deployment, []*storage.Image) {
	if len(filter.GetSkipContainerTypes()) == 0 {
		return deployment, images
	}

	skipSet := skipContainerTypeSet(filter)

	shouldKeep := func(c *storage.Container) bool {
		for skip := range skipSet {
			if containerTypeMatchesSkip(c.GetType(), skip) {
				return false
			}
		}
		return true
	}

	containers := deployment.GetContainers()
	var filteredContainers []*storage.Container
	var filteredImages []*storage.Image

	for i, c := range containers {
		if shouldKeep(c) {
			filteredContainers = append(filteredContainers, c)
			if i < len(images) {
				filteredImages = append(filteredImages, images[i])
			}
		}
	}

	if len(filteredContainers) == len(containers) {
		return deployment, images
	}

	cloned := deployment.CloneVT()
	cloned.Containers = filteredContainers
	return cloned, filteredImages
}

// filterImagesByLayer returns a filtered copy of images where components and
// layers are pruned according to the SkipImageLayers setting.
//
// SKIP_BASE: removes components whose layer_index <= max_layer_index of the
// base image, keeping only application-layer components.
//
// SKIP_APP: removes components whose layer_index > max_layer_index of the
// base image, keeping only base-layer components.
//
// If BaseImageInfo is unavailable for an image, the image is returned unmodified
// (Option B: skip evaluation produces zero violations for that image since the
// augmented object won't match layer-specific criteria).
func filterImagesByLayer(
	images []*storage.Image,
	filter *storage.EvaluationFilter,
) []*storage.Image {
	skip := filter.GetSkipImageLayers()
	if skip == storage.SkipImageLayers_SKIP_NONE {
		return images
	}

	result := make([]*storage.Image, len(images))
	for i, img := range images {
		result[i] = filterSingleImageByLayer(img, skip)
	}
	return result
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

// applyEvaluationFilter returns an EnhancedDeployment with containers and
// images pre-filtered according to the EvaluationFilter. If the filter is
// the default (nil or all-defaults), the original is returned unchanged.
func applyEvaluationFilter(
	ed EnhancedDeployment,
	filter *storage.EvaluationFilter,
) EnhancedDeployment {
	if !isNonDefaultFilter(filter) {
		return ed
	}

	dep, imgs := filterContainersBySkipTypes(ed.Deployment, ed.Images, filter)
	imgs = filterImagesByLayer(imgs, filter)

	return EnhancedDeployment{
		Deployment:             dep,
		Images:                 imgs,
		NetworkPoliciesApplied: ed.NetworkPoliciesApplied,
	}
}
