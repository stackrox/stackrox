package indexer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	gosync "sync"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/quay/claircore"
	"github.com/quay/claircore/indexer"
	"github.com/quay/zlog"
	"github.com/stackrox/stackrox/scanner/v4/internal/sync"
	"golang.org/x/sync/errgroup"
)

var (
	_ indexer.FetchArena = (*localFetchArena)(nil)
	_ indexer.Realizer   = (*localRealizer)(nil)
)

// localFetchArena implements indexer.FetchArena.
//
// It is designed to minimize layer downloads and attempt
// to only download a layer's contents once.
//
// A localFetchArena must not be copied after first use,
// and it should be initialized via newLocalFetchArena.
type localFetchArena struct {
	// root is the directory to which layers are downloaded.
	root string
	// ko is used to ensure each layer is only downloaded once.
	ko sync.KeyedOnce[string]

	// mu protects rc and layers.
	mu gosync.Mutex
	// rc is a map of digest to refcount.
	rc map[string]int
	// layers is a map of digest to layer.
	//
	// The purpose of this map is to give each image's Realizer access to
	// this v1.Layer which can download the layer.
	// The Realizer needs to be able to download the layer,
	// which it will do via (*v1.Layer).Uncompressed().
	layers map[string]v1.Layer
}

// newLocalFetchArena initializes a new localFetchArena.
func newLocalFetchArena(root string) *localFetchArena {
	return &localFetchArena{
		root:   root,
		rc:     make(map[string]int),
		layers: make(map[string]v1.Layer),
	}
}

// Realizer returns an indexer.Realizer.
func (f *localFetchArena) Realizer(_ context.Context) indexer.Realizer {
	return &localRealizer{
		f: f,
	}
}

// Get downloads the image's manifest and returns the related claircore.Manifest.
//
// Get also downloads each previously unseen layer of the image into the arena's root directory.
func (f *localFetchArena) Get(ctx context.Context, image string, opts ...Option) (*claircore.Manifest, error) {
	// Parse the image name before doing anything else,
	// as there is no reason to do anything if the image is not properly referenced.
	ref, err := name.ParseReference(image)
	if err != nil {
		return nil, err
	}

	o := makeOptions(opts...)
	// Fetch the image's manifest from the registry.
	desc, err := remote.Get(ref, remote.WithContext(ctx), remote.WithAuth(o.auth), remote.WithPlatform(o.platform))
	if err != nil {
		return nil, err
	}

	img, err := desc.Image()
	if err != nil {
		return nil, err
	}
	d, err := img.Digest()
	if err != nil {
		return nil, err
	}
	// Convert the image manifest's digest to a claircore.Digest.
	ccd, err := claircore.ParseDigest(d.String())
	if err != nil {
		return nil, fmt.Errorf("parsing manifest digest %s: %w", d.String(), err)
	}

	manifest := &claircore.Manifest{
		Hash: ccd,
	}

	layers, err := img.Layers()
	if err != nil {
		return nil, err
	}
	manifest.Layers = make([]*claircore.Layer, len(layers))
	for i := range layers {
		d, err := layers[i].Digest()
		if err != nil {
			return nil, err
		}
		// Convert the layer's digest to a claircore.Digest.
		ccd, err := claircore.ParseDigest(d.String())
		if err != nil {
			return nil, fmt.Errorf("parsing layer digest %s: %w", d.String(), err)
		}
		manifest.Layers[i] = &claircore.Layer{
			Hash: ccd,
		}
	}

	// The number of layers tends to be a small, finite number,
	// so lock once for the image instead of once per layer
	// to minimize context switches.
	f.mu.Lock()
	for i := range layers {
		key := manifest.Layers[i].Hash.String()
		if _, exists := f.layers[manifest.Layers[i].Hash.String()]; !exists {
			f.layers[key] = layers[i]
		}
	}
	f.mu.Unlock()

	return manifest, nil
}

