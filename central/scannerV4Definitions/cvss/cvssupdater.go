package cvss

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

var (
	pkgClient = &http.Client{}
	fp        driver.Fingerprint
)

type cvssUpdater struct {
	file        *file.File
	client      *http.Client
	downloadURL string
	interval    time.Duration
	once        sync.Once
	stopSig     concurrency.Signal
	enricher    *cvss.Enricher
}

func NewUpdaterWithEnricher(file *file.File, client *http.Client, downloadURL string, interval time.Duration) (*cvssUpdater, error) {
	e := &cvss.Enricher{}
	ctx := context.Background() // Or pass a context in if available.

	configFunc := func(cfg interface{}) error {
		c, ok := cfg.(*cvss.Config) // Type assertion for safety
		if !ok {
			return errors.New("invalid config type")
		}
		c.FeedRoot = &downloadURL
		return nil
	}

	err := e.Configure(ctx, configFunc, client)
	if err != nil {
		return nil, err
	}

	return &cvssUpdater{
		file:        file,
		client:      client,
		downloadURL: downloadURL,
		interval:    interval,
		stopSig:     concurrency.NewSignal(),
		enricher:    e,
	}, nil
}

func (u *cvssUpdater) Stop() {
	u.stopSig.Signal()
}

func (u *cvssUpdater) Start() {
	u.once.Do(func() {
		ctx := context.Background()
		u.update(ctx)
		go u.runForever()
	})
}

func (u *cvssUpdater) runForever() {
	t := time.NewTicker(u.interval)
	defer t.Stop()

	ctx := context.Background()

	for {
		select {
		case <-t.C:
			u.update(ctx)
		case <-u.stopSig.Done():
			return
		}
	}
}

func (u *cvssUpdater) update(ctx context.Context) {
	if err := u.doUpdate(ctx); err != nil {
		// TODO log error
	}
}

func (u *cvssUpdater) doUpdate(ctx context.Context) error {
	rc, _, err := runEnricher(ctx, u.enricher)
	if err != nil {
		// TODO log error
		return err
	}
	return u.file.WriteContent(rc)
}

func runEnricher(ctx context.Context, u driver.EnrichmentUpdater) (io.ReadCloser, driver.Fingerprint, error) {
	var rc io.ReadCloser
	var nfp driver.Fingerprint
	var err error

	for i := 0; i < 5; i++ {
		rc, nfp, err = u.FetchEnrichment(ctx, fp)
		if err == nil {
			break
		}

		select {
		case <-ctx.Done():
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
