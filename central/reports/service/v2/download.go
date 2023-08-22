package v2

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/central/reports/common"
	snapshotDataStore "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/zip"
	"google.golang.org/grpc/codes"
)

var (
	log = logging.LoggerForModule()
)

func parseJobID(r *http.Request) (id string, err error) {
	err = r.ParseForm()
	if err != nil {
		return
	}
	if id = r.Form.Get("id"); id == "" {
		err = errors.New("empty report job id")
	}
	return
}

// NewDownloadHandler is an HTTP handler for downloading reports
func NewDownloadHandler() http.HandlerFunc {
	snapshotStore := snapshotDataStore.Singleton()
	blobStore := blobDS.Singleton()
	handler := &downloadHandler{snapshotStore: snapshotStore, blobStore: blobStore}
	return handler.handle
}

type downloadHandler struct {
	snapshotStore snapshotDataStore.DataStore
	blobStore     blobDS.Datastore
}

func (h *downloadHandler) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteErrorf(w, http.StatusMethodNotAllowed, "Only GET requests are allowed")
		return
	}

	id, err := parseJobID(r)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, err)
		return
	}

	ctx := r.Context()
	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		httputil.WriteGRPCStyleError(w, codes.PermissionDenied, errors.New("Could not determine user identity from provided context"))
		return
	}

	rep, found, err := h.snapshotStore.Get(ctx, id)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrapf(err, "Error finding report snapshot with job ID %q.", id))
		return
	}

	if !found {
		httputil.WriteGRPCStyleError(w, codes.NotFound, errors.Errorf("Error finding report snapshot with job ID '%q'.", id))
		return
	}

	if slimUser.GetId() != rep.GetRequester().GetId() {
		httputil.WriteGRPCStyleError(w, codes.PermissionDenied,
			errors.Errorf("Report cannot be downloaded by a user %q who did not request the report.", slimUser.GetId()))
		return
	}

	status := rep.GetReportStatus()
	if status.GetReportNotificationMethod() != storage.ReportStatus_DOWNLOAD {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument,
			errors.Errorf("Report job id %q did not generate a downloadable report and hence report cannot be downloaded.", id))
		return
	}

	switch status.GetRunState() {
	case storage.ReportStatus_FAILURE:
		httputil.WriteGRPCStyleError(w, codes.FailedPrecondition,
			errors.Errorf("Report job %q has failed and hence no report to download", id))
		return
	case storage.ReportStatus_PREPARING, storage.ReportStatus_WAITING:
		httputil.WriteGRPCStyleError(w, codes.Unavailable,
			errors.Errorf("Report job %q is not ready for download", id))
		return
	}

	// Fetch data
	buf := bytes.NewBuffer(nil)
	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)),
	)

	_, exists, err := h.blobStore.Get(ctx, common.GetReportBlobPath(rep.GetReportConfigurationId(), id), buf)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.New("Failed to fetch report data"))
		return
	}

	if !exists {
		// If the blob does not exist, report error.
		httputil.WriteGRPCStyleError(w, codes.NotFound,
			errors.Errorf("Report is not available to download for job %q", id))
		return
	}

	// Tell the browser this is a download.
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="report-%s.zip"`, zip.GetSafeFilename(id)))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Length", fmt.Sprint(buf.Len()))
	_, err = w.Write(buf.Bytes())

	writeSnapshotCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)),
	)
	if err == nil && status.GetRunState() == storage.ReportStatus_GENERATED {
		rep.ReportStatus.RunState = storage.ReportStatus_DELIVERED
		err = h.snapshotStore.UpdateReportSnapshot(writeSnapshotCtx, rep)
		if err != nil {
			log.Error("Error setting report state to DELIVERED")
		}
	}
}