// realizeLayer returns a function which downloads the layer once.
//
// The function attempts to increment the digest's ref count for each call.
func (f *localFetchArena) realizeLayer(ctx context.Context, ccLayer *claircore.Layer) func() error {
	d := ccLayer.Hash.String()
	return func() error {
		path := filepath.Join(f.root, d)
		var tmp string

		select {
		case <-ctx.Done():
			return ctx.Err()
		case res := <-f.ko.DoChan(d, func() (any, error) {
			// Only the first call to DoChan will make it here.
			return f.downloadOnce(ctx, d)
		}):
			if err := res.Err; err != nil {
				return fmt.Errorf("could not download layer %s: %w", d, err)
			}
			tmp = res.V.(string)
		}

		f.mu.Lock()
		defer f.mu.Unlock()

		ct, ok := f.rc[d]
		// Is this the first time we reference the layer's file?
		if !ok {
			// Did the file get removed while we were waiting on the lock?
			if _, err := os.Stat(tmp); errors.Is(err, os.ErrNotExist) {
				return err
			}
			// Move the file to its final path.
			if err := os.Rename(tmp, path); err != nil {
				return fmt.Errorf("moving layer from temporary to final path: %w", err)
			}
		}

		ct++
		f.rc[d] = ct

		// Set the URI here for testing purposes.
		ccLayer.URI = path
		if err := ccLayer.SetLocal(path); err != nil {
			return fmt.Errorf("setting local path for %s: %w", ccLayer.Hash.String(), err)
		}

		return nil
	}
}

// downloadOnce downloads the contents of the layer into
// the arena's root directory at a temporary path.
func (f *localFetchArena) downloadOnce(ctx context.Context, digest string) (string, error) {
	f.mu.Lock()
	layer, exists := f.layers[digest]
	f.mu.Unlock()
	if !exists {
		return "", fmt.Errorf("layer %s unknown", digest)
	}

	// Write the uncompressed layer, as ClairCore's indexer assumes the layer is uncompressed.
	uncompressed, err := layer.Uncompressed()
	if err != nil {
		return "", fmt.Errorf("fetching layer %s: %w", digest, err)
	}
	defer func() {
		// TODO: consider logging failures as a warning
		// and/or tracking metrics.
		_ = uncompressed.Close()
	}()

	rm := true
	file, err := os.CreateTemp(f.root, "fetch.*")
	if err != nil {
		return "", fmt.Errorf("creating temp file for layer %s: %w", digest, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			zlog.Warn(ctx).Err(err).Msg("unable to close layer file")
		}
		if rm {
			if err := os.Remove(file.Name()); err != nil {
				zlog.Warn(ctx).Err(err).Msg("unable to remove unsuccessful layer fetch")
			}
		}
	}()

	_, err = io.Copy(file, uncompressed)
	if err != nil {
		return "", fmt.Errorf("writing contents of layer %s into temp path: %w", digest, err)
	}

	rm = false
	return file.Name(), nil
}

// forget decrements the layer's refcount and "forgets" the layer
// once the refcount reaches zero.
func (f *localFetchArena) forget(d string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	ct, ok := f.rc[d]
	if !ok {
		return nil
	}

	ct--
	if ct == 0 {
		delete(f.rc, d)
		delete(f.layers, d)
		defer f.ko.Forget(d)
		return os.Remove(filepath.Join(f.root, d))
	}

	f.rc[d] = ct

	return nil
}

// Close removes all files left in the arena.
//
// It's not an error to have active fetchers, but may cause errors to have files
// unlinked underneath their users.
func (f *localFetchArena) Close(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx,
		"component", "indexer/fetchArena.Close",
		"arena", f.root)

	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.rc) != 0 {
		zlog.Warn(ctx).
			Int("count", len(f.rc)).
			Msg("seem to have active fetchers")
		zlog.Info(ctx).
			Msg("clearing arena")
	}

	var errs []error
	for d := range f.rc {
		delete(f.rc, d)
		delete(f.layers, d)
		f.ko.Forget(d)
		if err := os.Remove(filepath.Join(f.root, d)); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}

	return nil
}

type localRealizer struct {
	f *localFetchArena
	// clean lists the layer hashes to clean up once no longer needed.
	clean []string
}

// Realize populates the local filepath for each layer.
//
// It is assumed the layer's URI is the local filesystem path to the layer.
func (f *localRealizer) Realize(ctx context.Context, ls []*claircore.Layer) error {
	f.clean = make([]string, len(ls))
	g, ctx := errgroup.WithContext(ctx)
	for i := range ls {
		f.clean[i] = ls[i].Hash.String()
		g.Go(f.f.realizeLayer(ctx, ls[i]))
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("realizing layer(s): %w", err)
	}

	return nil
}

// Close marks all the layers' backing files as unused.
//
// This method may actually delete the backing files.
func (f *localRealizer) Close() error {
	var errs []error
	for _, d := range f.clean {
		if err := f.f.forget(d); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}

	return nil
}
