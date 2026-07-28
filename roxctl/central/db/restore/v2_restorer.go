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

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/ioutils"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/pkg/v2backuprestore"
	"github.com/stackrox/rox/roxctl/central/db/transfer"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
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

	progressBar *transfer.ProgressBar

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
		return nil, errors.Wrap(err, "creating HTTP client for central database restore")
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

	lastVal := r.progressBar.Current()
	var avgSpeed float64
	const ewmaDecay = 2.0 / 31.0 // EWMA with age=30

	for {
		select {
		case <-ticker.C:
			currVal := r.progressBar.Current()
			progress := float64(currVal - lastVal)
			lastVal = currVal
			if avgSpeed == 0 {
				avgSpeed = progress
			} else {
				avgSpeed += ewmaDecay * (progress - avgSpeed)
			}
			speedInt := int64(avgSpeed)
			if speedInt <= 0 {
				continue
			}

			remaining := r.headerSize + r.totalDataSize - currVal
			remainingSecs := remaining / speedInt

			newText := fmt.Sprintf(
				"Transferring data at %s/s (ETA %02d:%02d:%02d)",
				transfer.FormatSize(speedInt),
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

	r.env.Logger().PrintfLn("Initiating restore ...")

	bar, shutdown := transfer.CreateProgressBar(subCtx, filepath.Base(file.Name()), r.headerSize+r.totalDataSize, r.env.InputOutput().ErrOut())
	r.progressBar = bar
	defer shutdown()

	pos, err := r.dataReader.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, errors.Wrap(err, "could not seek in stream")
	}
	if pos > 0 {
		r.progressBar.SetCurrent(r.headerSize + pos)
	}

	for ctx.Err() == nil {
		concurrency.WithLock(&r.transferStatusTextMutex, func() {
			r.transferStatusText = "Initiating transfer ..."
		})

		transferInProgressSig := concurrency.NewSignal()
		go r.updateTransferStatus(&transferInProgressSig)
		resp, err := r.performHTTPRequest(nextReq.WithContext(ctx))
		transferInProgressSig.Signal()

		if resp != nil {
			return resp, err
		}

		for i := 0; i < resumeRetries && err != nil; i++ {
			if !r.retryDeadline.IsZero() && time.Now().After(r.retryDeadline) {
				return nil, errox.InvariantViolation.New("absolute retry deadline has passed, please restart roxctl to resume the restore")
			}

			r.env.Logger().ErrfLn("Encountered a temporary error: %v. Retrying in %v (attempt %d out of %d)", err, retryDelay, i+1, resumeRetries)

			if concurrency.WaitWithDeadline(ctx, time.Now().Add(retryDelay)) {
				return nil, errors.Wrap(ctx.Err(), "waiting to retry restore")
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

	return nil, errors.Wrap(ctx.Err(), "context error during restore")
}

func (r *v2Restorer) performHTTPRequest(req *http.Request) (*http.Response, error) {
	if r.progressBar != nil {
		req.Body = r.progressBar.ProxyReader(req.Body)
	}
	resp, err := r.httpClient.Do(req)
	return resp, errors.Wrap(err, "executing restore HTTP request")
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
	return errors.Wrap(err, "creating restore data reader")
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
		return nil, errors.Wrap(err, "determining restore format")
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

	headerBytes, err := header.MarshalVT()
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
		return nil, errors.Wrap(err, "seeking to restore resume position")
	} else if pos != resumeInfo.GetPos() {
		return nil, errox.NotFound.Newf("could not seek to resume position %d in data: data ends at position %d", resumeInfo.GetPos(), pos)
	} else if r.progressBar != nil {
		r.progressBar.SetCurrent(r.headerSize + pos)
	}

	req, err := r.httpClient.NewReq(http.MethodPost, "/db/v2/resumerestore", io.NopCloser(r.dataReader))
	if err != nil {
		return nil, errors.Wrap(err, "creating restore resume request")
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
			err = common.MakeRetryable(err)
		}
		return nil, errors.Wrap(err, "getting active restore process")
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
			return nil, errors.Wrap(err, "interrupting restore process")
		}

		resumeInfo = interruptResp.GetResumeInfo()
	}

	return r.prepareResumeRequest(resumeInfo)
}
