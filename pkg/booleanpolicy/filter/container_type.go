package filter

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

func newContainerTypeFilter(skipTypes []storage.SkipContainerType) *EvaluationFilter {
	skip := set.NewSet[storage.ContainerType]()
	for _, skipType := range skipTypes {
		switch skipType {
		case storage.SkipContainerType_SKIP_INIT:
			skip.Add(storage.ContainerType_INIT)
		}
	}
	if skip.IsEmpty() {
		return nil
	}
	return &EvaluationFilter{
		isNonDefault: func() bool { return true },
		apply: func(dep *storage.Deployment, imgs []*storage.Image) (*storage.Deployment, []*storage.Image) {
			var filteredContainers []*storage.Container
			var filteredImgs []*storage.Image
			hasSkipped := false
			for i, c := range dep.GetContainers() {
				if skip.Contains(c.GetType()) {
					hasSkipped = true
					continue
				}
				filteredContainers = append(filteredContainers, c)
				if i < len(imgs) {
					filteredImgs = append(filteredImgs, imgs[i])
				}
			}
			if !hasSkipped {
				return dep, imgs
			}
			filtered := dep.CloneVT()
			filtered.Containers = filteredContainers
			return filtered, filteredImgs
		},
	}
}
