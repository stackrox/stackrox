package csaf

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/klauspost/compress/snappy"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/pkg/tmp"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/enricher/csaf/internal/zreader"
)

var (
	// compressedFileTimeout matches Claircore's VEX https://github.com/quay/claircore/blob/v1.5.34/rhel/vex/fetcher.go.
	compressedFileTimeout = 2 * time.Minute
)

// fingerprint is used to track the state of the changes.csv and deletions.csv endpoints.
//
// The spec (https://www.rfc-editor.org/rfc/rfc9110.html#name-etag) mentions
// that there is no need for the client to be aware of how each entity tag
// is constructed, however, it mentions that servers should avoid backslashes.
// Hence, the `\` character is used as a separator when stringifying.
type fingerprint struct {
	changesEtag, deletionsEtag string
	requestTime                time.Time
	version                    string
}

// parseFingerprint takes a generic driver.Fingerprint and creates a vex.fingerprint.
// The string format saved in the DB is returned by the fingerprint.String() method.
func parseFingerprint(in driver.Fingerprint) (*fingerprint, error) {
	fp := string(in)
	if fp == "" {
		return &fingerprint{}, nil
	}
	f := strings.Split(fp, `\`)
	if len(f) != 4 {
		return nil, errors.New("could not parse fingerprint")
	}
	rt, err := time.Parse(time.RFC3339, f[2])
	if err != nil {
		return nil, fmt.Errorf("could not parse fingerprint's requestTime: %w", err)
	}
	return &fingerprint{
		changesEtag:   f[0],
		deletionsEtag: f[1],
		requestTime:   rt,
		version:       f[3],
	}, nil
}

// String represents a fingerprint in string format with `\` acting as the delimiter.
func (fp *fingerprint) String() string {
	return fp.changesEtag + `\` + fp.deletionsEtag + `\` + fp.requestTime.Format(time.RFC3339) + `\` + fp.version
}

// FetchEnrichment implements driver.EnrichmentUpdater.
// This method fetches Red Hat's CSAF data (https://security.access.redhat.com/data/csaf/v2/advisories/),
// unless configured otherwise, and returns a Snappy-compressed file with each advisory's data written to each line.
func (e *Enricher) FetchEnrichment(ctx context.Context, hint driver.Fingerprint) (io.ReadCloser, driver.Fingerprint, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "enricher/csaf/Enricher/FetchEnrichment")
	fp, err := parseFingerprint(hint)
	if err != nil {
		return nil, hint, err
	}

	f, err := os.CreateTemp("", "enricher.csaf.")
	if err != nil {
		return nil, hint, err
	}

	cw := snappy.NewBufferedWriter(f)

	var success bool
	defer func() {
		if err := cw.Close(); err != nil {
			zlog.Warn(ctx).Err(err).Msg("unable to close snappy writer")
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			zlog.Warn(ctx).
				Err(err).
				Msg("unable to seek file back to start")
		}
		if !success {
			if err := f.Close(); err != nil {
				zlog.Warn(ctx).Err(err).Msg("unable to close spool")
			}
		}
	}()

	// We need to go after the full corpus of vulnerabilities
	// First we target the archive_latest.txt file.
	compressedURL, err := e.getCompressedFileURL(ctx)
	if err != nil {
		return nil, hint, fmt.Errorf("could not get compressed file URL: %w", err)
	}
	zlog.Debug(ctx).
		Str("url", compressedURL.String()).
		Msg("got compressed URL")

	fp.requestTime, err = e.getLastModified(ctx, compressedURL)
	if err != nil {
		return nil, hint, fmt.Errorf("could not get last-modified header: %w", err)
	}

	changed := map[string]bool{}
	err = e.processChanges(ctx, cw, fp, changed)
	if err != nil {
		return nil, hint, err
	}

	// Claircore processes deletions here; however, this is unnecessary for this enricher,
	// as there is no concept of a delta enricher. Processing deletions
	// only makes sense for the DeltaParse functions for delta updaters.

	rctx, cancel := context.WithTimeout(ctx, compressedFileTimeout)
	defer cancel()

	if compressedURL == nil {
		return nil, hint, errors.New("compressed file URL needs to be populated")
	}
	req, err := http.NewRequestWithContext(rctx, http.MethodGet, compressedURL.String(), nil)
	if err != nil {
		return nil, hint, err
	}

	res, err := e.c.Do(req)
	if err != nil {
		return nil, hint, err
	}
	defer utils.IgnoreError(res.Body.Close)

	err = checkResponse(res, http.StatusOK)
	if err != nil {
		return nil, hint, fmt.Errorf("unexpected response from latest compressed file: %w", err)
	}

	z, err := zreader.Reader(res.Body)
	if err != nil {
		return nil, hint, err
	}
	defer utils.IgnoreError(z.Close)
	r := tar.NewReader(z)

	var (
		h              *tar.Header
		buf, bc        bytes.Buffer
		entriesWritten int
	)
	for h, err = r.Next(); errors.Is(err, nil); h, err = r.Next() {
		buf.Reset()
		bc.Reset()
		if h.Typeflag != tar.TypeReg {
			continue
		}
		year, err := strconv.ParseInt(path.Dir(h.Name), 10, 64)
		if err != nil {
			return nil, hint, fmt.Errorf("error parsing year %w", err)
		}
		if year < lookBackToYear {
			continue
		}
		if changed[path.Base(h.Name)] {
			// We've already processed this file don't bother appending it to the output
			continue
		}
		buf.Grow(int(h.Size))
		if _, err := buf.ReadFrom(r); err != nil {
			return nil, hint, err
		}
		// Here we construct new-line-delimited JSON by first compacting the
		// JSON from the file and writing it to the bc buf, then writing a newline,
		// and finally writing all those bytes to the snappy.Writer.
		err = json.Compact(&bc, buf.Bytes())
		if err != nil {
			return nil, hint, fmt.Errorf("error compressing JSON %s: %w", h.Name, err)
		}
		bc.WriteByte('\n')
		if _, err := io.Copy(cw, &bc); err != nil {
			return nil, hint, fmt.Errorf("error writing compacted JSON to tmp file: %w", err)
		}
		entriesWritten++
	}
	if !errors.Is(err, io.EOF) {
		return nil, hint, fmt.Errorf("error reading tar contents: %w", err)
	}

	zlog.Debug(ctx).
		Str("enricher", e.Name()).
		Int("entries written", entriesWritten).
		Msg("finished writing compressed data to spool")

	fp.version = updaterVersion
	fp.requestTime = time.Now()
	success = true
	return f, driver.Fingerprint(fp.String()), nil
}

func (e *Enricher) getCompressedFileURL(ctx context.Context) (*url.URL, error) {
	latestURI, err := e.base.Parse(latestFile)
	if err != nil {
		return nil, err
	}
	latestReq, err := http.NewRequestWithContext(ctx, http.MethodGet, latestURI.String(), nil)
	if err != nil {
		return nil, err
	}
	latestRes, err := e.c.Do(latestReq)
	if err != nil {
		return nil, err
	}
	defer utils.IgnoreError(latestRes.Body.Close)

	err = checkResponse(latestRes, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("unexpected response from archive_latest.txt: %w", err)
	}

	body, err := io.ReadAll(latestRes.Body) // Fine to use as expecting small number of bytes.
	if err != nil {
		return nil, err
	}

	compressedFilename := string(body)
	compressedURL, err := e.base.Parse(compressedFilename)
	if err != nil {
		return nil, err
	}
	return compressedURL, nil
}

func (e *Enricher) getLastModified(ctx context.Context, cu *url.URL) (time.Time, error) {
	var empty time.Time
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, cu.String(), nil)
	if err != nil {
		return empty, err
	}

	res, err := e.c.Do(req)
	if err != nil {
		return empty, err
	}
	defer utils.IgnoreError(res.Body.Close)

	err = checkResponse(res, http.StatusOK)
	if err != nil {
		return empty, fmt.Errorf("unexpected HEAD response from latest compressed file: %w", err)
	}

	lm := res.Header.Get("last-modified")
	return time.Parse(http.TimeFormat, lm)
}

// processChanges deals with the published changes.csv, adding records
// to w means they are deemed to have changed since the compressed
// file was last processed. w and fp can be modified.
func (e *Enricher) processChanges(ctx context.Context, w io.Writer, fp *fingerprint, changed map[string]bool) error {
	tf, err := tmp.NewFile("", "enricher.stackrox.rhel-csaf-changes.")
	if err != nil {
		return err
	}
	defer utils.IgnoreError(tf.Close)

	uri, err := e.base.Parse(changesFile)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri.String(), nil)
	if err != nil {
		return err
	}
	if fp.changesEtag != "" {
		req.Header.Add("If-None-Match", fp.changesEtag)
	}
	res, err := e.c.Do(req)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(res.Body.Close)

	switch res.StatusCode {
	case http.StatusOK:
		if t := fp.changesEtag; t == "" || t != res.Header.Get("etag") {
			break
		}
		fallthrough
	case http.StatusNotModified:
		return nil
	default:
		return fmt.Errorf("unexpected response from changes.csv: %s", res.Status)
	}
	fp.changesEtag = res.Header.Get("etag")

	var r io.Reader = res.Body
	if _, err := io.Copy(tf, r); err != nil {
		return fmt.Errorf("unable to copy resp body to tempfile: %w", err)
	}
	if n, err := tf.Seek(0, io.SeekStart); err != nil || n != 0 {
		return fmt.Errorf("unable to seek changes to start: at %d, %w", n, err)
	}

	rd := csv.NewReader(tf)
	rd.FieldsPerRecord = 2
	rd.ReuseRecord = true
	var (
		l       int
		buf, bc bytes.Buffer
	)
	rec, err := rd.Read()
	for ; err == nil; rec, err = rd.Read() {
		buf.Reset()
		bc.Reset()
		if len(rec) != 2 {
			return errors.New("could not parse changes.csv file")
		}

		cvePath, uTime := rec[0], rec[1]
		year, err := strconv.ParseInt(path.Dir(cvePath), 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing year %w", err)
		}
		if year < lookBackToYear {
			continue
		}
		updatedTime, err := time.Parse(time.RFC3339, uTime)
		if err != nil {
			return fmt.Errorf("line %d: %w", l, err)
		}
		if updatedTime.Before(fp.requestTime) {
			continue
		}

		changed[path.Base(cvePath)] = true

		advisoryURI, err := e.base.Parse(cvePath)
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, advisoryURI.String(), nil)
		if err != nil {
			return fmt.Errorf("error creating advisory request %w", err)
		}

		// Use a func here as we're in a loop and want to make sure the
		// body is closed in all events.
		err = func() error {
			res, err := e.c.Do(req)
			if err != nil {
				return fmt.Errorf("error making advisory request %w", err)
			}
			defer utils.IgnoreError(res.Body.Close)
			err = checkResponse(res, http.StatusOK)
			if err != nil {
				return fmt.Errorf("unexpected response: %w", err)
			}

			// Add compacted JSON to buffer.
			_, err = buf.ReadFrom(res.Body)
			if err != nil {
				return fmt.Errorf("error reading from buffer: %w", err)
			}
			zlog.Debug(ctx).Str("url", advisoryURI.String()).Msg("copying body to file")
			err = json.Compact(&bc, buf.Bytes())
			if err != nil {
				return fmt.Errorf("error compressing JSON: %w", err)
			}

			bc.WriteByte('\n')
			_, _ = w.Write(bc.Bytes())
			l++
			return nil
		}()
		if !errors.Is(err, nil) {
			return err
		}
	}

	if !errors.Is(err, io.EOF) {
		return fmt.Errorf("error parsing the changes.csv file: %w", err)
	}
	return nil
}

// checkResponse takes a http.Response and a variadic of ints representing
// acceptable http status codes. The error returned will attempt to include
// some content from the server's response.
func checkResponse(resp *http.Response, acceptableCodes ...int) error {
	acceptable := false
	for _, code := range acceptableCodes {
		if resp.StatusCode == code {
			acceptable = true
			break
		}
	}
	if !acceptable {
		limitBody, err := io.ReadAll(io.LimitReader(resp.Body, 256))
		if err == nil {
			return fmt.Errorf("unexpected status code: %s (body starts: %q)", resp.Status, limitBody)
		}
		return fmt.Errorf("unexpected status code: %s", resp.Status)
	}
	return nil
}
