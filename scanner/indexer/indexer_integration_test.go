//go:build scanner_indexer_integration

// This test is based on https://github.com/quay/claircore/blob/v1.5.30/libindex/libindex_integration_test.go.

package indexer

import (
	"context"
	"crypto/sha256"
	"io"
	"testing"
	"time"

	"github.com/quay/claircore"
	"github.com/quay/claircore/test"
	"github.com/quay/claircore/test/integration"
	pgtest "github.com/quay/claircore/test/postgres"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustCreateDigest(t *testing.T, msg string) claircore.Digest {
	h := sha256.New()
	_, err := io.WriteString(h, msg)
	require.NoError(t, err)
	d, err := claircore.NewDigest("sha256", h.Sum(nil))
	require.NoError(t, err)
	return d
}

// descriptionsToLayers takes a slice of [claircore.LayerDescription]s and
// creates equivalent [claircore.Layer] pointers.
//
// This is a helper for shims from a new API that takes a
// [claircore.LayerDescription] slice to a previous API that takes a
// [claircore.Layer] pointer slice.
//
// This is copied from https://github.com/quay/claircore/blob/main/internal/wart/layerdescription.go#L45.
func descriptionsToLayers(ds []claircore.LayerDescription) []*claircore.Layer {
	// Set up return slice.
	ls := make([]claircore.Layer, len(ds))
	ret := make([]*claircore.Layer, len(ds))
	for i := range ls {
		ret[i] = &ls[i]
	}
	// Populate the Layers.
	for i := range ds {
		d, l := &ds[i], ret[i]
		l.Hash = claircore.MustParseDigest(d.Digest)
		l.URI = d.URI
		l.Headers = make(map[string][]string, len(d.Headers))
		for k, v := range d.Headers {
			c := make([]string, len(v))
			copy(c, v)
			l.Headers[k] = c
		}
	}
	return ret
}

func TestIndexer_Integration(t *testing.T) {
	const (
		scanners = 5
		layers   = 2
		packages = 3
	)

	integration.NeedDB(t)
	ctx := zlog.Test(context.Background(), t)
	// No need to close the pool, as this function already does it.
	// It also closes the DB.
	pool := pgtest.TestIndexerDB(ctx, t)

	indexerCfg := config.IndexerConfig{
		StackRoxServices: false,
		Database: config.Database{
			ConnString: pool.Config().ConnString(),
		},
		Enable:          true,
		GetLayerTimeout: config.Duration(1 * time.Second),
	}

	indexer, err := NewIndexer(ctx, indexerCfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, indexer.Close(ctx))
	})

	assert.NoError(t, indexer.Ready(ctx))

	c, descs := test.ServeLayers(t, layers)
	m := &claircore.Manifest{
		Hash:   mustCreateDigest(t, "image_integration_test"),
		Layers: descriptionsToLayers(descs),
	}

	indexer.IndexContainerImage(ctx)
}
