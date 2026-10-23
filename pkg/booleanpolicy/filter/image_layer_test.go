package filter

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewImageLayerFilter_SkipNone_ReturnsNil(t *testing.T) {
	filter := newImageLayerFilter(storage.SkipImageLayers_SKIP_NONE)
	assert.Nil(t, filter)
}

func TestFilterSingleImageByLayer_NilImage_ReturnsNil(t *testing.T) {
	result := filterSingleImageByLayer(nil, storage.SkipImageLayers_SKIP_BASE)
	assert.Nil(t, result)
}

func TestFilterSingleImageByLayer_MissingBaseImageInfo_ReturnsUnchanged(t *testing.T) {
	img := &storage.Image{
		Id: "test-img",
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{Name: "comp1", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 5}},
			},
		},
	}

	// Both SKIP_BASE and SKIP_APP should return the same pointer when BaseImageInfo is missing
	resultBase := filterSingleImageByLayer(img, storage.SkipImageLayers_SKIP_BASE)
	assert.True(t, img == resultBase, "expected same pointer for SKIP_BASE when BaseImageInfo is missing")

	resultApp := filterSingleImageByLayer(img, storage.SkipImageLayers_SKIP_APP)
	assert.True(t, img == resultApp, "expected same pointer for SKIP_APP when BaseImageInfo is missing")
}

func TestFilterSingleImageByLayer_EmptyBaseImageInfo_ReturnsUnchanged(t *testing.T) {
	img := &storage.Image{
		Id:            "test-img",
		BaseImageInfo: []*storage.BaseImageInfo{},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{Name: "comp1", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 5}},
			},
		},
	}

	resultBase := filterSingleImageByLayer(img, storage.SkipImageLayers_SKIP_BASE)
	assert.True(t, img == resultBase, "expected same pointer for SKIP_BASE when BaseImageInfo is empty")

	resultApp := filterSingleImageByLayer(img, storage.SkipImageLayers_SKIP_APP)
	assert.True(t, img == resultApp, "expected same pointer for SKIP_APP when BaseImageInfo is empty")
}

func TestFilterSingleImageByLayer_NilScan_NoLayers_ReturnsUnchanged(t *testing.T) {
	// When scan is nil AND there are no layers, the image is returned unchanged.
	// Layer filtering is decoupled from scan presence, but with no layers there's nothing to filter.
	img := &storage.Image{
		Id: "test-img",
		BaseImageInfo: []*storage.BaseImageInfo{
			{MaxLayerIndex: 2},
		},
	}

	result := filterSingleImageByLayer(img, storage.SkipImageLayers_SKIP_BASE)
	assert.True(t, img == result, "expected same pointer when Scan is nil and no layers present")
}

func TestFilterSingleImageByLayer_NilScan_StillFiltersLayers(t *testing.T) {
	// Layer filtering runs even when scan is nil, since the base/app boundary
	// comes from BaseImageInfo, not the scan. This lets layer filtering work on
	// images that have layer metadata + base-image info but no vulnerability scan.
	img := &storage.Image{
		Id: "test-img",
		BaseImageInfo: []*storage.BaseImageInfo{
			{MaxLayerIndex: 4}, // Base layers: 0-4; App layers: 5-7
		},
		Scan: nil, // No scan data
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Layers: []*storage.ImageLayer{
					{Instruction: "FROM", Value: "base-0"},
					{Instruction: "RUN", Value: "base-1"},
					{Instruction: "RUN", Value: "base-2"},
					{Instruction: "RUN", Value: "base-3"},
					{Instruction: "RUN", Value: "base-4"},
					{Instruction: "COPY", Value: "app-5"},
					{Instruction: "RUN", Value: "app-6"},
					{Instruction: "CMD", Value: "app-7"},
				},
			},
		},
	}

	result := filterSingleImageByLayer(img, storage.SkipImageLayers_SKIP_BASE)
	require.NotNil(t, result)

	// Verify result is a clone (original unchanged)
	assert.False(t, img == result, "should clone when filtering layers")
	assert.Len(t, img.GetMetadata().GetV1().GetLayers(), 8, "original should have 8 layers")

	// Verify layers were filtered to app layers only (indices 5-7)
	resultLayers := result.GetMetadata().GetV1().GetLayers()
	require.Len(t, resultLayers, 3, "should have 3 app layers after filtering")

	// Assert by content (behavior, not constants)
	assert.Equal(t, "COPY", resultLayers[0].GetInstruction())
	assert.Contains(t, resultLayers[0].GetValue(), "app", "first kept layer should be an app layer")

	assert.Equal(t, "RUN", resultLayers[1].GetInstruction())
	assert.Contains(t, resultLayers[1].GetValue(), "app", "second kept layer should be an app layer")

	assert.Equal(t, "CMD", resultLayers[2].GetInstruction())
	assert.Contains(t, resultLayers[2].GetValue(), "app", "third kept layer should be an app layer")

	// Verify scan is still nil (not populated by filtering)
	assert.Nil(t, result.GetScan(), "scan should remain nil")
}

