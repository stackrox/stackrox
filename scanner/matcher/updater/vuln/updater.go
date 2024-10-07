package vuln

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/claircore/pkg/ctxlock"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/datastore/postgres"
	"github.com/stackrox/rox/scanner/updater/jsonblob"
)

const (
	ifModifiedSinceHeader = `If-Modified-Since`
	lastModifiedHeader    = `Last-Modified`

	updateName = `scanner-v4-updater`

	updateFilePattern = `updates-*.json.zst`

	defaultUpdateInterval  = 5 * time.Minute
	defaultUpdateRetention = libvuln.DefaultUpdateRetention
	defaultFullGCInterval  = 24 * time.Hour

	defaultRetryMax   = 4
	defaultRetryDelay = 10 * time.Second
)

var (
	random = rand.New(rand.NewSource(time.Now().UnixNano()))

	jitterMinutes = []time.Duration{
		5 * time.Minute,
		10 * time.Minute,
		15 * time.Minute,
		20 * time.Minute,
	}
)

// Opts represents Updater options.
//
// Store, Locker, MetadataStore, and URL are required.
// The rest are optional.
type Opts struct {
	Store         postgres.MatcherStore
	Locker        *ctxlock.Locker
	MetadataStore postgres.MatcherMetadataStore

	Client         *http.Client
	URL            string
	Root           string
	UpdateInterval time.Duration

	SkipGC          bool
	UpdateRetention int
	FullGCInterval  time.Duration

	RetryDelay time.Duration
	RetryMax   int
}

// Updater represents a vulnerability updater.
// An Updater reaches out to a given URL periodically to fetch the latest vulnerability data.
type Updater struct {
	ctx    context.Context
	cancel context.CancelFunc

	store         postgres.MatcherStore
	locker        updates.LockSource
	metadataStore postgres.MatcherMetadataStore

	client         *http.Client
	url            string
	root           string
	updateInterval time.Duration
	initialized    atomic.Bool

	skipGC          bool
	updateRetention int
	fullGCInterval  time.Duration

	// importFunc import records from a reader (to be mocked by tests).
	importFunc func(ctx context.Context, reader io.Reader) error

	// iterateFunc iterates over operations and their records from a reader (to be
	// mocked by tests).
	iterateFunc func(r io.Reader) (jsonblob.OperationIter, func() error)

	retryDelay time.Duration
	retryMax   int

	distManager *distManager
}

