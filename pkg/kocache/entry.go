package kocache

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/ioutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/utils"
)

type clock interface {
	Now() time.Time
	TimestampNow() timestamp.MicroTS
}

type systemClock struct {
}

func (d *systemClock) Now() time.Time {
	return time.Now()
}

func (d *systemClock) TimestampNow() timestamp.MicroTS {
	return timestamp.Now()
}

type entry struct {
	done concurrency.ErrorSignal

	references sync.WaitGroup
	data       *ioutils.RWBuf

	clock        clock
	creationTime time.Time
	lastAccess   timestamp.MicroTS // atomically set
}

func newEntry() *entry {
	return newEntryWithClock(&systemClock{})
}

func newEntryWithClock(clock clock) *entry {
	return &entry{
		clock:        clock,
		done:         concurrency.NewErrorSignal(),
		creationTime: clock.Now(),
		lastAccess:   clock.TimestampNow(),
	}
}

func (e *entry) DoneSig() concurrency.ReadOnlyErrorSignal {
	return &e.done
}

func (e *entry) Contents() (io.ReaderAt, int64, error) {
	if err, ok := e.done.Error(); err != nil {
		return nil, 0, err
	} else if !ok {
		return nil, 0, errors.New("content is not yet available")
	}
	return e.data.Contents()
}

func (e *entry) AcquireRef() {
	e.references.Add(1)
	e.lastAccess.StoreAtomic(e.clock.TimestampNow())
}

func (e *entry) ReleaseRef() {
	e.lastAccess.StoreAtomic(e.clock.TimestampNow())
	e.references.Done()
}

func (e *entry) Destroy() {
	e.references.Wait()
	concurrency.Wait(e.DoneSig()) // will happen after 60s max
	_ = e.data.Close()
}

func (e *entry) CreationTime() time.Time {
	return e.creationTime
}

func (e *entry) LastAccess() time.Time {
	return e.lastAccess.LoadAtomic().GoTime()
}

func (e *entry) IsError() bool {
	err, ok := e.done.Error()
	return ok && err != nil
}

func (e *entry) Populate(ctx context.Context, client httpClient, upstreamURL string, opts *options) {
	err := e.doPopulate(ctx, client, upstreamURL, opts)
	defer e.done.SignalWithError(err)
	e.lastAccess.StoreAtomic(e.clock.TimestampNow())
}

func (e *entry) doPopulate(ctx context.Context, client httpClient, upstreamURL string, opts *options) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstreamURL, nil)
	if err != nil {
		return errors.Errorf("creating HTTP request: %v", err)
	}

	if opts.ModifyRequest != nil {
		opts.ModifyRequest(req)
	}

	resp, err := client.Do(req)
	if err != nil {
		return errors.Errorf("making upstream request: %v", err)
	}
	defer utils.IgnoreError(resp.Body.Close)

	// We really are not interested in anything other than 200. We can't assume the body being valid data on any 2xx
	// or 3xx status code.
	// Note that Golang's HTTP client by default follows redirects.
	if resp.StatusCode != http.StatusOK {
		// If the probe does not exist, we may receive 403 Forbidden due to security constraints
		// on the storage. Convert this to errProbeNotFound for better visibility of this scenario.
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
			return errProbeNotFound
		}
		return errors.Errorf("upstream HTTP request returned status %s", resp.Status)
	}

	contentLen := resp.ContentLength
	e.data = ioutils.NewRWBuf(ioutils.RWBufOptions{
		MemLimit:  opts.ObjMemLimit,
		HardLimit: opts.ObjHardLimit,
	})

	if n, err := io.Copy(e.data, resp.Body); err != nil {
		return errors.Errorf("downloading data from upstream: %v", err)
	} else if contentLen > 0 && n != contentLen {
		return errors.Errorf("unexpected number of bytes read from upstream: expected %d bytes, got %d", contentLen, n)
	}
	return nil
}
