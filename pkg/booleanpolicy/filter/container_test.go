package filter

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestContainerTypeFilter(t *testing.T) {
	regularContainer := &storage.Container{
		Name:  "app",
		Type:  storage.ContainerType_REGULAR,
		Image: &storage.ContainerImage{Name: &storage.ImageName{FullName: "app:latest"}},
	}
	initContainer := &storage.Container{
		Name:  "init",
		Type:  storage.ContainerType_INIT,
		Image: &storage.ContainerImage{Name: &storage.ImageName{FullName: "init:latest"}},
	}

	appImage := &storage.Image{Id: "app-img"}
	initImage := &storage.Image{Id: "init-img"}

	cases := map[string]struct {
		containers         []*storage.Container
		images             []*storage.Image
		skipTypes          []storage.SkipContainerType
		expectedContainers int
		expectedImageIDs   []string
	}{
		"skip init - only regular kept": {
			containers:         []*storage.Container{regularContainer, initContainer},
			images:             []*storage.Image{appImage, initImage},
			skipTypes:          []storage.SkipContainerType{storage.SkipContainerType_SKIP_INIT},
			expectedContainers: 1,
			expectedImageIDs:   []string{"app-img"},
		},
		"skip init - no init containers present": {
			containers:         []*storage.Container{regularContainer},
			images:             []*storage.Image{appImage},
			skipTypes:          []storage.SkipContainerType{storage.SkipContainerType_SKIP_INIT},
			expectedContainers: 1,
			expectedImageIDs:   []string{"app-img"},
		},
		"skip init - all init containers": {
			containers:         []*storage.Container{initContainer},
			images:             []*storage.Image{initImage},
			skipTypes:          []storage.SkipContainerType{storage.SkipContainerType_SKIP_INIT},
			expectedContainers: 0,
			expectedImageIDs:   nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newContainerTypeFilter(tc.skipTypes)
			assert.NotNil(t, f)
			assert.True(t, f.IsNonDefault())

			dep := &storage.Deployment{Containers: tc.containers}
			filteredDep, filteredImgs := f.Apply(dep, tc.images)

			assert.Len(t, filteredDep.GetContainers(), tc.expectedContainers)
			var imgIDs []string
			for _, img := range filteredImgs {
				imgIDs = append(imgIDs, img.GetId())
			}
			assert.Equal(t, tc.expectedImageIDs, imgIDs)

			// Original deployment should not be mutated.
			assert.Len(t, dep.GetContainers(), len(tc.containers))
		})
	}
}

func TestContainerTypeFilter_Empty(t *testing.T) {
	f := newContainerTypeFilter(nil)
	assert.Nil(t, f)
}
