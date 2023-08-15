package indexer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/libindex"
	"github.com/quay/claircore/pkg/ctxlock"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/utils"
)

// Indexer represents an image indexer.
//
//go:generate mockgen-wrapper
type Indexer interface {
	IndexContainerImage(context.Context, claircore.Digest, string, ...Option) (*claircore.IndexReport, error)
	GetIndexReport(ctx context.Context, manifestDigest claircore.Digest) (*claircore.IndexReport, bool, error)
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
		return nil, fmt.Errorf("connecting to postgres for indexer: %w", err)
	}
	store, err := postgres.InitPostgresIndexerStore(ctx, pool, true)
	if err != nil {
		return nil, fmt.Errorf("initializing postgres indexer store: %w", err)
	}
	locker, err := ctxlock.New(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("creating indexer postgres locker: %w", err)
	}

	// TODO: Update the HTTP client.
	c := http.DefaultClient
	// TODO: When adding Indexer.Close(), make sure to clean-up /tmp.
	faRoot, err := os.MkdirTemp("", "scanner-fetcharena-*")
	if err != nil {
		return nil, fmt.Errorf("creating indexer root directory: %w", err)
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
		return nil, fmt.Errorf("creating libindex: %w", err)
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
		return nil, fmt.Errorf("parsing image URL %q: %w", imageURL, err)
	}
	imgLayers, err := getContainerImageLayers(ctx, imgRef, o)
	if err != nil {
		return nil, fmt.Errorf("listing image layers (reference %q): %w", imgRef.String(), err)
	}
	httpClient := http.Client{Timeout: time.Minute}
	manifest := &claircore.Manifest{
		Hash: manifestDigest,
	}
	zlog.Info(ctx).
		Str("image_reference", imgRef.String()).
		Int("layers_count", len(imgLayers)).
		Msg("retrieving layers to populate container image manifest")
	for _, layer := range imgLayers {
		ccDigest, layerDigest, err := getLayerDigests(layer)
		if err != nil {
			return nil, fmt.Errorf("getting layer digests: %w", err)
		}
		// TODO Check for non-retriable errors (permission denied, etc.) to report properly.
		layerReq, err := getLayerRequest(&httpClient, imgRef, layerDigest)
		if err != nil {
			return nil, fmt.Errorf("getting layer request URL and headers (digest: %q): %w",
				layerDigest.String(), err)
		}
		layerReq.Header.Del("User-Agent")
		layerReq.Header.Del("Range")
		manifest.Layers = append(manifest.Layers, &claircore.Layer{
			Hash:    ccDigest,
			URI:     layerReq.URL.String(),
			Headers: layerReq.Header,
		})
	}
	return i.indexer.Index(ctx, manifest)
}

// getLayerDigests returns the clairclore and containerregistry digests for the layer.
func getLayerDigests(layer v1.Layer) (ccd claircore.Digest, ld v1.Hash, err error) {
	ld, err = layer.Digest()
	if err != nil {
		return ccd, ld, err
	}
	ccd, err = claircore.ParseDigest(ld.String())
	return ccd, ld, err
}

// getLayerRequest sends a partial request to retrieve the layer from the
// registry and return the request object containing relevant information to
// populate a container image manifest.
func getLayerRequest(httpClient *http.Client, imgRef name.Reference, layerDigest v1.Hash) (*http.Request, error) {
	imgRepo := imgRef.Context()
	registryURL := url.URL{
		Scheme: imgRepo.Scheme(),
		Host:   imgRepo.RegistryStr(),
	}
	imgPath := strings.TrimPrefix(imgRepo.RepositoryStr(), imgRepo.RegistryStr())
	imgURL := path.Join("/", "v2", imgPath, "blobs", layerDigest.String())
	u, err := registryURL.Parse(imgURL)
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
	return res.Request, nil
}

// GetIndexReport retrieves an IndexReport for a particular manifest hash, if it exists.
func (i *indexerImpl) GetIndexReport(ctx context.Context, manifestDigest claircore.Digest) (*claircore.IndexReport, bool, error) {
	return i.indexer.IndexReport(ctx, manifestDigest)
}

// getContainerImageLayers fetches the image's manifest from the registry to get
// a list of layers.
func getContainerImageLayers(ctx context.Context, ref name.Reference, o options) ([]v1.Layer, error) {
	// TODO Check for non-retriable errors (permission denied, etc.) to report properly.
	desc, err := remote.Get(ref, remote.WithContext(ctx), remote.WithAuth(o.auth), remote.WithPlatform(o.platform))
	if err != nil {
		return nil, err
	}
	img, err := desc.Image()
	if err != nil {
		return nil, err
	}
	layers, err := img.Layers()
	if err != nil {
		return nil, err
	}
	return layers, nil
}

// parseContainerImageURL returns a image reference from an image URL.
func parseContainerImageURL(imageURL string) (name.Reference, error) {
	// We expect input was sanitized, so all errors here are considered internal errors.
	if imageURL == "" {
		return nil, errors.New("invalid URL")
	}
	// Parse image reference to ensure it is valid.
	parsedURL, err := url.Parse(imageURL)
	if err != nil {
		return nil, err
	}
	// Check URL scheme and update ref parsing options.
	parseOpts := []name.Option{name.StrictValidation}
	switch parsedURL.Scheme {
	case "http":
		parseOpts = append(parseOpts, name.Insecure)
	case "https":
	default:
		return nil, errors.New("invalid URL")
	}
	// Strip the URL scheme:// and parse host/path as an image reference.
	imageRef := strings.TrimPrefix(imageURL, parsedURL.Scheme+"://")
	ref, err := name.ParseReference(imageRef, parseOpts...)
	if err != nil {
		return nil, err
	}
	return ref, nil
}