func TestFilterSingleImageByLayer_MultipleBaseImageInfo_UsesFirst(t *testing.T) {
	img := &storage.Image{
		Id: "test-img",
		BaseImageInfo: []*storage.BaseImageInfo{
			{MaxLayerIndex: 2},
			{MaxLayerIndex: 5}, // This should be ignored
		},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{Name: "base-comp", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 1}},
				{Name: "boundary-comp", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 2}},
				{Name: "app-comp", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 3}},
			},
		},
	}

	// SKIP_BASE should keep only components with index > 2 (using first BaseImageInfo)
	result := filterSingleImageByLayer(img, storage.SkipImageLayers_SKIP_BASE)
	require.NotNil(t, result)
	assert.False(t, img == result, "should clone when filtering")
	require.Len(t, result.GetScan().GetComponents(), 1)
	assert.Equal(t, "app-comp", result.GetScan().GetComponents()[0].GetName())
}

func TestFilterSingleImageByLayer_SkipBase(t *testing.T) {
	tests := map[string]struct {
		maxBaseLayerIdx int32
		components      []*storage.EmbeddedImageScanComponent
		expectedNames   []string
	}{
		"keeps app layers only": {
			maxBaseLayerIdx: 2,
			components: []*storage.EmbeddedImageScanComponent{
				{Name: "base-1", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0}},
				{Name: "base-2", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 2}},
				{Name: "app-1", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 3}},
				{Name: "app-2", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 5}},
			},
			expectedNames: []string{"app-1", "app-2"},
		},
		"boundary component is base (dropped)": {
			maxBaseLayerIdx: 2,
			components: []*storage.EmbeddedImageScanComponent{
				{Name: "boundary", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 2}},
				{Name: "app", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 3}},
			},
			expectedNames: []string{"app"},
		},
		"keeps components without layer index": {
			maxBaseLayerIdx: 2,
			components: []*storage.EmbeddedImageScanComponent{
				{Name: "base", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 1}},
				{Name: "no-index", HasLayerIndex: nil},
				{Name: "app", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 3}},
			},
			expectedNames: []string{"no-index", "app"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			img := &storage.Image{
				Id: "test-img",
				BaseImageInfo: []*storage.BaseImageInfo{
					{MaxLayerIndex: tc.maxBaseLayerIdx},
				},
				Scan: &storage.ImageScan{
					Components: tc.components,
				},
			}

			result := filterSingleImageByLayer(img, storage.SkipImageLayers_SKIP_BASE)
			require.NotNil(t, result)
			require.Len(t, result.GetScan().GetComponents(), len(tc.expectedNames))
			for i, name := range tc.expectedNames {
				assert.Equal(t, name, result.GetScan().GetComponents()[i].GetName())
			}

			// Verify original not mutated
			assert.Len(t, img.GetScan().GetComponents(), len(tc.components))
		})
	}
}

func TestFilterSingleImageByLayer_SkipApp(t *testing.T) {
	tests := map[string]struct {
		maxBaseLayerIdx int32
		components      []*storage.EmbeddedImageScanComponent
		expectedNames   []string
	}{
		"keeps base layers only": {
			maxBaseLayerIdx: 2,
			components: []*storage.EmbeddedImageScanComponent{
				{Name: "base-1", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 0}},
				{Name: "base-2", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 2}},
				{Name: "app-1", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 3}},
				{Name: "app-2", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 5}},
			},
			expectedNames: []string{"base-1", "base-2"},
		},
		"boundary component is base (kept)": {
			maxBaseLayerIdx: 2,
			components: []*storage.EmbeddedImageScanComponent{
				{Name: "boundary", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 2}},
				{Name: "app", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 3}},
			},
			expectedNames: []string{"boundary"},
		},
		"keeps components without layer index": {
			maxBaseLayerIdx: 2,
			components: []*storage.EmbeddedImageScanComponent{
				{Name: "base", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 1}},
				{Name: "no-index", HasLayerIndex: nil},
				{Name: "app", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 3}},
			},
			expectedNames: []string{"base", "no-index"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			img := &storage.Image{
				Id: "test-img",
				BaseImageInfo: []*storage.BaseImageInfo{
					{MaxLayerIndex: tc.maxBaseLayerIdx},
				},
				Scan: &storage.ImageScan{
					Components: tc.components,
				},
			}

			result := filterSingleImageByLayer(img, storage.SkipImageLayers_SKIP_APP)
			require.NotNil(t, result)
			require.Len(t, result.GetScan().GetComponents(), len(tc.expectedNames))
			for i, name := range tc.expectedNames {
				assert.Equal(t, name, result.GetScan().GetComponents()[i].GetName())
			}

			// Verify original not mutated
			assert.Len(t, img.GetScan().GetComponents(), len(tc.components))
		})
	}
}

