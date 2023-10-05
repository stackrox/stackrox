package manager

import (
	"context"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/backup"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/ioutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	defaultReattachTimeout = 24 * time.Hour
)

// RestoreProcess provides a handle to ongoing database restore processes.
type RestoreProcess interface {
	Metadata() *v1.DBRestoreProcessMetadata
	ProtoStatus() *v1.DBRestoreProcessStatus
	Completion() concurrency.ErrorWaitable
	Cancel()

	// Interrupt interrupts the currently active restore attempt, provided its ID matches the given attempt ID.
	// This function will behave as if the interruption was successful even if no attempt is currently active.
	Interrupt(ctx context.Context, attemptID string) (*v1.DBRestoreProcessStatus_ResumeInfo, error)

	// Resume tries to attach the given data reader at the given position, using the previous content checksum for
	// verifying integrity. If successful, the given attempt ID will be used to refer to this attempt.
	Resume(ctx context.Context, attemptID string, data io.Reader, pos int64, checksum []byte) (concurrency.ErrorWaitable, error)
}

type restoreFile struct {
	manifestFile *v1.DBExportManifest_File
	handlerFunc  common.RestoreFileHandlerFunc
}

type restoreProcess struct {
	// mutex protects all fields in the current block
	mutex           sync.RWMutex
	reattachPos     int64                // only valid if attachment is possible
	reattachC       chan reattachRequest // only non-nil if attachment is possible
	reattachableSig concurrency.Signal   // triggered whenever a new reader can be attached, reset after it has been attached
	attemptID       string               // for safe interruptions

	metadata *v1.DBRestoreProcessMetadata // static, populated on construction
	files    []*restoreFile               // static, populated on construction

	// Resumable data reader fields
	data            io.ReadCloser
	detachEvents    <-chan ioutils.ReaderDetachmentEvent
	reattachTimeout time.Duration

	started           concurrency.Flag        // prevents multiple starts of the same process
	cancelSig         concurrency.Signal      // trigger to cancel the current process
	completionSig     concurrency.ErrorSignal // triggered after process has finished running (successful or otherwise)
	currentAttemptSig concurrency.ErrorSignal // trigger to cancel current *attempt*

	// For statistics (atomic access; approximate only)
	bytesRead      int64
	filesProcessed int64

	// Informs whether processing a RocksDB or Postgres backup bundle
	postgresBundle bool
}

func newRestoreProcess(ctx context.Context, id string, header *v1.DBRestoreRequestHeader, handlerFuncs []common.RestoreFileHandlerFunc, data io.Reader) (*restoreProcess, error) {
	var postgresBundle bool
	mfFiles := header.GetManifest().GetFiles()
	if len(mfFiles) != len(handlerFuncs) {
		return nil, utils.ShouldErr(errors.Errorf("mismatch: %d handler functions provided for %d files in the manifest", len(handlerFuncs), len(mfFiles)))
	}

	files := make([]*restoreFile, 0, len(mfFiles))
	for i, manifestFile := range mfFiles {
		// Check to see if we are processing a postgres bundle
		if manifestFile.GetName() == backup.PostgresFileName {
			postgresBundle = true
		}
		files = append(files, &restoreFile{
			manifestFile: manifestFile,
			handlerFunc:  handlerFuncs[i],
		})
	}

	metadata := &v1.DBRestoreProcessMetadata{
		Id:        id,
		Header:    header,
		StartTime: types.TimestampNow(),
	}

	if identity := authn.IdentityFromContextOrNil(ctx); identity != nil {
		metadata.InitiatingUserName = identity.User().GetUsername()
	}

	resumableDataReader, initAttach, detachEvents := ioutils.NewResumableReader(crc32.NewIEEE())
	if err := initAttach.Attach(data, 0, nil); err != nil {
		return nil, utils.ShouldErr(errors.Wrap(err, "could not attach initial reader to resumable reader"))
	}

	p := &restoreProcess{
		metadata:        metadata,
		files:           files,
		detachEvents:    detachEvents,
		reattachTimeout: defaultReattachTimeout,

		cancelSig:       concurrency.NewSignal(),
		completionSig:   concurrency.NewErrorSignal(),
		reattachableSig: concurrency.NewSignal(),

		postgresBundle: postgresBundle,
	}

	p.data = ioutils.NewCountingReader(resumableDataReader, &p.bytesRead)

	return p, nil
}

func (p *restoreProcess) Metadata() *v1.DBRestoreProcessMetadata {
	return p.metadata
}

func (p *restoreProcess) Launch(tempOutputDir, finalDir string) (concurrency.ErrorWaitable, error) {
	if p.started.TestAndSet(true) {
		return nil, errors.New("restore process has already been started")
	}

	p.currentAttemptSig.Reset()
	currAttemptDone := p.currentAttemptSig.Snapshot()

	go p.run(tempOutputDir, finalDir)
	return currAttemptDone, nil
}

