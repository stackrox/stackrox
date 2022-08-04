package transfer

import (
	"context"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/vbauerster/mpb/v4"
)

type progressWatchReader struct {
	reader       io.Reader
	lastActivity timestamp.MicroTS
	progressBar  *mpb.Bar
}

func newProgressWatchReader(r io.Reader, progressBar *mpb.Bar) *progressWatchReader {
	return &progressWatchReader{
		reader:       r,
		lastActivity: timestamp.Now(),
		progressBar:  progressBar,
	}
}

func (r *progressWatchReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	count, err := r.reader.Read(p)
	if err == nil {
		atomic.StoreInt64((*int64)(&r.lastActivity), int64(timestamp.Now()))
		r.progressBar.IncrBy(len(p))
	}

	return count, err
}

func (r *progressWatchReader) Close() error {
	if rc, ok := r.reader.(io.ReadCloser); ok {
		return rc.Close()
	}
	r.progressBar.SetTotal(r.progressBar.Current(), true)
	atomic.StoreInt64((*int64)(&r.lastActivity), int64(timestamp.InfiniteFuture))
	return nil
}

func (r *progressWatchReader) GetLastActivityTime() time.Time {
	ts := timestamp.MicroTS(atomic.LoadInt64((*int64)(&r.lastActivity)))
	if ts == timestamp.InfiniteFuture {
		return time.Now()
	}
	return ts.GoTime()
}

// ViaHTTP transfers a file via HTTP, showing a progress indicator, if possible.
func ViaHTTP(req *http.Request, client common.RoxctlHTTPClient, earliestDeadline time.Time, idleTimeout time.Duration) (*http.Response, error) {
	totalLen := req.ContentLength
	name := "Uploading..."
	if req.Body == nil {
		return nil, errox.InvalidArgs.New("transfer request must have a non-nil body")
	}
	if srcFile, ok := req.Body.(interface{ Stat() (os.FileInfo, error) }); ok {
		srcFileStat, err := srcFile.Stat()
		if err != nil {
			return nil, errors.Wrap(err, "could not stat source file")
		}
		totalLen = srcFileStat.Size()
		name = srcFileStat.Name()
	}

	errSig := concurrency.NewErrorSignal()
	defer errSig.Signal()

	ctx, cancel := concurrency.DependentContext(req.Context(), &errSig)
	defer cancel()
	req = req.WithContext(ctx)

	bar, shutdown := createProgressBars(ctx, name, totalLen)
	defer shutdown()

	newBody := newProgressWatchReader(req.Body, bar)
	req.Body = newBody

	go watchdog(&errSig, earliestDeadline, newBody.GetLastActivityTime, idleTimeout)
	resp, err := client.Do(req)
	if err != nil {
		if sigErr := errSig.Err(); sigErr != nil {
			err = sigErr
		}
	}
	return resp, err
}

// Copy transfers data via io.Copy, showing a progress indicator, if possible.
func Copy(ctx context.Context, ctxCancel context.CancelFunc, fileName string, totalSize int64, from io.Reader, to io.Writer, earliestDeadline time.Time, idleTimeout time.Duration) error {
	errSig := concurrency.NewErrorSignal()
	defer errSig.Signal()
	concurrency.CancelContextOnSignal(ctx, ctxCancel, &errSig)

	bar, shutdown := createProgressBars(ctx, fileName, totalSize)
	defer shutdown()

	watchReader := newProgressWatchReader(from, bar)
	go watchdog(&errSig, earliestDeadline, watchReader.GetLastActivityTime, idleTimeout)

	if _, err := io.Copy(to, watchReader); err != nil {
		if sigErr := errSig.Err(); sigErr != nil {
			err = sigErr
		}
		return err
	}
	return nil
}
