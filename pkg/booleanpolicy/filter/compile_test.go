package filter

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileEvaluationFilter_AllFiltersAreNonDefault(t *testing.T) {
	t.Setenv(features.EvaluationFilter.EnvVar(), "true")

	for _, proto := range []*storage.EvaluationFilter{
		nil,
		{},
	} {
		filters := CompileEvaluationFilter(proto)
		for i, f := range filters {
			assert.Truef(t, f.IsNonDefault(), "filter at index %d should be non-default", i)
		}
	}
}

func TestCompileEvaluationFilter_DisabledFeatureReturnsNil(t *testing.T) {
	t.Setenv(features.EvaluationFilter.EnvVar(), "false")
	assert.Nil(t, CompileEvaluationFilter(&storage.EvaluationFilter{}))
}

func TestCompileEvaluationFilter_NoSkipTypes_ReturnsNil(t *testing.T) {
	t.Setenv(features.EvaluationFilter.EnvVar(), "true")
	assert.Nil(t, CompileEvaluationFilter(nil))
	assert.Nil(t, CompileEvaluationFilter(&storage.EvaluationFilter{}))
}

func TestCompileEvaluationFilter_ContainerTypeFilter(t *testing.T) {
	tests := map[string]struct {
		skipTypes          []storage.ContainerType
		containers         []*storage.Container
		images             []*storage.Image
		expectedContainers []string
		expectedImages     []string
		expectSamePointer  bool
	}{
		"skip init containers": {
			skipTypes: []storage.ContainerType{storage.ContainerType_INIT},
			containers: []*storage.Container{
				{Name: "init-setup", Type: storage.ContainerType_INIT},
				{Name: "app", Type: storage.ContainerType_REGULAR},
				{Name: "init-db", Type: storage.ContainerType_INIT},
				{Name: "sidecar", Type: storage.ContainerType_REGULAR},
			},
			images: []*storage.Image{
				{Id: "init-setup-img"},
				{Id: "app-img"},
				{Id: "init-db-img"},
				{Id: "sidecar-img"},
			},
			expectedContainers: []string{"app", "sidecar"},
			expectedImages:     []string{"app-img", "sidecar-img"},
		},
		"skip init but none present returns original": {
			skipTypes: []storage.ContainerType{storage.ContainerType_INIT},
			containers: []*storage.Container{
				{Name: "app", Type: storage.ContainerType_REGULAR},
			},
			images:            []*storage.Image{{Id: "app-img"}},
			expectSamePointer: true,
		},
		"skip regular containers": {
			skipTypes: []storage.ContainerType{storage.ContainerType_REGULAR},
			containers: []*storage.Container{
				{Name: "init-setup", Type: storage.ContainerType_INIT},
				{Name: "app", Type: storage.ContainerType_REGULAR},
			},
			images: []*storage.Image{
				{Id: "init-img"},
				{Id: "app-img"},
			},
			expectedContainers: []string{"init-setup"},
			expectedImages:     []string{"init-img"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv(features.EvaluationFilter.EnvVar(), "true")

			filters := CompileEvaluationFilter(&storage.EvaluationFilter{
				SkipContainerTypes: tc.skipTypes,
			})
			require.Len(t, filters, 1)
			assert.True(t, filters[0].IsNonDefault())

			dep := &storage.Deployment{Containers: tc.containers}
			resultDep, resultImgs := filters[0].Apply(dep, tc.images)

			if tc.expectSamePointer {
				assert.True(t, dep == resultDep, "expected same deployment pointer when no filtering needed")
				return
			}

			require.Len(t, resultDep.GetContainers(), len(tc.expectedContainers))
			for i, name := range tc.expectedContainers {
				assert.Equal(t, name, resultDep.GetContainers()[i].GetName())
			}
			require.Len(t, resultImgs, len(tc.expectedImages))
			for i, id := range tc.expectedImages {
				assert.Equal(t, id, resultImgs[i].GetId())
			}

			// Original not mutated.
			assert.Len(t, dep.GetContainers(), len(tc.containers))
		})
	}
}
