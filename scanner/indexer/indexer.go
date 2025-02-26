package indexer

import (
	"context"
	"crypto/sha512"
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand/v2"
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
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/quay/claircore"
	"github.com/quay/claircore/alpine"
	"github.com/quay/claircore/chainguard"
	ccpostgres "github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/dpkg"
	"github.com/quay/claircore/gobin"
	ccindexer "github.com/quay/claircore/indexer"
	"github.com/quay/claircore/java"
	"github.com/quay/claircore/libindex"
	"github.com/quay/claircore/nodejs"
	"github.com/quay/claircore/pkg/ctxlock"
	"github.com/quay/claircore/python"
	"github.com/quay/claircore/rhel"
	"github.com/quay/claircore/rhel/rhcc"
	"github.com/quay/claircore/rpm"
	"github.com/quay/claircore/ruby"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/config"
	"github.com/stackrox/rox/scanner/datastore/postgres"
	"github.com/stackrox/rox/scanner/indexer/manifest"
	"github.com/stackrox/rox/scanner/internal/httputil"
	"github.com/stackrox/rox/scanner/internal/version"
)

var (
	// minManifestDeleteStart is the minimum amount of time in the future to consider deleting manifests.
	minManifestDeleteStart = minManifestDeleteIntervalStart()
	// minManifestDeleteDuration is the minimum amount of time of the interval for considering deleting manifests.
	minManifestDeleteDuration = minManifestDeleteIntervalDuration()
)

// minManifestDeleteIntervalStart returns the minimum manifest deletion interval start.
// For release builds: 1 hour
// For dev builds: 1 minute
func minManifestDeleteIntervalStart() time.Duration {
	if buildinfo.ReleaseBuild {
		return 1 * time.Hour
	}
	return time.Minute
}

// minManifestDeleteIntervalDuration returns the minimum duration of the manifest deletion interval.
// For release builds: 1 hour
// For dev builds: 2 minutes
func minManifestDeleteIntervalDuration() time.Duration {
	if buildinfo.ReleaseBuild {
		return 1 * time.Hour
	}
	return 2 * time.Minute
}

// ecosystems specifies the package ecosystems to use for indexing.
func ecosystems(ctx context.Context) []*ccindexer.Ecosystem {
	es := []*ccindexer.Ecosystem{
		alpine.NewEcosystem(ctx),
		dpkg.NewEcosystem(ctx),
		chainguard.NewEcosystem(ctx),
		gobin.NewEcosystem(ctx),
		java.NewEcosystem(ctx),
		nodejs.NewEcosystem(ctx),
		python.NewEcosystem(ctx),
		rhcc.NewEcosystem(ctx),
		rhel.NewEcosystem(ctx),
		rpm.NewEcosystem(ctx),
		ruby.NewEcosystem(ctx),
	}
	return es
}

var (
	// remoteTransport is the http.RoundTripper to use when talking to image registries.
	remoteTransport         = proxiedRemoteTransport(false)
	insecureRemoteTransport = proxiedRemoteTransport(true)

	// regsNoRange is a set of registries which do not accept the Range HTTP header.
	// This is used for logging purposes, only.
	regsNoRange *lru.Cache[string, struct{}]
)

func init() {
	var err error
	// Size was chosen rather arbitrarily.
	regsNoRange, err = lru.New[string, struct{}](100)
	utils.CrashOnError(err) // This should never happen.
}

func proxiedRemoteTransport(insecure bool) http.RoundTripper {
	tr := func() *http.Transport {
		tr, ok := remote.DefaultTransport.(*http.Transport)
		if !ok {
			// The proxy function was already modified to proxy.TransportFunc.
			// See scanner/cmd/scanner/main.go.
			return http.DefaultTransport.(*http.Transport).Clone()
		}
		tr = tr.Clone()
		tr.Proxy = proxy.TransportFunc
		return tr
	}()
	if insecure {
		tr.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	return tr
}

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
	Ready(context.Context) error
}

// localIndexer is the Indexer implementation that runs libindex locally.
type localIndexer struct {
	libIndex        *libindex.Libindex
	pool            *pgxpool.Pool
	vscnrs          ccindexer.VersionedScanners
	root            string
	getLayerTimeout time.Duration

	metadataStore          postgres.IndexerMetadataStore
	manifestManager        *manifest.Manager
	deleteIntervalStart    int64
	deleteIntervalDuration int64
}

