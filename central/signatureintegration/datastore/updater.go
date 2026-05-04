package datastore

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	maxResponseBodySize = 5 * 1024 * 1024 // 5 MB
	requestTimeout      = 60 * time.Second
)

type keyBundleUpdater struct {
	client   *http.Client
	url      string
	filePath string
	interval time.Duration
	stopSig  concurrency.Signal
	doneSig  concurrency.Signal
}

const minUpdateInterval = 1 * time.Minute

func newKeyBundleUpdater(url, filePath string, interval time.Duration) *keyBundleUpdater {
	if interval < minUpdateInterval {
		log.Warnf("Update interval %v is below minimum %v, clamping", interval, minUpdateInterval)
		interval = minUpdateInterval
	}
	return &keyBundleUpdater{
		client: &http.Client{
			Transport: proxy.RoundTripper(),
		},
		url:      url,
		filePath: filePath,
		interval: interval,
		stopSig:  concurrency.NewSignal(),
		doneSig:  concurrency.NewSignal(),
	}
}

func (u *keyBundleUpdater) Start() {
	go u.run()
}

func (u *keyBundleUpdater) Stop() {
	u.stopSig.Signal()
	<-u.doneSig.Done()
}

func (u *keyBundleUpdater) run() {
	log.Info("Starting Red Hat signing key bundle updater")
	defer log.Info("Stopped Red Hat signing key bundle updater")
	defer u.doneSig.Signal()

	if err := os.MkdirAll(filepath.Dir(u.filePath), 0700); err != nil {
		log.Errorf("Failed to create directory for key bundle file: %v", err)
		return
	}

	u.download()

	t := time.NewTimer(u.interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			u.download()
			t.Reset(u.interval)
		case <-u.stopSig.Done():
			return
		}
	}
}

func (u *keyBundleUpdater) download() {
	start := time.Now()
	err := u.doDownload()
	updaterDownloadDuration.Observe(time.Since(start).Seconds())

	if err != nil {
		log.Warnf("Failed to download Red Hat signing key bundle from %q: %v", u.url, err)
		updaterDownloadTotal.WithLabelValues("error").Inc()
	} else {
		updaterDownloadTotal.WithLabelValues("success").Inc()
		updaterLastSuccessTimestamp.SetToCurrentTime()
	}
}

func (u *keyBundleUpdater) doDownload() error {
	log.Debugf("Downloading Red Hat signing key bundle from %q", u.url)

	ctx, cancel := concurrency.DependentContext(context.Background(), &u.stopSig)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.url, nil)
	if err != nil {
		return errors.Wrap(err, "constructing request")
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "executing request")
	}
	defer utils.IgnoreError(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}

	limitedReader := io.LimitReader(resp.Body, maxResponseBodySize+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return errors.Wrap(err, "reading response body")
	}
	if len(body) > maxResponseBodySize {
		return errors.Errorf("response body exceeds maximum size of %d bytes", maxResponseBodySize)
	}

	if err := atomicWriteFile(u.filePath, body); err != nil {
		return errors.Wrap(err, "writing key bundle file")
	}

	log.Infof("Successfully downloaded Red Hat signing key bundle from %q", u.url)
	return nil
}

func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".key-bundle-*.tmp")
	if err != nil {
		return errors.Wrap(err, "creating temp file")
	}
	tmpPath := tmp.Name()

	if err := os.Chmod(tmpPath, 0600); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return errors.Wrap(err, "setting temp file permissions")
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return errors.Wrap(err, "writing temp file")
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return errors.Wrap(err, "syncing temp file")
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return errors.Wrap(err, "closing temp file")
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return errors.Wrap(err, "renaming temp file")
	}
	return nil
}