// New creates a new Updater based on the given options.
func New(ctx context.Context, opts Opts) (*Updater, error) {
	if err := fillOpts(&opts); err != nil {
		return nil, fmt.Errorf("invalid updater options: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	u := &Updater{
		ctx:    ctx,
		cancel: cancel,

		store:         opts.Store,
		locker:        opts.Locker,
		metadataStore: opts.MetadataStore,

		client:         opts.Client,
		url:            opts.URL,
		root:           opts.Root,
		updateInterval: opts.UpdateInterval,

		skipGC:          opts.SkipGC,
		updateRetention: opts.UpdateRetention,
		fullGCInterval:  opts.FullGCInterval,

		retryDelay: opts.RetryDelay,
		retryMax:   opts.RetryMax,

		distManager: newDistManager(opts.Store),
	}
	u.importFunc = func(ctx context.Context, reader io.Reader) error {
		return u.Import(ctx, reader)
	}
	u.iterateFunc = jsonblob.Iterate

	u.tryRemoveExiting()

	return u, nil
}

func fillOpts(opts *Opts) error {
	if opts == nil {
		panic("programmer error")
	}
	if err := validate(*opts); err != nil {
		return err
	}

	if opts.Client == nil {
		opts.Client = http.DefaultClient
	}
	if opts.Root == "" {
		opts.Root = os.TempDir()
	}
	if opts.UpdateInterval < time.Minute {
		opts.UpdateInterval = defaultUpdateInterval
	}
	if opts.UpdateRetention <= 1 {
		opts.UpdateRetention = defaultUpdateRetention
	}
	if opts.FullGCInterval < 3*time.Hour {
		opts.FullGCInterval = defaultFullGCInterval
	}
	if opts.RetryDelay == 0 {
		opts.RetryDelay = defaultRetryDelay
	}
	if opts.RetryMax == 0 {
		opts.RetryMax = defaultRetryMax
	}
	return nil
}

func validate(opts Opts) error {
	if opts.Store == nil || opts.Locker == nil || opts.MetadataStore == nil {
		return errors.New("must provide a Store, a Locker, and a MetadataStore")
	}
	if _, err := url.Parse(opts.URL); err != nil {
		return fmt.Errorf("invalid URL: %q", opts.URL)
	}
	return nil
}

func (u *Updater) Import(ctx context.Context, in io.Reader) (err error) {
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/vuln/Updater.Import")
	iter, iterErr := u.iterateFunc(in)
	// For each update operation in the JSON blob file.
	iter(func(op *driver.UpdateOperation, it jsonblob.RecordIter) bool {
		var ops map[string][]driver.UpdateOperation
		ops, err = u.store.GetUpdateOperations(ctx, op.Kind, op.Updater)
		if err != nil {
			return false
		}
		for _, o := range ops[op.Updater] {
			// This only helps if updaters don't keep something that
			// changes in the fingerprint.
			if o.Fingerprint == op.Fingerprint {
				zlog.Info(ctx).
					Str("updater", op.Updater).
					Msg("fingerprint match, skipping")
				return true
			}
		}
		zlog.Info(ctx).
			Str("updater", op.Updater).
			Str("kind", string(op.Kind)).
			Msg("importing update")
		var ref uuid.UUID
		count := 0
		switch op.Kind {
		case driver.VulnerabilityKind:
			ref, err = u.store.UpdateVulnerabilitiesIter(ctx, op.Updater, op.Fingerprint, func(yield func(*claircore.Vulnerability, error) bool) {
				// For each vulnerability in the update operation.
				it(func(v *claircore.Vulnerability, _ *driver.EnrichmentRecord) bool {
					count++
					// Offer one vulnerability to the datastore iterator.
					return yield(v, nil)
				})
				if err := iterErr(); err != nil {
					yield(nil, err)
				}
			})
		case driver.EnrichmentKind:
			ref, err = u.store.UpdateEnrichmentsIter(ctx, op.Updater, op.Fingerprint, func(yield func(*driver.EnrichmentRecord, error) bool) {
				// For each enrichment in the update operation.
				it(func(_ *claircore.Vulnerability, e *driver.EnrichmentRecord) bool {
					count++
					// Offer one enrichment to the datastore iterator.
					return yield(e, nil)
				})
				if err := iterErr(); err != nil {
					yield(nil, err)
				}
			})
		default:
			zlog.Warn(ctx).Str("kind", string(op.Kind)).Msg("unknown kind, skipping")
		}
		if err != nil {
			err = fmt.Errorf("updating %s: %w", op.Kind, err)
			return false
		}
		zlog.Info(ctx).
			Str("updater", op.Updater).
			Str("kind", string(op.Kind)).
			Str("ref", ref.String()).
			Int("count", count).
			Msg("update imported")
		return true
	})
	if err := iterErr(); err != nil {
		return fmt.Errorf("iterating on the reader: %w", err)
	}
	if err != nil {
		return err
	}
	return nil
}

// tryRemoveExiting attempts to remove any leftover update files present in the updater's
// root directory.
//
// It is assumed the previous updater used the same root directory.
// Any errors when attempting to remove pre-existing files are simply logged.
func (u *Updater) tryRemoveExiting() {
	ctx := zlog.ContextWithValues(u.ctx, "component", "matcher/updater/vuln/Updater.tryRemoveExiting")

	files, err := fs.Glob(os.DirFS(u.root), updateFilePattern)
	if err != nil {
		zlog.Warn(ctx).Err(err).Msg("searching for previous update files to remove")
		return
	}
	for _, file := range files {
		if err := os.Remove(filepath.Join(u.root, file)); err != nil {
			zlog.Warn(ctx).Err(err).Msgf("removing previous update file %s", file)
		}
	}
}

// Stop cancels the updater's context and prevents future update cycles.
// It does **not** cleanup any other resources. It is the user's responsibility to do so.
func (u *Updater) Stop() error {
	u.cancel()
	return nil
}

// Start periodically updates the vulnerability data.
// Each period is adjusted by some amount of jitter.
func (u *Updater) Start() error {
	ctx := zlog.ContextWithValues(u.ctx, "component", "matcher/updater/vuln/Updater.Start")

	if !u.skipGC {
		go u.runGCFullPeriodic()
	}

	if err := u.distManager.update(ctx); err != nil {
		zlog.Warn(ctx).Err(err).Msg("failed to initialize known-distributions")
	}

	// Start immediately, all matchers will compete to update each vulnerability
	// bundle if multi-bundle mode is on, or the single bundle.
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			zlog.Info(ctx).Msg("starting update")
			if err := u.Update(ctx); err != nil {
				zlog.Error(ctx).Err(err).Msg("errors encountered during updater run")
			}
			zlog.Info(ctx).Msg("completed update")

			timer.Reset(u.updateInterval + jitter())
		}
	}
}