func (p *restoreProcess) run(tempOutputDir, finalDir string) {
	defer utils.IgnoreError(p.data.Close)
	defer p.cancelSig.Signal()

	// Make sure the process runs in a context that respects the stop signal.
	ctx, cancel := concurrency.DependentContext(context.Background(), &p.cancelSig)
	defer cancel()

	go p.resumeCtrl()

	err := p.doRun(ctx, tempOutputDir, finalDir)
	p.currentAttemptSig.SignalWithError(err)
	p.completionSig.SignalWithError(err)
}

func (p *restoreProcess) doRun(ctx context.Context, tempOutputDir, finalDir string) error {
	// If processing a postgres bundle, do not create the restore directories
	if !p.postgresBundle {
		if err := os.MkdirAll(tempOutputDir, 0700); err != nil {
			return errors.Wrapf(err, "could not create temporary output directory %s", tempOutputDir)
		}
	}

	// store if Postgres bundle here
	restoreCtx := newRestoreProcessContext(ctx, tempOutputDir, p.postgresBundle)

	if err := p.processFiles(restoreCtx); err != nil {
		return err
	}

	if err := restoreCtx.waitForAsyncChecks(); err != nil {
		return err
	}

	// If processing a postgres bundle, do not update the restore symlink
	if !p.postgresBundle {
		if err := os.Symlink(filepath.Base(tempOutputDir), finalDir); err != nil {
			return errors.Wrapf(err, "failed to atomically create a symbolic link to restore directory %s", tempOutputDir)
		}
	}

	return nil
}

func (p *restoreProcess) Cancel() {
	p.cancelSig.Signal()
	p.currentAttemptSig.SignalWithError(errors.New("restore canceled"))
}

func (p *restoreProcess) Interrupt(ctx context.Context, attemptID string) (*v1.DBRestoreProcessStatus_ResumeInfo, error) {

	reattachCond, resumeInfo, err := concurrency.WithLock3(&p.mutex, func() (concurrency.Waitable, *v1.DBRestoreProcessStatus_ResumeInfo, error) {
		if p.reattachC != nil {
			// Process is already interrupted
			return nil, &v1.DBRestoreProcessStatus_ResumeInfo{
				Pos: p.reattachPos,
			}, nil
		}

		if p.attemptID != attemptID {
			return nil, nil,
				errors.Errorf("provided attempt ID %q does not match ID %s of current attempt", attemptID, p.attemptID)
		}

		p.currentAttemptSig.SignalWithError(errors.New("attempt canceled"))
		return p.reattachableSig.Snapshot(), nil, nil
	})

	if reattachCond == nil {
		// function passed to WithLock above returned early, so return early here.
		return resumeInfo, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-reattachCond.Done():
	}

	pos, reattachC := p.getReattachC()
	if reattachC == nil {
		return nil, errors.New("process was interrupted, but has already been resumed")
	}
	return &v1.DBRestoreProcessStatus_ResumeInfo{
		Pos: pos,
	}, nil
}

func (p *restoreProcess) getReattachC() (int64, chan<- reattachRequest) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.reattachC == nil {
		return 0, nil
	}

	return p.reattachPos, p.reattachC
}

type reattachRequest struct {
	attemptID string
	reader    io.Reader
	pos       int64
	checksum  []byte

	respC chan reattachResponse
}

type reattachResponse struct {
	attemptDone concurrency.ErrorWaitable
	err         error
}

func (p *restoreProcess) Resume(ctx context.Context, attemptID string, data io.Reader, pos int64, checksum []byte) (concurrency.ErrorWaitable, error) {
	subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, reattachC := p.getReattachC()
	if reattachC == nil {
		return nil, errors.New("restore process is not currently available for resuming")
	}

	respC := make(chan reattachResponse, 1)

	req := reattachRequest{
		attemptID: attemptID,
		reader:    data,
		pos:       pos,
		checksum:  checksum,
		respC:     respC,
	}

	select {
	case reattachC <- req:
	case <-p.completionSig.Done():
		return nil, errors.New("could not resume restore process: process was stopped")
	case <-subCtx.Done():
		return nil, errors.Wrap(subCtx.Err(), "timed out trying to resume (the restore process might have been resumed otherwise, or timed out)")
	}

	select {
	case resp := <-respC:
		if resp.err != nil {
			return nil, errors.Wrap(resp.err, "could not resume restore process")
		}
		return resp.attemptDone, nil
	case <-subCtx.Done():
		return nil, errors.Wrap(subCtx.Err(), "timed out waiting for resume response")
	}
}

