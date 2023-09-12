package cvss

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"os"
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
	tmpFile, err := os.CreateTemp("", "gzippedContent-")
	if err != nil {
		// TODO log error
		return err
	}
	defer os.RemoveAll(tmpFile.Name()) // Remove temp file after usage

	if err := runEnricher(ctx, u.enricher, tmpFile); err != nil {
		// TODO log error
		os.RemoveAll(tmpFile.Name())
		return err
	}
	tmpFile.Seek(0, 0) // Reset file pointer to the start

	return u.file.WriteContent(tmpFile)
}

func runEnricher(ctx context.Context, u driver.EnrichmentUpdater, w io.Writer) error {
	var rc io.ReadCloser
	var err error

	for i := 0; i < 5; i++ {
		rc, _, err = u.FetchEnrichment(ctx, fp)
		if err == nil {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After((2 << i) * time.Second):
		}
	}

	if err != nil {
		return err
	}
	defer rc.Close() // Close the reader once done

	return gzipContent(rc, w)
}

// gzipContent takes an io.Reader and writes the gzipped content to an io.Writer.
func gzipContent(r io.Reader, w io.Writer) error {
	gzw := gzip.NewWriter(w)
	_, err := io.Copy(gzw, r)
	if err != nil {
		return err
	}
	return gzw.Close()
}
