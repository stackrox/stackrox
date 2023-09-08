package scannerV4Definitions

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/quay/claircore/enricher/cvss"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	lastModifiedHeader    = "Last-Modified"
	ifModifiedSinceHeader = "If-Modified-Since"
)

var (
	pkgClient = &http.Client{}

	fp driver.Fingerprint
)

// updater periodically updates a file by downloading the contents from the downloadURL.
type updater struct {
	file *file.File

	client      *http.Client
	downloadURL string
	interval    time.Duration
	once        sync.Once
	stopSig     concurrency.Signal
}

// newUpdater creates a new updater.
func newUpdater(file *file.File, client *http.Client, downloadURL string, interval time.Duration) *updater {
	return &updater{
		file:        file,
		client:      client,
		downloadURL: downloadURL,
		interval:    interval,
		stopSig:     concurrency.NewSignal(),
	}
}

// Stop stops the updater.
func (u *updater) Stop() {
	u.stopSig.Signal()
}

// Start starts the updater.
// The updater is only started once.
func (u *updater) Start() {
	u.once.Do(func() {
		ctx := context.Background() // Creating a background context

		// Run the first update in a blocking-manner.
		u.update(ctx)
		go u.runForever()
	})
}

func (u *updater) runForever() {
	t := time.NewTicker(u.interval)
	defer t.Stop()

	ctx := context.Background() // Creating a background context

	for {
		select {
		case <-t.C:
			u.update(ctx) // Passing the context to update function
		case <-u.stopSig.Done():
			return
		}
	}
}

func (u *updater) update(ctx context.Context) {
	if err := u.doUpdate(ctx); err != nil {
		// TODO log error
	}
}

func (u *updater) doUpdate(ctx context.Context) error {
	e := &cvss.Enricher{}
	// Create a custom config unmarshaler
	configFunc := func(cfg interface{}) error {
		c, ok := cfg.(*cvss.Config) // Type assertion for safety
		if !ok {
			return errors.New("invalid config type")
		}
		c.FeedRoot = &u.downloadURL
		return nil
	}

	err := e.Configure(ctx, configFunc, pkgClient)
	if err != nil {
		// TODO log error
		return err
	}

	_, _, err = runEnricher(ctx, e)
	if err != nil {
		// TODO log error
		return err
	}
	return nil
}

func runEnricher(ctx context.Context, u driver.EnrichmentUpdater) (io.ReadCloser, driver.Fingerprint, error) {
	var rc io.ReadCloser
	var nfp driver.Fingerprint
	var err error

	// Debounce any network hiccups.
	for i := 0; i < 5; i++ {
		rc, nfp, err = u.FetchEnrichment(ctx, fp)
		if err == nil {
			break
		}

		select {
		case <-ctx.Done():
			// Return the error instead of calling t.Fatal
			return nil, nfp, ctx.Err()
		case <-time.After((2 << i) * time.Second):
		}
	}

	if err != nil {
		return nil, nfp, err
	}

	defer func() {
		if err := rc.Close(); err != nil {
			// TODO log error
			return
		}
	}()

	return rc, nfp, nil
}