func (p *restoreProcess) onReaderDetached(event ioutils.ReaderDetachmentEvent) {
	_ = ioutils.Close(event.DetachedReader())

	if event.ReadError() == io.EOF {
		if err := event.Finish(io.EOF); err != nil {
			utils.Should(err)
		}
		return
	}

	timer := time.NewTimer(p.reattachTimeout)
	defer timeutil.StopTimer(timer)

	reattachC := make(chan reattachRequest)
	concurrency.WithLock(&p.mutex, func() {
		p.reattachC = reattachC
		p.reattachPos = event.Position()
		p.attemptID = ""
	})

	p.reattachableSig.Signal()
	defer p.reattachableSig.Reset()

	var winningAttemptID string
	defer concurrency.WithLock(&p.mutex, func() {
		p.reattachC = nil
		p.reattachPos = 0
		p.attemptID = winningAttemptID
	})

	for {
		select {
		case <-p.cancelSig.Done():
			if err := event.Finish(errors.New("process canceled")); err != nil {
				utils.Should(err)
			}
			return
		case <-timer.C:
			if err := event.Finish(errors.Errorf("timeout: no new data stream attached after %v", p.reattachTimeout)); err != nil {
				utils.Should(err)
			}
			return
		case req := <-reattachC:
			reattachErr := event.Attach(req.reader, req.pos, req.checksum)
			// req.respC is buffered with capacity 1 and only written to once, so the following sends will never block.
			if reattachErr == nil {
				p.currentAttemptSig.Reset()
				req.respC <- reattachResponse{
					attemptDone: p.currentAttemptSig.Snapshot(),
				}
				winningAttemptID = req.attemptID
				return
			}
			req.respC <- reattachResponse{
				err: reattachErr,
			}
		}
	}
}

func (p *restoreProcess) resumeCtrl() {
	for event := range p.detachEvents {
		p.onReaderDetached(event)
	}
}

func (p *restoreProcess) Completion() concurrency.ErrorWaitable {
	return &p.completionSig
}

func (p *restoreProcess) processFiles(ctx *restoreProcessContext) error {
	for _, file := range p.files {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		if err := p.processSingleFile(ctx, file); err != nil {
			return errors.Wrapf(err, "error processing file %s", file.manifestFile.GetName())
		}
	}
	return nil
}

func (p *restoreProcess) processSingleFile(ctx *restoreProcessContext, file *restoreFile) error {
	mfFile := file.manifestFile
	fileChunkReader := io.LimitReader(p.data, mfFile.GetEncodedSize())
	uncompressedReader, err := decodingReader(fileChunkReader, mfFile.GetEncoding())
	if err != nil {
		return err
	}

	fileCtx := ctx.forFile(mfFile.GetName())

	checksumReader := ioutils.NewCRC32ChecksumReader(
		uncompressedReader,
		crc32.IEEETable,
		mfFile.GetDecodedCrc32(),
	)

	// Wrap the checksumReader inside a NopCloser to ensure we are the ones to close it, as that does the final checksum
	// validation.
	if err := file.handlerFunc(fileCtx, io.NopCloser(checksumReader), mfFile.GetDecodedSize()); err != nil {
		return err
	}

	// Note that a LimitReader is always a NopCloser, so this doesn't affect the underlying stream.
	if err := checksumReader.Close(); err != nil {
		return err
	}

	// Ensure that all bytes were read.
	var buf [1]byte
	if n, err := fileChunkReader.Read(buf[:]); err != io.EOF || n != 0 {
		return errors.New("not all bytes in file chunk were read")
	}

	atomic.AddInt64(&p.filesProcessed, 1)

	return nil
}

func (p *restoreProcess) ProtoStatus() *v1.DBRestoreProcessStatus {
	status := &v1.DBRestoreProcessStatus{
		Metadata:       p.Metadata(),
		BytesRead:      atomic.LoadInt64(&p.bytesRead),
		FilesProcessed: atomic.LoadInt64(&p.filesProcessed),
	}

	if !p.started.Get() {
		status.State = v1.DBRestoreProcessStatus_NOT_STARTED
	} else if err, done := p.completionSig.Error(); !done {
		concurrency.WithLock(&p.mutex, func() {
			if p.reattachC != nil {
				status.State = v1.DBRestoreProcessStatus_PAUSED
				status.ResumeInfo = &v1.DBRestoreProcessStatus_ResumeInfo{
					Pos: p.reattachPos,
				}
			} else {
				status.State = v1.DBRestoreProcessStatus_IN_PROGRESS
				status.AttemptId = p.attemptID
			}
		})
	} else {
		status.State = v1.DBRestoreProcessStatus_COMPLETED
		if err != nil {
			status.Error = err.Error()
		}
	}

	return status
}
