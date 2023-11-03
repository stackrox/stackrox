package restore

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/VividCortex/ewma"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/ioutils"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/pkg/v2backuprestore"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/vbauerster/mpb/v4"
	"github.com/vbauerster/mpb/v4/decor"
	"golang.org/x/crypto/ssh/terminal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	readerWindowSize = 32 * (1 << 20) // 32 MB, should be more than enough while also not hurting

	// Timeout for gRPC requests
	grpcRequestTimeout = 30 * time.Second

	// Number of times to try to resume before giving up.
	resumeRetries = 10
	// Time to wait after every resume retry
	retryDelay = 6 * time.Second
)

var (
	defaultSpinner = []string{"⠇", "⠏", "⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧"}
	waitingSpinner = sliceutils.Concat(defaultSpinner, sliceutils.Reversed(defaultSpinner))
)

type v2Restorer struct {
	env           environment.Environment
	retryDeadline time.Time // does not affect ongoing transfers

	interrupt bool
	confirm   func() error

	processID     string
	lastAttemptID string

	dataReader    ioutils.SeekableReaderWithChecksum
	headerSize    int64
	totalDataSize int64

	transferProgressBar *mpb.Bar
	errorLine           statusLine
	statusLine          statusLine

	httpClient common.RoxctlHTTPClient
	dbClient   v1.DBServiceClient

	transferStatusText      string
	transferStatusTextMutex sync.RWMutex
}

func (cmd *centralDbRestoreCommand) newV2Restorer(confirm func() error, retryDeadline time.Time) (*v2Restorer, error) {
	conn, err := cmd.env.GRPCConnection()
	if err != nil {
		return nil, errors.Wrap(err, "could not establish gRPC connection to central")
	}

	dbClient := v1.NewDBServiceClient(conn)
	httpClient, err := cmd.env.HTTPClient(0)
	if err != nil {
		return nil, err
	}

	return &v2Restorer{
		env:           cmd.env,
		httpClient:    httpClient,
		dbClient:      dbClient,
		retryDeadline: retryDeadline,
		interrupt:     cmd.interrupt,
		confirm:       confirm,
	}, nil
}

func (r *v2Restorer) updateTransferStatus(cancelCond concurrency.Waitable) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	concurrency.WithLock(&r.transferStatusTextMutex, func() {
		r.transferStatusText = "Transferring data ..."
	})

	// Function will only run if r.transferProgressBar != nil
	lastVal := r.transferProgressBar.Current()
	speed := ewma.NewMovingAverage()

	for {
		select {
		case <-ticker.C:
			currVal := r.transferProgressBar.Current()
			progress := currVal - lastVal
			lastVal = currVal
			speed.Add(float64(progress))
			avgSpeed := int64(speed.Value())
			if avgSpeed <= 0 {
				continue
			}

			remaining := r.headerSize + r.totalDataSize - currVal
			remainingSecs := int64(remaining / avgSpeed)

			newText := fmt.Sprintf(
				"Transferring data at % 10.1f/s (ETA %02d:%02d:%02d)",
				decor.SizeB1024(avgSpeed),
				remainingSecs/3600,
				(remainingSecs%3600)/60,
				remainingSecs%60,
			)

			concurrency.WithLock(&r.transferStatusTextMutex, func() {
				r.transferStatusText = newText
			})
		case <-cancelCond.Done():
			return
		}
	}
}

func (r *v2Restorer) transferStatus() string {
	r.transferStatusTextMutex.RLock()
	defer r.transferStatusTextMutex.RUnlock()
	return r.transferStatusText
}

