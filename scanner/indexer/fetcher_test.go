package indexer

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetcherSimple(t *testing.T) {
	root, err := filepath.Abs("testdata")
	require.NoError(t, err)

	a := newLocalFetchArena(root)

	image := "quay.io/stackrox-io/main:4.0.0"
	t.Run(image, func(t *testing.T) {
		run(t, a, image, 13)
	})
}

func TestFetcherParallel(t *testing.T) {
	root, err := filepath.Abs("testdata")
	require.NoError(t, err)

	a := newLocalFetchArena(root)

	// Repeat each image to test duplicate calls.
	for _, testcase := range []struct {
		name   string
		image  string
		layers int
	}{
		{
			name:   "main",
			image:  "quay.io/stackrox-io/main:4.0.0",
			layers: 13,
		},
		{
			name:   "duplicate main",
			image:  "quay.io/stackrox-io/main:4.0.0",
			layers: 13,
		},
		{
			name:   "scanner",
			image:  "quay.io/stackrox-io/scanner:4.0.0",
			layers: 2,
		},
		{
			name:   "duplicate scanner",
			image:  "quay.io/stackrox-io/scanner:4.0.0",
			layers: 2,
		},
		{
			name:   "scanner slim",
			image:  "quay.io/stackrox-io/scanner-slim:4.0.0",
			layers: 2,
		},
		{
			name:   "duplicate scanner slim",
			image:  "quay.io/stackrox-io/scanner-slim:4.0.0",
			layers: 2,
		},
		{
			name:   "scanner db",
			image:  "quay.io/stackrox-io/scanner-db:4.0.0",
			layers: 2,
		},
		{
			name:   "duplicate scanner db",
			image:  "quay.io/stackrox-io/scanner-db:4.0.0",
			layers: 2,
		},
	} {
		testcase := testcase
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			run(t, a, testcase.image, testcase.layers)
		})
	}
}

// run runs the test.
//
// run is just simply ensuring there are no errors when fetching the manifest and layers
// of the image and the layer's path is set properly.
func run(t *testing.T, a *localFetchArena, image string, expectedLayers int) {
	ctx := context.Background()

	m, err := a.Get(ctx, image)
	require.NoError(t, err)

	r := a.Realizer(ctx)

	err = r.Realize(ctx, m.Layers)
	require.NoError(t, err)

	assert.Equal(t, expectedLayers, len(m.Layers), "did not get expected number of layers")
	for _, l := range m.Layers {
		t.Logf("%+v", l)

		expected := filepath.Join(a.root, l.Hash.String())
		assert.Equal(t, expected, l.URI, "URI != local filepath")
	}
	require.NoError(t, r.Close())
}
