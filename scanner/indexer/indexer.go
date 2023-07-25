package indexer

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/libindex"
	"github.com/quay/claircore/pkg/ctxlock"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/utils"
)

// Indexer represents an image indexer.
//go:generate mockgen-wrapper
type Indexer interface {
	IndexContainerImage(context.Context, claircore.Digest, string, ...Option) (*claircore.IndexReport, error)
	Close(context.Context) error
}

type indexerImpl struct {
	indexer *libindex.Libindex
}

// NewIndexer creates a new indexer.
func NewIndexer(ctx context.Context) (Indexer, error) {
	// TODO: Update the conn string to something configurable.
	pool, err := postgres.Connect(ctx, "postgresql:///postgres?host=/var/run/postgresql", "libindex")
	if err != nil {
		return nil, errors.Wrap(err, "connecting to postgres for indexer")
	}
	store, err := postgres.InitPostgresIndexerStore(ctx, pool, true)
	if err != nil {
		return nil, errors.Wrap(err, "initializing postgres indexer store")
	}
	locker, err := ctxlock.New(ctx, pool)
	if err != nil {
		return nil, errors.Wrap(err, "creating indexer postgres locker")
	}

	// TODO: Update the HTTP client.
	c := http.DefaultClient
	// TODO: When adding Indexer.Close(), make sure to clean-up /tmp.
	faRoot, err := os.MkdirTemp("", "scanner-fetcharena-*")
	if err != nil {
		return nil, errors.Wrap(err, "creating indexer root directory")
	}
	defer utils.IgnoreError(func() error {
		if err != nil {
			return os.RemoveAll(faRoot)
		}
		return nil
	})
	// TODO: Consider making layer scan concurrency configurable?
	opts := libindex.Options{
		Store:                store,
		Locker:               locker,
		FetchArena:           libindex.NewRemoteFetchArena(c, faRoot),
		ScanLockRetry:        libindex.DefaultScanLockRetry,
		LayerScanConcurrency: libindex.DefaultLayerScanConcurrency,
	}

	indexer, err := libindex.New(ctx, &opts, c)
	if err != nil {
		return nil, errors.Wrap(err, "creating libindex")
	}

	return &indexerImpl{
		indexer: indexer,
	}, nil
}

// Close closes the indexer.
func (i *indexerImpl) Close(ctx context.Context) error {
	return i.indexer.Close(ctx)
}

// IndexContainerImage creates a ClairCore index report for a given container
// image. The manifest is populated with layers from the image specified by a
// URL. This method performs a partial content request on each layer to generate
// the layer's URI and headers.
func (i *indexerImpl) IndexContainerImage(
	ctx context.Context,
	manifestDigest claircore.Digest,
	imageURL string,
	opts ...Option,
) (*claircore.IndexReport, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/indexer")
	o := makeOptions(opts...)
	imgRef, err := parseContainerImageURL(imageURL)
	if err != nil {
		return nil, err
	}
	imgDigest, imgLayers, err := getContainerImageLayers(ctx, o, imgRef)
	if err != nil {
		return nil, err
	}
	imgRepo := imgRef.Context()
	registryURL := url.URL{
		Scheme: imgRepo.Scheme(),
		Host:   imgRepo.RegistryStr(),
	}
	httpClient := http.Client{Timeout: time.Duration(1) * time.Minute}
	manifest := &claircore.Manifest{
		Hash: manifestDigest,
	}
	zlog.Info(ctx).
		Str("image_digest", imgDigest).
		Str("registry", imgRepo.RegistryStr()).
		Int("layers_count", len(imgLayers)).
		Msg("retrieving layers to populate manifest")
	for _, l := range imgLayers {
		d, err := l.Digest()
		if err != nil {
			return nil, err
		}
		ccd, err := claircore.ParseDigest(d.String())
		if err != nil {
			return nil, err
		}
		imgPath := strings.TrimPrefix(imgRepo.RepositoryStr(), imgRepo.RegistryStr())
		u, err := registryURL.Parse(path.Join("/", "v2", imgPath, "blobs", d.String()))
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Add("Range", "bytes=0-0")
		res, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		utils.IgnoreError(res.Body.Close)
		res.Request.Header.Del("User-Agent")
		res.Request.Header.Del("Range")
		manifest.Layers = append(manifest.Layers, &claircore.Layer{
			Hash:    ccd,
			URI:     res.Request.URL.String(),
			Headers: res.Request.Header,
		})
	}
	return i.indexer.Index(ctx, manifest)
}

// getContainerImageLayers fetches the image's manifest from the registry to get
// a list of layers.
func getContainerImageLayers(ctx context.Context, o options, ref name.Reference) (string, []v1.Layer, error) {
	// TODO Check for non-retriable errors (permission denied, etc.) to report properly.
	desc, err := remote.Get(ref, remote.WithContext(ctx), remote.WithAuth(o.auth), remote.WithPlatform(o.platform))
	if err != nil {
		return "", nil, err
	}
	img, err := desc.Image()
	if err != nil {
		return "", nil, err
	}
	ccd, err := img.Digest()
	if err != nil {
		return "", nil, err
	}
	layers, err := img.Layers()
	if err != nil {
		return "", nil, err
	}
	return ccd.String(), layers, nil
}

// parseContainerImageURL returns a image reference from an image URL.
func parseContainerImageURL(imageURL string) (name.Reference, error) {
	// Parse image reference to ensure we have a valid reference.
	parsedURL, err := url.Parse(imageURL)
	if err != nil {
		// We expect input was sanitized, so this is an internal error.
		return nil, err
	}
	parseOpts := []name.Option{name.StrictValidation}
	if parsedURL.Scheme == "http" {
		parseOpts = append(parseOpts, name.Insecure)
	}
	imageRef := strings.TrimPrefix(imageURL, parsedURL.Scheme+"://")
	ref, err := name.ParseReference(imageRef, parseOpts...)
	if err != nil {
		return nil, err
	}
	return ref, nil
}