// Initialized returns true if the vulnerability updater has fully initialized
// the vulnerability store.
func (u *Updater) Initialized(ctx context.Context) bool {
	if u.initialized.Load() {
		return true
	}
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/vuln/Updater.Initialized")
	ts, err := u.metadataStore.GetLastVulnerabilityUpdate(ctx)
	if err != nil {
		zlog.
			Warn(ctx).
			Err(err).
			Msg("did not get previous vuln update timestamp")
		return false
	}
	if ts.IsZero() {
		return false
	}
	zlog.Info(ctx).Msg("all vulnerability bundles were updated at least once: setting to initialized")
	u.initialized.Store(true)
	return true
}

func (u *Updater) KnownDistributions() []claircore.Distribution {
	return u.distManager.get()
}

// Update runs the full vulnerability update process.
//
// Note: periodic full GC will not be started.
func (u *Updater) Update(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/vuln/Updater.Update")

	var (
		updated bool
		err     error
	)
	if features.ScannerV4MultiBundle.Enabled() {
		updated, err = u.runMultiBundleUpdate(ctx)
	} else {
		updated, err = u.runSingleBundleUpdate(ctx)
	}
	if err != nil {
		return err
	}

	// Only bother running the GC when it's not disabled
	// and when the vulnerabilities have been updated.
	if !u.skipGC && updated {
		u.runGC(ctx)
	} else if !u.skipGC {
		// Only log if GC is enabled to reduce noise when GC is disabled.
		zlog.Info(ctx).Msg("no vulnerability updates: skipping GC")
	}

	return nil
}

// runSingleBundleUpdate updates the vulnerability data with one single bundle and
// returns a bool indicating if any updates actually happened.
func (u *Updater) runSingleBundleUpdate(ctx context.Context) (bool, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/vuln/Updater.runSingleBundleUpdate")

	// Use TryLock instead of Lock to prevent simultaneous updates.
	ctx, done := u.locker.TryLock(ctx, updateName)
	defer done()
	if err := ctx.Err(); err != nil {
		zlog.Info(ctx).
			Str("lock", updateName).
			Msg("did not obtain lock, skipping update run")
		return false, nil
	}

	prevTimestamp, err := u.metadataStore.GetLastVulnerabilityUpdate(ctx)
	if err != nil {
		zlog.Debug(ctx).
			Err(err).
			Msg("did not get previous vuln update timestamp")
		return false, err
	}
	zlog.Info(ctx).
		Str("timestamp", prevTimestamp.Format(http.TimeFormat)).
		Msg("previous vuln update")

	f, timestamp, err := u.fetch(ctx, prevTimestamp)
	if err != nil {
		return false, err
	}
	if f == nil {
		// Nothing to update at this time.
		zlog.Info(ctx).Msg("no new vulnerability update")
		return false, nil
	}
	defer func() {
		if err := f.Close(); err != nil {
			zlog.Error(ctx).Err(err).Msg("closing temp update file")
		}
		if err := os.Remove(f.Name()); err != nil {
			zlog.Error(ctx).Err(err).Msgf("removing temp update file %q", f.Name())
		}
	}()

	dec, err := zstd.NewReader(f)
	if err != nil {
		return false, fmt.Errorf("creating zstd reader: %w", err)
	}
	defer dec.Close()

	if err := u.importFunc(ctx, dec); err != nil {
		return false, err
	}

	if err := u.metadataStore.SetLastVulnerabilityUpdate(ctx, postgres.SingleBundleUpdateKey, timestamp); err != nil {
		return false, err
	}

	if err := u.distManager.update(ctx); err != nil {
		return false, fmt.Errorf("updating known-distributions: %w", err)
	}

	if u.initialized.CompareAndSwap(false, true) {
		zlog.Info(ctx).Msg("finished initial updater run: setting updater to initialized")
	}

	return true, nil
}