func (r *v2Restorer) Run(ctx context.Context, file *os.File) (*http.Response, error) {
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	nextReq, err := r.init(subCtx, file)
	if err != nil {
		return nil, err
	}

	r.statusLine.SetSpinner(waitingSpinner)
	r.statusLine.SetTextStatic("Initiating restore ...")

	termWidth, _, err := terminal.GetSize(int(os.Stderr.Fd())) //nolint:forbidigo // TODO(ROX-13473)
	if err == nil && termWidth > 40 {
		if termWidth > 120 {
			termWidth = 120
		}

		progressBarContainer := mpb.NewWithContext(subCtx, mpb.WithOutput(r.env.InputOutput().ErrOut()), mpb.WithWidth(termWidth))
		defer progressBarContainer.Wait()
		defer cancel() // canceling twice doesn't hurt, but we need to ensure this gets called before Wait() above.

		r.transferProgressBar = progressBarContainer.AddBar(
			r.headerSize+r.totalDataSize,
			mpb.PrependDecorators(decor.CountersKibiByte("% 10.1f / % 10.1f")),
			mpb.AppendDecorators(
				decor.Percentage(),
				decor.Name(" "),
				decor.Name(filepath.Base(file.Name())),
			),
		)

		progressBarContainer.Add(0, &r.errorLine)
		progressBarContainer.Add(0, &r.statusLine)

		// In case we resumed, initialized the transfer progress bar with the refill.
		pos, err := r.dataReader.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, errors.Wrap(err, "could not seek in stream")
		}
		r.transferProgressBar.IncrInt64(r.headerSize + pos)
		r.transferProgressBar.SetRefill(r.headerSize + pos)
	}

	for ctx.Err() == nil {
		r.errorLine.SetTextStatic("")
		r.statusLine.SetSpinner(defaultSpinner)
		concurrency.WithLock(&r.transferStatusTextMutex, func() {
			r.transferStatusText = "Initiating transfer ..."
		})
		r.statusLine.SetTextDynamic(r.transferStatus)

		transferInProgressSig := concurrency.NewSignal()
		if r.transferProgressBar != nil {
			go r.updateTransferStatus(&transferInProgressSig)
		} else {
			r.env.Logger().PrintfLn("Transferring data ...")
		}
		resp, err := r.performHTTPRequest(nextReq.WithContext(ctx))
		transferInProgressSig.Signal()

		if resp != nil {
			return resp, err
		}

		for i := 0; i < resumeRetries && err != nil; i++ {
			r.errorLine.SetTextStatic(err.Error())

			if !r.retryDeadline.IsZero() && time.Now().After(r.retryDeadline) {
				return nil, errox.InvariantViolation.New("absolute retry deadline has passed, please restart roxctl to resume the restore")
			}

			if r.transferProgressBar == nil {
				r.env.Logger().ErrfLn("Encountered a temporary error: %v. Retrying in %v (attempt %d out of %d)", err, retryDelay, i+1, resumeRetries)
			}

			continueTime := time.Now().Add(retryDelay)
			r.statusLine.SetSpinner(waitingSpinner)
			r.statusLine.SetTextDynamic(func() string {
				secondsLeft := time.Until(continueTime) / time.Second
				inText := "now"
				if secondsLeft > 0 {
					inText = fmt.Sprintf("in %ds", secondsLeft)
				}
				return fmt.Sprintf("Encountered a temporary error. Retrying %s...", inText)
			})

			if concurrency.WaitWithDeadline(ctx, continueTime) {
				return nil, ctx.Err()
			}

			nextReq, err = r.resumeAfterError(ctx)
			if err != nil && !retry.IsRetryable(err) {
				return nil, err
			}
		}
		if err != nil {
			return nil, errors.Wrapf(err, "could not resume restore after %d attempts, last error", resumeRetries)
		}
	}

	return nil, ctx.Err()
}

func (r *v2Restorer) performHTTPRequest(req *http.Request) (*http.Response, error) {
	if r.transferProgressBar != nil {
		req.Body = r.transferProgressBar.ProxyReader(req.Body)
	}
	return r.httpClient.Do(req)
}

func (r *v2Restorer) initDataReader(file *os.File, manifest *v1.DBExportManifest) error {
	r.totalDataSize = v2backuprestore.RestoreBodySize(manifest)

	dataReaders, err := dataReadersForManifest(file, manifest)
	if err != nil {
		return errors.Wrap(err, "could not get data readers for manifest")
	}

	r.dataReader, err = ioutils.NewSlidingReader(
		func() io.Reader { return ioutils.ChainReadersLazy(dataReaders...) },
		readerWindowSize,
		func() hash.Hash { return crc32.NewIEEE() },
	)
	return err
}