// NewIndexer creates a new indexer.
func NewIndexer(ctx context.Context, cfg config.IndexerConfig) (Indexer, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/indexer.NewIndexer")

	var success bool

	pool, err := postgres.Connect(ctx, cfg.Database.ConnString, "libindex")
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres for indexer: %w", err)
	}
	defer func() {
		if !success {
			pool.Close()
		}
	}()

	store, err := ccpostgres.InitPostgresIndexerStore(ctx, pool, true)
	if err != nil {
		return nil, fmt.Errorf("initializing postgres indexer store: %w", err)
	}
	defer func() {
		if !success {
			_ = store.Close(ctx)
		}
	}()

	locker, err := ctxlock.New(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("creating indexer postgres locker: %w", err)
	}
	defer func() {
		if !success {
			_ = locker.Close(ctx)
		}
	}()

	var metadataStore postgres.IndexerMetadataStore
	if features.ScannerV4ReIndex.Enabled() {
		metadataStore, err = postgres.InitPostgresIndexerMetadataStore(ctx, pool, true, postgres.IndexerMetadataStoreOpts{
			IndexerStore: store,
		})
		if err != nil {
			return nil, fmt.Errorf("initializing postgres indexer metadata store: %w", err)
		}
	}

	root, err := os.MkdirTemp("", "scanner-fetcharena-*")
	if err != nil {
		return nil, fmt.Errorf("creating indexer root directory: %w", err)
	}
	defer func() {
		if !success {
			_ = os.RemoveAll(root)
		}
	}()

	// Note: http.DefaultTransport has already been modified to handle configured proxies.
	// See scanner/cmd/scanner/main.go.
	t, err := httputil.TransportMux(http.DefaultTransport, httputil.WithDenyStackRoxServices(!cfg.StackRoxServices))
	if err != nil {
		return nil, fmt.Errorf("creating HTTP transport: %w", err)
	}
	client := &http.Client{
		Transport: t,
	}

	indexer, err := newLibindex(ctx, cfg, client, root, store, locker)
	if err != nil {
		return nil, err
	}

	// Use indexer.Ecosystems instead of the ecosystems we pass to libindex.New
	// in case libindex.New adds any ecosystems (which it does).
	vscnrs, err := versionedScanners(ctx, indexer.Ecosystems)
	if err != nil {
		return nil, err
	}

	var manifestManager *manifest.Manager
	if features.ScannerV4ReIndex.Enabled() {
		manifestManager = manifest.NewManager(ctx, metadataStore, locker)
		// Set any manifests indexed prior to the existence of the manifest_metadata table
		// to expire immediately.
		// TODO(ROX-26957): Consider moving this elsewhere so we do not block initialization.
		// TODO(ROX-26995): Consider updating the immediate purge condition.
		// It may be possible we want to purge all manifests upon startup for other reasons.
		err = manifestManager.MigrateManifests(ctx, time.Now())
		if err != nil {
			// TODO(ROX-26958): Consider just logging this instead once we start deleting entries
			// missing from the metadata table, too.
			return nil, fmt.Errorf("migrating manifests to metadata store: %w", err)
		}
		// Start the manifest GC.
		go func() {
			if err := manifestManager.StartGC(); err != nil {
				zlog.Error(ctx).Err(err).Msg("manifest GC failed")
			}
		}()
	}

	deleteIntervalStart := env.ScannerV4ManifestDeleteStart.DurationSetting()
	if deleteIntervalStart < minManifestDeleteStart {
		zlog.Warn(ctx).Msgf("configured manifest delete interval (%v) start is too small: setting to %v", deleteIntervalStart, minManifestDeleteStart)
		deleteIntervalStart = minManifestDeleteStart
	}
	deleteIntervalDuration := env.ScannerV4ManifestDeleteDuration.DurationSetting()
	if deleteIntervalDuration < minManifestDeleteDuration {
		zlog.Warn(ctx).Msgf("configured manifest delete interval (%v) duration is too small: setting to %v", deleteIntervalDuration, minManifestDeleteDuration)
		deleteIntervalDuration = minManifestDeleteDuration
	}

	success = true
	return &localIndexer{
		libIndex:        indexer,
		vscnrs:          vscnrs,
		pool:            pool,
		root:            root,
		getLayerTimeout: time.Duration(cfg.GetLayerTimeout),

		metadataStore:          metadataStore,
		manifestManager:        manifestManager,
		deleteIntervalStart:    int64(deleteIntervalStart.Seconds()),
		deleteIntervalDuration: int64(deleteIntervalDuration.Seconds()),
	}, nil
}