// runMultiBundleUpdate updates the vulnerability data with a multi-bundle and
// returns a bool indicating if any updates actually happened.
func (u *Updater) runMultiBundleUpdate(ctx context.Context) (bool, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/vuln/Updater.runMultiBundleUpdate")

	prevTime, err := u.metadataStore.GetLastVulnerabilityUpdate(ctx)
	if err != nil {
		zlog.Debug(ctx).
			Err(err).
			Msg("did not get previous vuln update timestamp")
		return false, err
	}
	zlog.Info(ctx).
		Time("timestamp", prevTime).
		Msg("previous vuln update")

	zipFile, zipTime, err := u.fetch(ctx, prevTime)
	if err != nil {
		return false, err
	}
	if zipFile == nil {
		// Nothing to update at this time.
		zlog.Info(ctx).Msg("no new vulnerability update")
		return false, nil
	}
	defer func() {
		if err := zipFile.Close(); err != nil {
			zlog.Error(ctx).Err(err).Msg("closing temp update file")
		}
		if err := os.Remove(zipFile.Name()); err != nil {
			zlog.Error(ctx).Err(err).Msgf("removing temp update file %q", zipFile.Name())
		}
	}()

	zipInfo, err := zipFile.Stat()
	if err != nil {
		return false, err
	}
	zipReader, err := zip.NewReader(zipFile, zipInfo.Size())
	if err != nil {
		return false, err
	}

	// Iterate through each vulnerability bundle in the .zip archive
	names := make([]string, 0, len(zipReader.File))
	for i := range zipReader.File {
		bundleF := zipReader.File[i]
		names = append(names, bundleF.Name)
		ctx := zlog.ContextWithValues(ctx, "bundle", bundleF.Name)
		zlog.Info(ctx).Msg("starting bundle update")
		if err := u.updateBundle(ctx, bundleF, zipTime, prevTime); err != nil {
			zlog.Error(ctx).Err(err).Msg("updating bundle failed")
			return false, fmt.Errorf("updating bundle %s: %w", bundleF.Name, err)
		}
		zlog.Info(ctx).Msg("completed bundle update")
	}

	// Clean updaters that were deleted (not in the zip and older than this update).
	// Safe to be run concurrently.
	err = u.metadataStore.GCVulnerabilityUpdates(ctx, names, zipTime)
	if err != nil {
		return false, fmt.Errorf("cleaning vuln updates: %w", err)
	}

	err = u.distManager.update(ctx)
	if err != nil {
		return false, fmt.Errorf("updating known-distributions: %w", err)
	}

	_ = u.Initialized(ctx)

	return true, nil
}