func TestFilterSingleImageByLayer_LayersFiltering(t *testing.T) {
	tests := map[string]struct {
		skip                 storage.SkipImageLayers
		maxBaseLayerIdx      int32
		layers               []*storage.ImageLayer
		expectedInstructions []string
	}{
		"SKIP_BASE filters Dockerfile layers": {
			skip:            storage.SkipImageLayers_SKIP_BASE,
			maxBaseLayerIdx: 2,
			layers: []*storage.ImageLayer{
				{Instruction: "FROM", Value: "base"},     // index 0
				{Instruction: "RUN", Value: "base-cmd"},  // index 1
				{Instruction: "RUN", Value: "base-cmd2"}, // index 2
				{Instruction: "COPY", Value: "app"},      // index 3
				{Instruction: "RUN", Value: "app-cmd"},   // index 4
			},
			expectedInstructions: []string{"COPY", "RUN"}, // indices 3 and 4
		},
		"SKIP_APP filters Dockerfile layers": {
			skip:            storage.SkipImageLayers_SKIP_APP,
			maxBaseLayerIdx: 2,
			layers: []*storage.ImageLayer{
				{Instruction: "FROM", Value: "base"},     // index 0
				{Instruction: "RUN", Value: "base-cmd"},  // index 1
				{Instruction: "RUN", Value: "base-cmd2"}, // index 2
				{Instruction: "COPY", Value: "app"},      // index 3
				{Instruction: "RUN", Value: "app-cmd"},   // index 4
			},
			expectedInstructions: []string{"FROM", "RUN", "RUN"}, // indices 0, 1, 2
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			img := &storage.Image{
				Id: "test-img",
				BaseImageInfo: []*storage.BaseImageInfo{
					{MaxLayerIndex: tc.maxBaseLayerIdx},
				},
				Scan: &storage.ImageScan{
					Components: []*storage.EmbeddedImageScanComponent{},
				},
				Metadata: &storage.ImageMetadata{
					V1: &storage.V1Metadata{
						Layers: tc.layers,
					},
				},
			}

			result := filterSingleImageByLayer(img, tc.skip)
			require.NotNil(t, result)
			resultLayers := result.GetMetadata().GetV1().GetLayers()
			require.Len(t, resultLayers, len(tc.expectedInstructions))

			// Assert the exact layers survive, not just the count
			for i, expectedInstruction := range tc.expectedInstructions {
				assert.Equal(t, expectedInstruction, resultLayers[i].GetInstruction(),
					"layer %d should have instruction %s", i, expectedInstruction)
			}

			// Verify original not mutated
			assert.Len(t, img.GetMetadata().GetV1().GetLayers(), len(tc.layers))
		})
	}
}

func TestFilterSingleImageByLayer_NilMetadata_NoPanic(t *testing.T) {
	img := &storage.Image{
		Id: "test-img",
		BaseImageInfo: []*storage.BaseImageInfo{
			{MaxLayerIndex: 2},
		},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{Name: "base-comp", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 1}},
				{Name: "app-comp", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 3}},
			},
		},
		Metadata: nil, // Explicitly nil
	}

	// Should not panic and should still filter components correctly
	result := filterSingleImageByLayer(img, storage.SkipImageLayers_SKIP_BASE)
	require.NotNil(t, result, "result should not be nil")
	require.NotNil(t, result.GetScan(), "result scan should not be nil")

	// Only app-comp should survive (index 3 > maxBaseLayerIdx 2)
	resultComponents := result.GetScan().GetComponents()
	require.Len(t, resultComponents, 1)
	assert.Equal(t, "app-comp", resultComponents[0].GetName())

	// Verify Metadata is still nil (layers filtering is a no-op)
	assert.Nil(t, result.GetMetadata())

	// Verify original not mutated
	assert.Len(t, img.GetScan().GetComponents(), 2)
}

