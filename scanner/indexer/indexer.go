package indexer

import (
	"context"
	"crypto/sha512"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/libindex"
	"github.com/quay/claircore/pkg/ctxlock"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/config"
	"github.com/stackrox/rox/scanner/internal/version"
)

// ReportGetter can get index reports from an Indexer.
type ReportGetter interface {
	GetIndexReport(context.Context, string) (*claircore.IndexReport, bool, error)
}

// Indexer represents an image indexer.
//
//go:generate mockgen-wrapper
type Indexer interface {
	ReportGetter
	IndexContainerImage(context.Context, string, string, ...Option) (*claircore.IndexReport, error)
	Close(context.Context) error
}

// localIndexer is the Indexer implementation that runs libindex locally.
type localIndexer struct {
	libIndex        *libindex.Libindex
	getLayerTimeout time.Duration
}

// NewIndexer creates a new indexer.
func NewIndexer(ctx context.Context, cfg config.IndexerConfig) (Indexer, error) {
	pool, err := postgres.Connect(ctx, cfg.Database.ConnString, "libindex")
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

	return &localIndexer{
		libIndex:        indexer,
		getLayerTimeout: time.Duration(cfg.GetLayerTimeout),
	}, nil
}

// Close closes the indexer.
func (i *localIndexer) Close(ctx context.Context) error {
	return i.libIndex.Close(ctx)
}

// IndexContainerImage creates a ClairCore index report for a given container
// image. The manifest is populated with layers from the image specified by a
// URL. This method performs a partial content request on each layer to generate
// the layer's URI and headers.
func (i *localIndexer) IndexContainerImage(
	ctx context.Context,
	hashID string,
	imageURL string,
	opts ...Option,
) (*claircore.IndexReport, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/indexer")
	manifestDigest, err := createManifestDigest(hashID)
	if err != nil {
		return nil, err
	}
	o := makeOptions(opts...)
	imgRef, err := parseContainerImageURL(imageURL)
	if err != nil {
		return nil, fmt.Errorf("parsing image URL %q: %w", imageURL, err)
	}
	imgLayers, err := getContainerImageLayers(ctx, imgRef, o)
	if err != nil {
		return nil, fmt.Errorf("listing image layers (reference %q): %w", imgRef.String(), err)
	}
	httpClient, err := getLayerHTTPClient(ctx, imgRef, o.auth, i.getLayerTimeout)
	if err != nil {
		return nil, err
	}
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
		// TODO Check for non-retryable errors (permission denied, etc.) to report properly.
		layerReq, err := getLayerRequest(httpClient, imgRef, layerDigest)
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
	return i.libIndex.Index(ctx, manifest)
}

func getLayerHTTPClient(ctx context.Context, imgRef name.Reference, auth authn.Authenticator, timeout time.Duration) (*http.Client, error) {
	repo := imgRef.Context()
	reg := repo.Registry
	tr := remote.DefaultTransport
	tr = transport.NewUserAgent(tr, `StackRox Scanner/`+version.Version)
	tr = transport.NewRetry(tr)
	var err error
	tr, err = transport.NewWithContext(ctx, reg, auth, tr, []string{repo.Scope(transport.PullScope)})
	if err != nil {
		return nil, err
	}
	httpClient := http.Client{
		Timeout:   timeout,
		Transport: tr,
	}
	return &httpClient, nil
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

// GetIndexReport retrieves an IndexReport for the given hash ID, if it exists.
func (i *localIndexer) GetIndexReport(ctx context.Context, hashID string) (*claircore.IndexReport, bool, error) {
	manifestDigest, err := createManifestDigest(hashID)
	if err != nil {
		return nil, false, err
	}
	return i.libIndex.IndexReport(ctx, manifestDigest)
}

// createManifestDigest creates a unique claircore.Digest from a Scanner's manifest hash ID.
func createManifestDigest(hashID string) (claircore.Digest, error) {
	hashIDSum := sha512.Sum512([]byte(hashID))
	d, err := claircore.NewDigest(claircore.SHA512, hashIDSum[:])
	if err != nil {
		return claircore.Digest{}, fmt.Errorf("creating manifest digest: %w", err)
	}
	return d, nil
}

// getContainerImageLayers fetches the image's manifest from the registry to get
// a list of layers.
func getContainerImageLayers(ctx context.Context, ref name.Reference, o options) ([]v1.Layer, error) {
	// TODO Check for non-retryable errors (permission denied, etc.) to report properly.
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

// parseContainerImageURL returns an image reference from an image URL.
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

// GetDigestFromURL returns an image digest from the given image URL.
func GetDigestFromURL(imgURL string, auth authn.Authenticator) (name.Digest, error) {
	ref, err := parseContainerImageURL(imgURL)
	if err != nil {
		return name.Digest{}, err
	}
	return GetDigestFromReference(ref, auth)
}

// GetDigestFromReference returns an image digest from a reference, it either
// returns the digest specified in the image reference or reads from the
// registry's image manifest.
func GetDigestFromReference(ref name.Reference, auth authn.Authenticator) (name.Digest, error) {
	if d, ok := ref.(name.Digest); ok {
		return d, nil
	}
	// If not, convert to a digest reference by retrieving the digest.
	img, err := remote.Image(ref, remote.WithAuth(auth))
	if err != nil {
		return name.Digest{}, err
	}
	hash, err := img.Digest()
	if err != nil {
		return name.Digest{}, err
	}
	s := fmt.Sprintf("%s@%s", ref.Context().String(), hash.String())
	dRef, err := name.NewDigest(s)
	if err != nil {
		return name.Digest{}, fmt.Errorf("internal error: %w", err)
	}
	return dRef, nil
}
