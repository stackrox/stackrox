package handler

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/stackrox/rox/pkg/utils"
)

func (h *httpHandler) getV2(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get(`uuid`)
	fileName := r.URL.Query().Get(`file`)

	if uuid == "" {
		writeErrorBadRequest(w)
		return
	}

	// Open the most recent definitions file for the provided uuid.
	f, err := h.openMostRecentDefinitions(r.Context(), uuid)
	if err != nil {
		writeErrorForFile(w, err, uuid)
		return
	}

	// It is possible no offline Scanner definitions were uploaded, Central cannot
	// reach the definitions object, or there are no definitions for the given
	// uuid; in any of those cases, f will be nil.
	if f == nil {
		writeErrorNotFound(w)
		return
	}

	defer utils.IgnoreError(f.Close)

	// No specific file was requested, so return all definitions.
	if fileName == "" {
		serveContent(w, r, f.Name(), f.modTime, f)
		return
	}

	// A specific file was requested, so extract from definitions bundle to a
	// temporary file and serve that instead.
	namedFile, cleanUp, err := openFromArchive(f.Name(), fileName)
	if err != nil {
		writeErrorForFile(w, err, fileName)
		return
	}
	defer cleanUp()
	defer utils.IgnoreError(namedFile.Close)
	serveContent(w, r, namedFile.Name(), f.modTime, namedFile)
}

// openMostRecentDefinitions opens the latest Scanner Definitions based on
// modification time. It's either the one selected by `uuid` if present and
// online, otherwise fallback to the manually uploaded definitions. The file
// object can be `nil` if the definitions file does not exist, rather than
// returning an error.
func (h *httpHandler) openMostRecentDefinitions(ctx context.Context, uuid string) (file *vulDefFile, err error) {
	// If in offline mode or uuid is not provided, default to the offline file.
	if !h.online || uuid == "" {
		file, err = h.openOfflineBlob(ctx, offlineScannerDefinitionBlobName)
		if err == nil && file == nil {
			log.Warnf("Blob %s does not exist", offlineScannerDefinitionBlobName)
		}
		return
	}

	// Start the updater, can be called multiple times for the same uuid, but will
	// only start the updater once. The Start() call blocks if the definitions were
	// not downloaded yet.
	u := h.getUpdater(v2UpdaterType, uuid)

	toClose := func(f *vulDefFile) {
		if file != f && f != nil {
			utils.IgnoreError(f.Close)
		}
	}

	// Open both the "online" and "offline", and save their modification times.
	var onlineFile *vulDefFile
	onlineOSFile, onlineTime, err := h.startUpdaterAndOpenFile(u)
	if err != nil {
		return
	}
	if onlineOSFile != nil {
		onlineFile = &vulDefFile{File: onlineOSFile, modTime: onlineTime}
	}

	defer toClose(onlineFile)
	offlineFile, err := h.openOfflineBlob(ctx, offlineScannerDefinitionBlobName)
	if err != nil {
		return
	}
	defer toClose(offlineFile)

	// Return the most recent file, notice that if both don't exist, nil is returned
	// since modification time will be zero.
	file = onlineFile
	if offlineFile != nil && offlineFile.modTime.After(onlineTime) {
		file = offlineFile
	}
	return
}

func (h *httpHandler) startUpdaterAndOpenFile(u *requestedUpdater) (*os.File, time.Time, error) {
	u.Start()
	osFile, modTime, err := u.file.Open()
	if err != nil {
		return nil, time.Time{}, err
	}
	if osFile == nil {
		return nil, time.Time{}, nil
	}
	return osFile, modTime, nil
}
