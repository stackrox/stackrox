package manager

import (
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gogo/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/httputil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (m *manager) RestoreHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		err := m.handleRestoreV2(req)
		if err != nil {
			httputil.WriteError(w, err)
			return
		}
	})
}

func (m *manager) handleRestoreV2(req *http.Request) error {
	if req.Method != http.MethodPost {
		return httputil.Errorf(http.StatusMethodNotAllowed, "Only POST requests are allowed")
	}

	queryValues := req.URL.Query()
	headerLenStr := queryValues.Get("headerLength")
	headerLen, err := strconv.Atoi(headerLenStr)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid header length %q: %v", headerLenStr, err)
	}

	headerBytes := make([]byte, headerLen)
	if _, err := io.ReadFull(req.Body, headerBytes); err != nil {
		return status.Errorf(codes.InvalidArgument, "could not read request header (%d bytes): %v", headerLen, err)
	}

	var header v1.DBRestoreRequestHeader
	if err := proto.Unmarshal(headerBytes, &header); err != nil {
		return status.Errorf(codes.InvalidArgument, "could not parse restore request header: %v", err)
	}

	process, err := m.LaunchRestoreProcess(req.Context(), &header, ioutil.NopCloser(req.Body))
	if err != nil {
		return status.Errorf(codes.Internal, "Could not create a restore process: %v", err)
	}

	restoreErr, wasDone := concurrency.WaitForErrorUntil(process.Completion(), req.Context())
	if !wasDone {
		return status.Errorf(codes.Canceled, "context canceled before restore could complete: %v", req.Context().Err())
	}
	if restoreErr != nil {
		return status.Errorf(codes.Internal, "database restore failed: %v", err)
	}

	return nil
}
