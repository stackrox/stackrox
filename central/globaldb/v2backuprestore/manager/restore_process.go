package manager

import (
	"context"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/ioutils"
	"github.com/stackrox/rox/pkg/uuid"
)

// RestoreProcess provides a handle to ongoing database restore processes.
type RestoreProcess interface {
	Metadata() *v1.DBRestoreProcessMetadata
	Completion() concurrency.ErrorWaitable
	Cancel()
}

type restoreFile struct {
	manifestFile *v1.DBExportManifest_File
	handlerFunc  common.RestoreFileHandlerFunc
}

type restoreProcess struct {
	metadata *v1.DBRestoreProcessMetadata

	files []*restoreFile
	data  io.Reader

	started    concurrency.Flag
	cancelSig  concurrency.Signal
	stoppedSig concurrency.ErrorSignal
}

func newRestoreProcess(ctx context.Context, header *v1.DBRestoreRequestHeader, handlerFuncs []common.RestoreFileHandlerFunc, data io.Reader) (*restoreProcess, error) {
	mfFiles := header.GetManifest().GetFiles()
	if len(mfFiles) != len(handlerFuncs) {
		return nil, errorhelpers.PanicOnDevelopmentf("mismatch: %d handler functions provided for %d files in the manifest", len(handlerFuncs), len(mfFiles))
	}

	files := make([]*restoreFile, 0, len(mfFiles))
	for i, manifestFile := range mfFiles {
		files = append(files, &restoreFile{
			manifestFile: manifestFile,
			handlerFunc:  handlerFuncs[i],
		})
	}

	metadata := &v1.DBRestoreProcessMetadata{
		Id:        uuid.NewV4().String(),
		Header:    header,
		StartTime: types.TimestampNow(),
	}

	if identity := authn.IdentityFromContext(ctx); identity != nil {
		metadata.InitiatingUserName = identity.User().GetUsername()
	}

	return &restoreProcess{
		metadata: metadata,
		files:    files,
		data:     data,

		cancelSig:  concurrency.NewSignal(),
		stoppedSig: concurrency.NewErrorSignal(),
	}, nil
}

func (p *restoreProcess) Metadata() *v1.DBRestoreProcessMetadata {
	return p.metadata
}

func (p *restoreProcess) Launch(tempOutputDir, finalDir string) error {
	if p.started.TestAndSet(true) {
		return errors.New("restore process has already been started")
	}
	go p.run(context.Background(), tempOutputDir, finalDir)
	return nil
}

func (p *restoreProcess) cleanUp(outputDir string) {
	// RemoveAll will return a nil error if the directory does not exist.
	if err := os.RemoveAll(outputDir); err != nil {
		log.Warnf("Could not remove temporary restore output directory %s: %v", outputDir, err)
	}
}

func (p *restoreProcess) run(ctx context.Context, tempOutputDir, finalDir string) {
	defer p.cleanUp(tempOutputDir)

	defer p.cancelSig.Signal()
	// Make sure the process runs in a context that respects the stop signal.
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	concurrency.CancelContextOnSignal(subCtx, cancel, &p.cancelSig)

	p.stoppedSig.SignalWithError(p.doRun(subCtx, tempOutputDir, finalDir))
}

func (p *restoreProcess) doRun(ctx context.Context, tempOutputDir, finalDir string) error {
	if err := os.MkdirAll(tempOutputDir, 0700); err != nil {
		return errors.Wrapf(err, "could not create temporary output directory %s", tempOutputDir)
	}

	restoreCtx := newRestoreProcessContext(ctx, tempOutputDir)

	if err := p.processFiles(restoreCtx); err != nil {
		return err
	}

	if err := restoreCtx.waitForAsyncChecks(); err != nil {
		return err
	}

	if err := os.Rename(tempOutputDir, finalDir); err != nil {
		return errors.Wrapf(err, "restore process succeeded, but failed to atomically rename output directory %s", tempOutputDir)
	}

	return nil
}

func (p *restoreProcess) Cancel() {
	p.cancelSig.Signal()
}

func (p *restoreProcess) Completion() concurrency.ErrorWaitable {
	return &p.stoppedSig
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
	if err := file.handlerFunc(fileCtx, ioutil.NopCloser(checksumReader), mfFile.GetDecodedSize()); err != nil {
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
	return nil
}
