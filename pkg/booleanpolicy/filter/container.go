package filter

import "github.com/stackrox/rox/generated/storage"

func newContainerTypeFilter(skipTypes []storage.SkipContainerType) *EvaluationFilter {
	if len(skipTypes) == 0 {
		return nil
	}

	skipSet := make(map[storage.SkipContainerType]struct{}, len(skipTypes))
	for _, ct := range skipTypes {
		skipSet[ct] = struct{}{}
	}

	shouldSkip := func(c *storage.Container) bool {
		for skip := range skipSet {
			if containerTypeMatchesSkip(c.GetType(), skip) {
				return true
			}
		}
		return false
	}

	return &EvaluationFilter{
		isNonDefault: func() bool { return true },
		apply: func(deployment *storage.Deployment, images []*storage.Image) (*storage.Deployment, []*storage.Image) {
			containers := deployment.GetContainers()
			var filteredContainers []*storage.Container
			var filteredImages []*storage.Image

			for i, c := range containers {
				if !shouldSkip(c) {
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
		},
	}
}

func containerTypeMatchesSkip(ct storage.ContainerType, skip storage.SkipContainerType) bool {
	switch skip {
	case storage.SkipContainerType_SKIP_INIT:
		return ct == storage.ContainerType_INIT
	default:
		return false
	}
}