func (r *v2Restorer) initResume(ctx context.Context, file *os.File, activeStatus *v1.DBRestoreProcessStatus) (*http.Request, error) {
	r.processID = activeStatus.GetMetadata().GetId()
	manifest := activeStatus.GetMetadata().GetHeader().GetManifest()
	if err := r.initDataReader(file, manifest); err != nil {
		return nil, err
	}

	resumeInfo := activeStatus.GetResumeInfo()
	if resumeInfo == nil {
		if r.interrupt {
			r.env.Logger().PrintfLn("Active database restore process information")
			r.env.Logger().PrintfLn("===========================================")
			printStatus(r.env.Logger(), activeStatus)
			r.env.Logger().PrintfLn("")
			r.env.Logger().PrintfLn("The above restore process will be interrupted for resuming.")
			if err := r.confirm(); err != nil {
				return nil, err
			}

			subCtx, cancel := context.WithTimeout(ctx, grpcRequestTimeout)
			defer cancel()

			interruptResp, err := r.dbClient.InterruptRestoreProcess(subCtx, &v1.InterruptDBRestoreProcessRequest{
				ProcessId: r.processID,
				AttemptId: activeStatus.GetAttemptId(),
			})
			if err != nil {
				return nil, errors.Wrap(err, "could not interrupt ongoing restore process")
			}
			resumeInfo = interruptResp.GetResumeInfo()
		} else {
			return nil, errox.InvariantViolation.Newf("active restore process %s is not currently in resumable state. If you believe this process is stuck, use the `--interrupt` flag", activeStatus.GetMetadata().GetId())
		}
	}

	return r.prepareResumeRequest(resumeInfo)
}

func (r *v2Restorer) initNewProcess(ctx context.Context, file *os.File) (*http.Request, error) {
	subCtx, cancel := context.WithTimeout(ctx, grpcRequestTimeout)
	defer cancel()

	caps, err := r.dbClient.GetExportCapabilities(subCtx, &v1.Empty{})
	if err != nil {
		return nil, errors.Wrap(err, "could not get v2 DB restore capabilities")
	}

	supportedCompressionTypes := make(map[v1.DBExportManifest_EncodingType]struct{}, len(caps.GetSupportedEncodings()))
	for _, ct := range caps.GetSupportedEncodings() {
		supportedCompressionTypes[ct] = struct{}{}
	}

	manifest, err := assembleManifestFromZIP(file, supportedCompressionTypes)
	if err != nil {
		return nil, errors.Wrap(err, "could not create manifest from ZIP file")
	}

	format, _, err := v2backuprestore.DetermineFormat(manifest, caps.GetFormats())
	if err != nil {
		return nil, err
	}

	st, err := file.Stat()
	if err != nil {
		return nil, errors.Wrap(err, "could not stat input file")
	}

	header := &v1.DBRestoreRequestHeader{
		FormatName: format.GetFormatName(),
		Manifest:   manifest,
		LocalFile: &v1.DBRestoreRequestHeader_LocalFileInfo{
			Path:      file.Name(),
			BytesSize: st.Size(),
		},
	}

	headerBytes, err := proto.Marshal(header)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal restore header")
	}

	r.headerSize = int64(len(headerBytes))

	if err := r.initDataReader(file, manifest); err != nil {
		return nil, errors.Wrap(err, "could not get data readers for manifest")
	}

	bodyReader := ioutils.ChainReadersEager(bytes.NewReader(headerBytes), io.NopCloser(r.dataReader))
	req, err := r.httpClient.NewReq(http.MethodPost, "/db/v2/restore", bodyReader)
	if err != nil {
		return nil, errors.Wrap(err, "could not create restore HTTP request")
	}

	queryParams := req.URL.Query()
	queryParams.Set("headerLength", strconv.Itoa(len(headerBytes)))
	r.processID = uuid.NewV4().String()
	r.lastAttemptID = r.processID
	queryParams.Set("id", r.processID)
	req.URL.RawQuery = queryParams.Encode()

	return req, nil
}

