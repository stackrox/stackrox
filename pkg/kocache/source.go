package kocache

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/ioutils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()
)

const (
	probeAccessCheckResource = "/meta.json"
)

// Currently we only require that the resource `probeAccessCheckResource`
// can be decoded as a JSON object.
// For future use cases we might need to populate this metadata struct
// with actual fields.
type resourceMeta struct {
}

func (c *koCache) LoadProbe(ctx context.Context, filePath string) (io.ReadCloser, int64, error) {
	if c.upstreamBaseURL == "" {
		// Probably offline mode.
		return nil, 0, nil
	}

	e, err := c.getOrAddEntry(filePath)
	if err != nil || e == nil {
		log.Error(err)
		return nil, 0, err
	}
	releaseRef := true
	defer func() {
		if releaseRef {
			e.ReleaseRef()
		}
	}()

	if !concurrency.WaitInContext(e.DoneSig(), ctx) {
		log.Errorf("context error waiting for download of %s from upstream: %v", filePath, ctx.Err())
		return nil, 0, errors.Wrap(ctx.Err(), "context error waiting for download from upstream")
	}

	data, size, err := e.Contents()
	if err != nil {
		if errors.Is(err, errNotFound) {
			log.Errorf("probe %s does not exist in upstream %s", filePath, c.upstreamBaseURL)
			err = nil
		}
		return nil, 0, err
	}

	log.Infof("loading probe %s from upstream %s", filePath, c.upstreamBaseURL)
	// We need to make sure that `e` does not get destroyed before reading from the reader is complete, so shift
	// the responsibility to release the reference to the `Close()` method of the returned reader.
	dataReader := io.NewSectionReader(data, 0, size)

	dataReaderWithCloser := ioutils.ReaderWithCloser(dataReader, func() error {
		e.ReleaseRef()
		return nil
	})
	releaseRef = false // prevent releasing reference upon return
	return dataReaderWithCloser, size, nil
}

func (c *koCache) checkProbeDownloadSite(ctx context.Context) error {
	url := fmt.Sprintf("%s%s", strings.TrimRight(c.upstreamBaseURL, "/"), probeAccessCheckResource)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return errors.Wrapf(err, "Failed to create HTTP request for accessing resource %s", url)
	}
	resp, err := c.upstreamClient.Do(req)
	if err != nil {
		log.Errorf("errors downloading probe from %s: %v", url, err)
		return err
	}
	defer utils.IgnoreError(resp.Body.Close)
	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Failed to access resource %s: Unexpected HTTP response status: %s", url, resp.Status)
		log.Error(msg)
		return errors.New(msg)
	}

	var meta resourceMeta
	err = json.NewDecoder(resp.Body).Decode(&meta)
	if err != nil {
		return errors.Wrapf(err, "Failed to access resource %s: Decoding error", url)
	}

	return nil
}

func (c *koCache) verifyProbeDownloadSiteReachable(ctx context.Context) (bool, error) {
	err := c.checkProbeDownloadSite(ctx)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *koCache) IsAvailable(ctx context.Context) (bool, error) {
	isAvailable, err := c.verifyProbeDownloadSiteReachable(ctx)
	if err != nil {
		return false, err
	}
	return isAvailable, nil
}