// versionedScanners returns the versioned scanners derived from the given ecosystems.
//
// This is based on https://github.com/quay/claircore/blob/v1.5.33/libindex/libindex.go#L127.
func versionedScanners(ctx context.Context, ecosystems []*ccindexer.Ecosystem) (ccindexer.VersionedScanners, error) {
	// Get all current versioned scanners.
	pscnrs, dscnrs, rscnrs, fscnrs, err := ccindexer.EcosystemsToScanners(ctx, ecosystems)
	if err != nil {
		return nil, fmt.Errorf("converting ecosystems to scanners: %w", err)
	}
	return ccindexer.MergeVS(pscnrs, dscnrs, rscnrs, fscnrs), nil
}

func castToConfig[T any](f func(cfg T)) func(o any) error {
	return func(o any) error {
		cfg, ok := o.(T)
		if !ok {
			return errors.New("internal error: casting failed")
		}
		f(cfg)
		return nil
	}
}

func newLibindex(ctx context.Context, indexerCfg config.IndexerConfig, client *http.Client, root string, store ccindexer.Store, locker *ctxlock.Locker) (*libindex.Libindex, error) {
	// TODO: Consider making layer scan concurrency configurable?
	opts := libindex.Options{
		Store:                store,
		Locker:               locker,
		FetchArena:           libindex.NewRemoteFetchArena(client, root),
		ScanLockRetry:        libindex.DefaultScanLockRetry,
		LayerScanConcurrency: libindex.DefaultLayerScanConcurrency,
		Ecosystems:           ecosystems(ctx),
		ScannerConfig: struct {
			Package, Dist, Repo, File map[string]func(any) error
		}{
			Repo: map[string]func(any) error{
				"rhel-repository-scanner": castToConfig(func(cfg *rhel.RepositoryScannerConfig) {
					cfg.DisableAPI = true
					cfg.Repo2CPEMappingURL = indexerCfg.RepositoryToCPEURL
					cfg.Repo2CPEMappingFile = indexerCfg.RepositoryToCPEFile
				}),
			},
			Package: map[string]func(any) error{
				"rhel_containerscanner": castToConfig(func(cfg *rhcc.ScannerConfig) {
					cfg.Name2ReposMappingURL = indexerCfg.NameToReposURL
					cfg.Name2ReposMappingFile = indexerCfg.NameToReposFile
				}),
				"java": castToConfig(func(cfg *java.ScannerConfig) {
					cfg.DisableAPI = true
				}),
			},
		},
	}

	indexer, err := libindex.New(ctx, &opts, client)
	if err != nil {
		return nil, fmt.Errorf("creating libindex: %w", err)
	}

	return indexer, nil
}

// Close closes the indexer.
func (i *localIndexer) Close(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/indexer.Close")
	err := errors.Join(i.libIndex.Close(ctx), os.RemoveAll(i.root))
	if features.ScannerV4ReIndex.Enabled() && i.manifestManager != nil {
		err = errors.Join(err, i.manifestManager.StopGC())
	}
	i.pool.Close()
	return err
}

func (i *localIndexer) Ready(ctx context.Context) error {
	if err := i.pool.Ping(ctx); err != nil {
		return fmt.Errorf("indexer DB ping failed: %w", err)
	}
	return nil
}

// IndexContainerImage creates a ClairCore index report for a given container
// image. The manifest is populated with layers from the image specified by a
// URL. This method performs a partial content request on each layer to generate
// the layer's URI and headers.
func (i *localIndexer) IndexContainerImage(ctx context.Context, hashID string, imageURL string, opts ...Option) (*claircore.IndexReport, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/indexer.IndexContainerImage")

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
	httpClient, err := getLayerHTTPClient(ctx, imgRef, o.auth, i.getLayerTimeout, o.insecureSkipTLSVerify)
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
		layerReq, err := getLayerRequest(ctx, httpClient, imgRef, layerDigest)
		if err != nil {
			return nil, fmt.Errorf("getting layer request URL and headers (digest: %q): %w", layerDigest.String(), err)
		}
		layerReq.Header.Del("User-Agent")
		layerReq.Header.Del("Range")
		manifest.Layers = append(manifest.Layers, &claircore.Layer{
			Hash:    ccDigest,
			URI:     layerReq.URL.String(),
			Headers: layerReq.Header,
		})
	}

	ir, err := i.libIndex.Index(ctx, manifest)
	if err != nil {
		return nil, err
	}

	if features.ScannerV4ReIndex.Enabled() && i.metadataStore != nil {
		err = i.metadataStore.StoreManifest(ctx, manifestDigest.String(), i.randomExpiry(time.Now()))
		if err != nil {
			return nil, err
		}
	}

	return ir, nil
}

