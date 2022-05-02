package service

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/binenc"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/ioutils"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *service) RestoreHandler() http.Handler {
	return httputil.WrapHandlerFunc(s.handleRestore)
}

func (s *service) ResumeRestoreHandler() http.Handler {
	return httputil.WrapHandlerFunc(s.handleResumeRestore)
}

func (s *service) handleRestore(req *http.Request) error {
	if req.Method != http.MethodPost {
		return httputil.Errorf(http.StatusMethodNotAllowed, "Only POST requests are allowed")
	}

	queryValues := req.URL.Query()
	headerLenStr := queryValues.Get("headerLength")
	headerLen, err := strconv.Atoi(headerLenStr)
	if err != nil {
		return errors.Wrapf(errox.InvalidArgs, "invalid header length %q: %v", headerLenStr, err)
	}

	id := queryValues.Get("id")
	if id == "" {
		id = uuid.NewV4().String()
	} else if _, err := uuid.FromString(id); err != nil {
		return errors.Wrapf(errox.InvalidArgs, "ID must be unset or a valid UUID (got: %q)", id)
	}

	headerBytes := make([]byte, headerLen)
	if _, err := io.ReadFull(req.Body, headerBytes); err != nil {
		return errors.Wrapf(errox.InvalidArgs, "could not read request header (%d bytes): %v", headerLen, err)
	}

	var header v1.DBRestoreRequestHeader
	if err := proto.Unmarshal(headerBytes, &header); err != nil {
		return errors.Wrapf(errox.InvalidArgs, "could not parse restore request header: %v", err)
	}

	// Make sure we perform a clean cut when reading from the stream. Returning from a handler while a concurrent call
	// to read is ongoing might result in corrupted data being reported (this could happen if we launch a restore
	// operation with the `--interrupt` flag while another one is active).
	body, interrupt := ioutils.NewInterruptibleReader(req.Body)
	defer interrupt()

	// Make sure calls to `Read` will return an error once the handler returns. We have observed cases where a
	// connection interruption caused a call to `Read` to hang indefinitely, so even though the handler was exited, the
	// process could neither be interrupted nor canceled nor resumed, since the reader would never detach.
	readCtx, cancel := context.WithCancel(req.Context())
	defer cancel()
	body = ioutils.NewContextBoundReader(readCtx, body)

	attemptDone, err := s.mgr.LaunchRestoreProcess(req.Context(), id, &header, io.NopCloser(body))
	if err != nil {
		return errors.Errorf("Could not create a restore process: %v", err)
	}

	restoreErr, wasDone := concurrency.WaitForErrorUntil(attemptDone, req.Context())
	if !wasDone {
		return status.Errorf(codes.Canceled, "context canceled before restore could complete: %v", req.Context().Err())
	}
	if restoreErr != nil {
		return errors.Errorf("database restore failed: %v", restoreErr)
	}

	return nil
}

func (s *service) handleResumeRestore(req *http.Request) error {
	if req.Method != http.MethodPost {
		return httputil.Errorf(http.StatusMethodNotAllowed, "Only POST requests are allowed")
	}

	queryValues := req.URL.Query()
	processID := queryValues.Get("id")
	if processID == "" {
		return errors.Wrap(errox.InvalidArgs, "need to specify a restore process ID for resuming")
	}

	attemptID := queryValues.Get("attemptId")
	if _, err := uuid.FromString(attemptID); err != nil {
		return errors.Wrapf(errox.InvalidArgs, "invalid attempt ID %q: %v", attemptID, err)
	}

	posStr := queryValues.Get("pos")
	pos, err := strconv.ParseInt(posStr, 10, 64)
	if err != nil {
		return errors.Wrapf(errox.InvalidArgs, "invalid position specification %q: %v", posStr, err)
	}

	crc32Str := queryValues.Get("crc32")
	crc32Val, err := strconv.ParseUint(crc32Str, 16, 32)
	if err != nil {
		return errors.Wrapf(errox.InvalidArgs, "invalid CRC32 value %q: %v", crc32Str, err)
	}

	crc32 := uint32(crc32Val)

	activeProcess := s.mgr.GetActiveRestoreProcess()
	if activeProcess.Metadata().GetId() != processID {
		return errors.Wrapf(errox.InvalidArgs, "specified process ID %s does not match ID of currently active restore process", processID)
	}

	// Make sure we perform a clean cut when reading from the stream. Returning from a handler while a concurrent call
	// to read is ongoing might result in corrupted data being reported (this could happen if we launch a restore
	// operation with the `--interrupt` flag while another one is active).
	body, interrupt := ioutils.NewInterruptibleReader(req.Body)
	defer interrupt()

	// Make sure calls to `Read` will return an error once the handler returns. We have observed cases where a
	// connection interruption caused a call to `Read` to hang indefinitely, so even though the handler was exited, the
	// process could neither be interrupted nor canceled nor resumed, since the reader would never detach.
	readCtx, cancel := context.WithCancel(req.Context())
	defer cancel()
	body = ioutils.NewContextBoundReader(readCtx, body)

	attemptDone, err := activeProcess.Resume(req.Context(), attemptID, io.NopCloser(body), pos, binenc.BigEndian.EncodeUint32(crc32))
	if err != nil {
		return errors.Errorf("could not resume restore process %s: %v", processID, err)
	}

	attemptErr, wasDone := concurrency.WaitForErrorUntil(attemptDone, req.Context())
	if !wasDone {
		return status.Errorf(codes.Canceled, "context canceled before restore could complete: %v", req.Context().Err())
	}
	if attemptErr != nil {
		return errors.Errorf("database restore failed: %v", attemptErr)
	}

	return nil
}