func TestFilterSingleImageByLayer_NoOpReturnsOriginalPointer(t *testing.T) {
	tests := map[string]struct {
		skip storage.SkipImageLayers
		img  *storage.Image
	}{
		"all components kept - no filtering needed": {
			skip: storage.SkipImageLayers_SKIP_BASE,
			img: &storage.Image{
				Id: "test-img",
				BaseImageInfo: []*storage.BaseImageInfo{
					{MaxLayerIndex: 2},
				},
				Scan: &storage.ImageScan{
					Components: []*storage.EmbeddedImageScanComponent{
						{Name: "app-1", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 3}},
						{Name: "app-2", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 4}},
					},
				},
			},
		},
		"no components - nothing to filter": {
			skip: storage.SkipImageLayers_SKIP_BASE,
			img: &storage.Image{
				Id: "test-img",
				BaseImageInfo: []*storage.BaseImageInfo{
					{MaxLayerIndex: 2},
				},
				Scan: &storage.ImageScan{
					Components: []*storage.EmbeddedImageScanComponent{},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := filterSingleImageByLayer(tc.img, tc.skip)
			assert.True(t, tc.img == result, "expected same pointer when no filtering needed")
		})
	}
}

func TestImageLayerFilter_NilImageInSlice(t *testing.T) {
	filter := newImageLayerFilter(storage.SkipImageLayers_SKIP_BASE)
	require.NotNil(t, filter)

	dep := &storage.Deployment{Id: "test-dep"}
	images := []*storage.Image{
		nil,
		{
			Id: "test-img",
			BaseImageInfo: []*storage.BaseImageInfo{
				{MaxLayerIndex: 2},
			},
			Scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{Name: "app", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 3}},
				},
			},
		},
		nil,
	}

	resultDep, resultImgs := filter.apply(dep, images)
	assert.Equal(t, dep, resultDep)
	require.Len(t, resultImgs, 3)
	assert.Nil(t, resultImgs[0])
	assert.NotNil(t, resultImgs[1])
	assert.Nil(t, resultImgs[2])
}

func TestCompileEvaluationFilter_ImageLayerFilter(t *testing.T) {
	tests := map[string]struct {
		skipImageLayers storage.SkipImageLayers
		expectFilter    bool
	}{
		"SKIP_NONE produces no filter": {
			skipImageLayers: storage.SkipImageLayers_SKIP_NONE,
			expectFilter:    false,
		},
		"SKIP_BASE produces filter": {
			skipImageLayers: storage.SkipImageLayers_SKIP_BASE,
			expectFilter:    true,
		},
		"SKIP_APP produces filter": {
			skipImageLayers: storage.SkipImageLayers_SKIP_APP,
			expectFilter:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv(features.ImageLayerFilter.EnvVar(), "true")

			filters := CompileEvaluationFilter(&storage.EvaluationFilter{
				SkipImageLayers: tc.skipImageLayers,
			})

			if tc.expectFilter {
				require.Len(t, filters, 1)
				assert.True(t, filters[0].IsNonDefault())
			} else {
				assert.Nil(t, filters)
			}
		})
	}
}

func TestCompileEvaluationFilter_ImageLayerFilter_FeatureDisabled(t *testing.T) {
	t.Setenv(features.ImageLayerFilter.EnvVar(), "false")

	filters := CompileEvaluationFilter(&storage.EvaluationFilter{
		SkipImageLayers: storage.SkipImageLayers_SKIP_BASE,
	})

	assert.Nil(t, filters, "should return nil when feature is disabled")
}