func (r *v2Restorer) init(ctx context.Context, file *os.File) (*http.Request, error) {
	conn, err := r.env.GRPCConnection()
	if err != nil {
		return nil, errors.Wrap(err, "could not establish gRPC connection to central")
	}

	r.dbClient = v1.NewDBServiceClient(conn)

	subCtx, cancel := context.WithTimeout(ctx, checkCapsTimeout)
	defer cancel()
	activeProcessResp, err := r.dbClient.GetActiveRestoreProcess(subCtx, &v1.Empty{})
	if err != nil {
		if status.Convert(err).Code() == codes.Unimplemented {
			err = ErrV2RestoreNotSupported
		}
		return nil, err
	}

	activeStatus := activeProcessResp.GetActiveStatus()

	if activeStatus != nil {
		return r.initResume(ctx, file, activeStatus)
	}

	return r.initNewProcess(ctx, file)
}

func (r *v2Restorer) prepareResumeRequest(resumeInfo *v1.DBRestoreProcessStatus_ResumeInfo) (*http.Request, error) {
	if pos, err := r.dataReader.Seek(resumeInfo.GetPos(), io.SeekStart); err != nil {
		return nil, err
	} else if pos != resumeInfo.GetPos() {
		return nil, errox.NotFound.Newf("could not seek to resume position %d in data: data ends at position %d", resumeInfo.GetPos(), pos)
	} else if r.transferProgressBar != nil {
		// Interestingly, `SetCurrent` on a progress bar does not work as expected. It only works if it is used without
		// ever using `Incr` beforehand.
		r.transferProgressBar.IncrInt64(r.headerSize + pos - r.transferProgressBar.Current())
		r.transferProgressBar.SetRefill(r.headerSize + pos)
	}

	req, err := r.httpClient.NewReq(http.MethodPost, "/db/v2/resumerestore", io.NopCloser(r.dataReader))
	if err != nil {
		return nil, err
	}

	queryValues := req.URL.Query()
	queryValues.Set("id", r.processID)
	r.lastAttemptID = uuid.NewV4().String()
	queryValues.Set("attemptId", r.lastAttemptID)
	queryValues.Set("crc32", strconv.FormatUint(uint64(binary.BigEndian.Uint32(r.dataReader.CurrentChecksum())), 16))
	queryValues.Set("pos", strconv.FormatInt(resumeInfo.GetPos(), 10))
	req.URL.RawQuery = queryValues.Encode()

	return req, nil
}

func (r *v2Restorer) resumeAfterError(ctx context.Context) (*http.Request, error) {
	subCtx, cancel := context.WithTimeout(ctx, grpcRequestTimeout)
	defer cancel()

	// Get info about the currently active process to make sure it is still the current process.
	resp, err := r.dbClient.GetActiveRestoreProcess(subCtx, &v1.Empty{})
	if err != nil {
		// Unavailable and DeadlineExceeded indicate transport failures & timeouts. All other errors (permissions etc.)
		// are likely permanent.
		if code := status.Convert(err).Code(); code == codes.Unavailable || code == codes.DeadlineExceeded {
			err = retry.MakeRetryable(err)
		}
		return nil, err
	}

	activeProcess := resp.GetActiveStatus()

	if activeProcess.GetMetadata().GetId() != r.processID {
		return nil, errox.InvariantViolation.Newf("active restore process has changed: expected %s, got %s", r.processID, activeProcess.GetMetadata().GetId())
	}

	resumeInfo := activeProcess.GetResumeInfo()
	if resumeInfo == nil {
		// Interrupt the current attempt - the server might not have detected that the connection broke.
		// Note that specifying the attempt ID guarantees that we only interrupt the restore process if we were the one
		// who initiated it.
		subCtx, cancel = context.WithTimeout(ctx, grpcRequestTimeout)
		defer cancel()

		interruptResp, err := r.dbClient.InterruptRestoreProcess(subCtx, &v1.InterruptDBRestoreProcessRequest{
			ProcessId: r.processID,
			AttemptId: r.lastAttemptID,
		})
		if err != nil {
			return nil, err
		}

		resumeInfo = interruptResp.GetResumeInfo()
	}

	return r.prepareResumeRequest(resumeInfo)
}
