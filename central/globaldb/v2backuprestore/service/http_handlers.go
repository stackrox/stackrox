package service

import (
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gogo/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/binenc"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/httputil"
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
		return status.Errorf(codes.InvalidArgument, "invalid header length %q: %v", headerLenStr, err)
	}

	id := queryValues.Get("id")
	if id == "" {
		id = uuid.NewV4().String()
	} else if _, err := uuid.FromString(id); err != nil {
		return status.Errorf(codes.InvalidArgument, "ID must be unset or a valid UUID (got: %q)", id)
	}

	headerBytes := make([]byte, headerLen)
	if _, err := io.ReadFull(req.Body, headerBytes); err != nil {
		return status.Errorf(codes.InvalidArgument, "could not read request header (%d bytes): %v", headerLen, err)
	}

	var header v1.DBRestoreRequestHeader
	if err := proto.Unmarshal(headerBytes, &header); err != nil {
		return status.Errorf(codes.InvalidArgument, "could not parse restore request header: %v", err)
	}

	attemptDone, err := s.mgr.LaunchRestoreProcess(req.Context(), id, &header, ioutil.NopCloser(req.Body))
	if err != nil {
		return status.Errorf(codes.Internal, "Could not create a restore process: %v", err)
	}

	restoreErr, wasDone := concurrency.WaitForErrorUntil(attemptDone, req.Context())
	if !wasDone {
		return status.Errorf(codes.Canceled, "context canceled before restore could complete: %v", req.Context().Err())
	}
	if restoreErr != nil {
		return status.Errorf(codes.Internal, "database restore failed: %v", err)
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
		return status.Errorf(codes.InvalidArgument, "need to specify a restore process ID for resuming")
	}

	attemptID := queryValues.Get("attemptId")
	if _, err := uuid.FromString(attemptID); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid attempt ID %q: %v", attemptID, err)
	}

	posStr := queryValues.Get("pos")
	pos, err := strconv.ParseInt(posStr, 10, 64)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid position specification %q: %v", posStr, err)
	}

	crc32Str := queryValues.Get("crc32")
	crc32Val, err := strconv.ParseUint(crc32Str, 16, 32)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid CRC32 value %q: %v", crc32Str, err)
	}

	crc32 := uint32(crc32Val)

	activeProcess := s.mgr.GetActiveRestoreProcess()
	if activeProcess.Metadata().GetId() != processID {
		return status.Errorf(codes.InvalidArgument, "specified process ID %s does not match ID of currently active restore process", processID)
	}

	attemptDone, err := activeProcess.Resume(req.Context(), attemptID, req.Body, pos, binenc.BigEndian.EncodeUint32(crc32))
	if err != nil {
		return status.Errorf(codes.Internal, "could not resume restore process %s: %v", processID, err)
	}

	attemptErr, wasDone := concurrency.WaitForErrorUntil(attemptDone, req.Context())
	if !wasDone {
		return status.Errorf(codes.Canceled, "context canceled before restore could complete: %v", req.Context().Err())
	}
	if attemptErr != nil {
		return status.Errorf(codes.Internal, "database restore failed: %v", err)
	}

	return nil
}