func (u *Updater) updateBundle(ctx context.Context, zipF *zip.File, zipTime time.Time, prevTime time.Time) error {
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/vuln/Updater.updateBundle")

	// Use TryLock to prevent simultaneous updates for the same bundle.
	lCtx, lDone := u.locker.TryLock(ctx, zipF.Name)
	defer lDone()
	if err := lCtx.Err(); err != nil {
		zlog.Info(ctx).Err(err).Msg("skipping: did not obtain lock")
		return nil
	}

	// Ensure there is an update timestamp for this bundle, and check if it's newer
	// than this update.
	lastTime, err := u.metadataStore.GetOrSetLastVulnerabilityUpdate(lCtx, zipF.Name, prevTime)
	if err != nil {
		return fmt.Errorf("querying last update: %w", err)
	}
	if !lastTime.Before(zipTime) {
		zlog.Info(ctx).
			Time("last_update_time", lastTime).
			Time("update_time", zipTime).
			Msg("skipping: last update time is greater or equal to the zip archive time")
		return nil
	}

	r, err := zipF.Open()
	if err != nil {
		return fmt.Errorf("opening bundle: %w", err)
	}
	defer func() {
		_ = r.Close()
	}()

	dec, err := zstd.NewReader(r)
	if err != nil {
		return fmt.Errorf("creating zstd reader: %w", err)
	}
	defer dec.Close()

	if err := u.importFunc(lCtx, dec); err != nil {
		return fmt.Errorf("importing vulnerabilities: %w", err)
	}

	err = u.metadataStore.SetLastVulnerabilityUpdate(lCtx, zipF.Name, zipTime)
	if err != nil {
		return fmt.Errorf("updating timestamp (import was successful): %w", err)
	}

	return nil
}

// fetch acquires the latest vulnerability data and writes it to the returned file.
// If successful, fetch returns the `Last-Modified` timestamp as well.
// If there have been no updates since prevTimestamp, the returned file will be nil.
// It is up to the user to close and delete the returned file.
func (u *Updater) fetch(ctx context.Context, prevTimestamp time.Time) (*os.File, time.Time, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "matcher/updater/vuln/Updater.fetch")

	var resp *http.Response
	defer closeResponse(resp)
	// After this loop, either resp != nil or we will have returned an error.
	for attempt := 1; attempt <= u.retryMax; attempt++ {
		zlog.Info(ctx).
			Str("url", u.url).
			Int("attempt", attempt).
			Msg("fetching vuln update")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.url, nil)
		if err != nil {
			return nil, time.Time{}, err
		}
		req.Header.Set(ifModifiedSinceHeader, prevTimestamp.Format(http.TimeFormat))
		// Request multi-bundle if multi-bundle is enabled.
		if features.ScannerV4MultiBundle.Enabled() {
			req.Header.Set("X-Scanner-V4-Accept", "application/vnd.stackrox.scanner-v4.multi-bundle+zip")
		}
		resp, err = u.client.Do(req)
		// If we haven't exhausted our connection refused attempts and the vuln URL is unavailable, retry.
		if attempt < u.retryMax && isConnectionRefused(err) {
			zlog.Error(ctx).
				Err(err).
				Str("delay", u.retryDelay.String()).
				Msg("retrying...")
			time.Sleep(u.retryDelay) // Wait for retryDelay before retrying
			continue
		}
		// If we have exhausted our retry attempts or there is some other kind of error, do not retry.
		if err != nil {
			return nil, time.Time{}, err
		}

		break // No errors, so don't retry.
	}

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotModified:
		// No updates.
		return nil, time.Time{}, nil
	default:
		return nil, time.Time{}, fmt.Errorf("received status code %d querying update endpoint", resp.StatusCode)
	}

	f, err := os.CreateTemp(u.root, updateFilePattern)
	if err != nil {
		return nil, time.Time{}, err
	}
	var succeeded bool
	defer func() {
		if succeeded {
			return
		}
		if err := f.Close(); err != nil {
			zlog.Error(ctx).Err(err).Msg("closing temp update file")
		}
		if err := os.Remove(f.Name()); err != nil {
			zlog.Error(ctx).Err(err).Msgf("removing temp update file %q", f.Name())
		}
	}()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("writing vulnerabilities to temp file: %w", err)
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, time.Time{}, fmt.Errorf("seeking temp update file to start: %w", err)
	}

	modTime := resp.Header.Get(lastModifiedHeader)
	timestamp, err := http.ParseTime(modTime)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("parsing %s header: %w", lastModifiedHeader, err)
	}
	succeeded = true
	return f, timestamp, nil
}

func jitter() time.Duration {
	return jitterMinutes[random.Intn(len(jitterMinutes))]
}

func closeResponse(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		utils.IgnoreError(resp.Body.Close)
	}
}

func isConnectionRefused(err error) bool {
	var opErr *net.OpError
	return errors.As(err, &opErr) && opErr.Op == "dial" && strings.Contains(opErr.Err.Error(), "connection refused")
}
