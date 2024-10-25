package service

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	"github.com/stackrox/rox/central/reports/common"
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

func NewDownloadHandler() http.HandlerFunc {
	snapshotStore := snapshotDS.Singleton()
	blobStore := blobDS.Singleton()
	handler := &downloadHandler{
		snapshotDataStore: snapshotStore,
		blobStore:         blobStore,
	}
	return handler.handle
}

type downloadHandler struct {
	snapshotDataStore snapshotDS.DataStore
	blobStore         blobDS.Datastore
}

func (h *downloadHandler) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteErrorf(w, http.StatusMethodNotAllowed, "Only GET requests are allowed")
		return
	}

	id, err := parseReportID(r)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.Wrap(err, "error parsing report ID"))
		return
	}

	ctx := r.Context()
	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		httputil.WriteGRPCStyleError(w, codes.PermissionDenied, errors.New("Could not determine user identity from provided context"))
		return
	}

	snapshot, found, err := h.snapshotDataStore.GetSnapshot(ctx, id)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.NotFound, errors.Errorf("Error retrieving snapshot with the given id %s", id))
		return
	}
	if !found {
		httputil.WriteGRPCStyleError(w, codes.NotFound, errors.Errorf("Unable to find the snapshot with the given id %s", id))
		return
	}

	if slimUser.GetId() != snapshot.GetUser().GetId() {
		httputil.WriteGRPCStyleError(w, codes.PermissionDenied, errors.Errorf("Report %s cannot be downloaded by the user %s", id, slimUser.GetId()))
		return
	}

	status := snapshot.GetReportStatus()
	if status.GetReportNotificationMethod() != storage.ComplianceOperatorReportStatus_DOWNLOAD {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.Errorf("This report %s cannot be delivered using the download method", id))
		return
	}

	switch status.GetRunState() {
	case storage.ComplianceOperatorReportStatus_FAILURE:
		httputil.WriteGRPCStyleError(w, codes.FailedPrecondition, errors.Errorf("Report %s failed: %s", id, status.GetErrorMsg()))
		return
	case storage.ComplianceOperatorReportStatus_WAITING, storage.ComplianceOperatorReportStatus_PREPARING:
		httputil.WriteGRPCStyleError(w, codes.Unavailable, errors.Errorf("Report %s is not yet ready for download", id))
		return
	}

	buf := bytes.NewBuffer(nil)
	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)

	_, exists, err := h.blobStore.Get(ctx, common.GetComplianceReportBlobPath(snapshot.GetScanConfigurationId(), id), buf)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.New("Failed to fetch report data"))
		return
	}
	if !exists {
		httputil.WriteGRPCStyleError(w, codes.NotFound, errors.Errorf("The download of report %s is unavailable", id))
		return
	}

	// Tell the browser this is a download.
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="report-%s.zip"`, zip.GetSafeFilename(id)))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Length", fmt.Sprint(buf.Len()))
	_, err = w.Write(buf.Bytes())
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.New("Unable to attach the download to the response"))
		return
	}

	writeSnapshotCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)),
	)
	if status.GetRunState() == storage.ComplianceOperatorReportStatus_GENERATED {
		snapshot.GetReportStatus().RunState = storage.ComplianceOperatorReportStatus_DELIVERED
		err = h.snapshotDataStore.UpsertSnapshot(writeSnapshotCtx, snapshot)
		if err != nil {
			log.Error("Error setting report state to DELIVERED")
		}
	}
}

func parseReportID(r *http.Request) (string, error) {
	err := r.ParseForm()
	if err != nil {
		return "", errors.Wrap(err, "unable to parse the request form")
	}
	var id string
	if id = r.Form.Get("id"); id == "" {
		return "", errors.New("empty report id")
	}
	return id, nil
}