func TestCompileEvaluationFilter_IndependentFeatureGates(t *testing.T) {
	t.Run("only EvaluationFilter enabled", func(t *testing.T) {
		t.Setenv(features.EvaluationFilter.EnvVar(), "true")
		t.Setenv(features.ImageLayerFilter.EnvVar(), "false")

		filters := CompileEvaluationFilter(&storage.EvaluationFilter{
			SkipContainerTypes: []storage.ContainerType{storage.ContainerType_INIT},
			SkipImageLayers:    storage.SkipImageLayers_SKIP_BASE,
		})

		require.Len(t, filters, 1, "should only have container type filter")
	})

	t.Run("only ImageLayerFilter enabled", func(t *testing.T) {
		t.Setenv(features.EvaluationFilter.EnvVar(), "false")
		t.Setenv(features.ImageLayerFilter.EnvVar(), "true")

		filters := CompileEvaluationFilter(&storage.EvaluationFilter{
			SkipContainerTypes: []storage.ContainerType{storage.ContainerType_INIT},
			SkipImageLayers:    storage.SkipImageLayers_SKIP_BASE,
		})

		require.Len(t, filters, 1, "should only have image layer filter")
	})

	t.Run("both features enabled", func(t *testing.T) {
		t.Setenv(features.EvaluationFilter.EnvVar(), "true")
		t.Setenv(features.ImageLayerFilter.EnvVar(), "true")

		filters := CompileEvaluationFilter(&storage.EvaluationFilter{
			SkipContainerTypes: []storage.ContainerType{storage.ContainerType_INIT},
			SkipImageLayers:    storage.SkipImageLayers_SKIP_BASE,
		})

		require.Len(t, filters, 2, "should have both filters")
	})

	t.Run("both features disabled", func(t *testing.T) {
		t.Setenv(features.EvaluationFilter.EnvVar(), "false")
		t.Setenv(features.ImageLayerFilter.EnvVar(), "false")

		filters := CompileEvaluationFilter(&storage.EvaluationFilter{
			SkipContainerTypes: []storage.ContainerType{storage.ContainerType_INIT},
			SkipImageLayers:    storage.SkipImageLayers_SKIP_BASE,
		})

		assert.Nil(t, filters, "should return nil when both features disabled")
	})
}

func TestFilterSingleImageByLayer_SkipBase_PreservesComponentLayerIndexAndCompactsLayers(t *testing.T) {
	// Test the layer-index contract: component LayerIndex values are preserved (not rebased),
	// while Metadata.V1.Layers is compacted. This creates intentional non-alignment.
	img := &storage.Image{
		Id: "test-img",
		BaseImageInfo: []*storage.BaseImageInfo{
			{MaxLayerIndex: 4}, // Base layers: 0-4; App layers: 5-7
		},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{Name: "base-component", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 3}},
				{Name: "app-component", HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{LayerIndex: 6}},
			},
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Layers: []*storage.ImageLayer{
					{Instruction: "FROM", Value: "base-layer-0"},
					{Instruction: "RUN", Value: "base-layer-1"},
					{Instruction: "RUN", Value: "base-layer-2"},
					{Instruction: "RUN", Value: "base-layer-3"},
					{Instruction: "RUN", Value: "base-layer-4"},
					{Instruction: "COPY", Value: "app-layer-5"},
					{Instruction: "RUN", Value: "app-layer-6"},
					{Instruction: "CMD", Value: "app-layer-7"},
				},
			},
		},
	}

	result := filterSingleImageByLayer(img, storage.SkipImageLayers_SKIP_BASE)
	require.NotNil(t, result)

	// (a) Verify only app component (idx > 4) survived, with LayerIndex UNCHANGED
	resultComponents := result.GetScan().GetComponents()
	require.Len(t, resultComponents, 1, "should keep only app components")
	assert.Equal(t, "app-component", resultComponents[0].GetName())

	li, hasIdx := resultComponents[0].GetHasLayerIndex().(*storage.EmbeddedImageScanComponent_LayerIndex)
	require.True(t, hasIdx, "component should have layer index")
	assert.Equal(t, int32(6), li.LayerIndex, "component LayerIndex should be unchanged (original value)")

	// (b) Verify Metadata.V1.Layers was compacted to only app layers (original positions 5,6,7)
	resultLayers := result.GetMetadata().GetV1().GetLayers()
	require.Len(t, resultLayers, 3, "should have 3 app layers (compacted)")

	// Assert the kept layers are the original app layers, now repositioned to 0,1,2
	assert.Equal(t, "COPY", resultLayers[0].GetInstruction())
	assert.Equal(t, "app-layer-5", resultLayers[0].GetValue(), "first kept layer should be original layer 5")

	assert.Equal(t, "RUN", resultLayers[1].GetInstruction())
	assert.Equal(t, "app-layer-6", resultLayers[1].GetValue(), "second kept layer should be original layer 6")

	assert.Equal(t, "CMD", resultLayers[2].GetInstruction())
	assert.Equal(t, "app-layer-7", resultLayers[2].GetValue(), "third kept layer should be original layer 7")

	// (c) Assert the non-alignment: component LayerIndex (6) is >= len(Layers) (3)
	// This proves that indexing Layers by component.LayerIndex would be out of range
	assert.GreaterOrEqual(t, li.LayerIndex, int32(len(resultLayers)),
		"component LayerIndex should be >= Layers length (non-aligned)")

	// Verify original not mutated
	assert.Len(t, img.GetScan().GetComponents(), 2)
	assert.Len(t, img.GetMetadata().GetV1().GetLayers(), 8)
}