// randomExpiry generates a random time.Time within the manifest deletion interval
// rounding down to the nearest second.
func (i *localIndexer) randomExpiry(now time.Time) time.Time {
	expirySec := now.Unix() + rand.Int64N(i.deleteIntervalDuration) + i.deleteIntervalStart
	return time.Unix(expirySec, 0)
}

func getLayerHTTPClient(ctx context.Context, imgRef name.Reference, auth authn.Authenticator, timeout time.Duration, insecure bool) (*http.Client, error) {
	repo := imgRef.Context()
	reg := repo.Registry
	tr := remoteTransport
	if insecure {
		tr = insecureRemoteTransport
	}
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

// getLayerDigests returns the claircore and containerregistry digests for the layer.
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
func getLayerRequest(ctx context.Context, httpClient *http.Client, imgRef name.Reference, layerDigest v1.Hash) (*http.Request, error) {
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
	if res.StatusCode != http.StatusPartialContent {
		if exists, _ := regsNoRange.ContainsOrAdd(registryURL.Host, struct{}{}); !exists {
			zlog.Warn(ctx).
				Str("registry", registryURL.Host).
				Msg("Range HTTP header may not be supported, so indexing may required about twice as many image pulls")
		}

		zlog.Debug(ctx).
			Int("status_code", res.StatusCode).
			Int("len", int(res.ContentLength)).
			Str("url", u.String()).
			Msg("server might not support requests with Range HTTP header")
	}
	return res.Request, nil
}

// GetIndexReport retrieves an IndexReport for the given hash ID, if it exists and is up-to-date.
func (i *localIndexer) GetIndexReport(ctx context.Context, hashID string) (*claircore.IndexReport, bool, error) {
	manifestDigest, err := createManifestDigest(hashID)
	if err != nil {
		return nil, false, err
	}
	if features.ScannerV4ReIndex.Enabled() && i.metadataStore != nil {
		exists, err := i.metadataStore.ManifestExists(ctx, manifestDigest.String())
		if err != nil {
			return nil, false, fmt.Errorf("checking if manifest metadata exists: %w", err)
		}
		if !exists {
			// Even if the indexing was successful, if the manifest metadata was not stored, then we consider
			// this manifest as non-existent.
			//
			// Note: There are two known situations in which this may happen:
			//
			// 1. When storing the manifest metadata fails but the indexing succeeds.
			//    We will not run into a situation where manifest metadata was added successfully but deleted prior
			//    to successfully fetching the index report, as the manifest metadata and index
			//    report are deleted together in a transaction.
			// 2. Upon upgrade from an older version of the Indexer in which the manifest metadata table does not
			//    exist to a version in which it does. It is possible the upgraded version migrates all
			//    known manifests over to the metadata table, but there is still an older Indexer running which successfully
			//    indexes a manifest after the migration. The manifest metadata table will now be missing an entry related to
			//    this new index report. This is ok, as it will be caught here and the manifest will be re-indexed.
			return nil, false, nil
		}
	}
	scanned, err := i.libIndex.Store.ManifestScanned(ctx, manifestDigest, i.vscnrs)
	if err != nil {
		return nil, false, fmt.Errorf("fetching manifest: %w", err)
	}
	if !scanned {
		// The IndexReport is obsolete, as there has been an update to
		// the versioned scanners since this manifest was indexed.
		return nil, false, nil
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
	tr := remoteTransport
	if o.insecureSkipTLSVerify {
		tr = insecureRemoteTransport
	}

	desc, err := remote.Get(ref, remote.WithContext(ctx), remote.WithAuth(o.auth), remote.WithPlatform(o.platform), remote.WithTransport(tr))

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
		return nil, errors.New("invalid URL: empty")
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
		return nil, fmt.Errorf("invalid URL scheme %q", parsedURL.Scheme)
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
