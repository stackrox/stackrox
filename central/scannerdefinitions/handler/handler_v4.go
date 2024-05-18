package handler

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
)

// v4OpenOpts are options to open most recent V4 definition files.
type v4OpenOpts struct {
	// urlPath specifies the update URL path when setting up online updaters.
	urlPath string
	// mappingFile specifies the mapping file to open for mappings updaters.
	mappingFile string
	// vulnVersion specifies the version of the vulnerability bundle for
	// vulnerability updaters.
	vulnVersion string
	// vulnBundle specifies the vulnerability bundle name for vulnerability updaters.
	vulnBundle string
}

func (h *httpHandler) getV4(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Query().Get(`file`)
	v := r.URL.Query().Get(`version`)
	ctx := r.Context()

	var uType updaterType
	var opts v4OpenOpts

	switch {
	case fileName != "" && v == "":
		// If only file is requested, then this is request for Scanner v4 mapping file.
		v4FileName, exists := v4FileMapping[fileName]
		if !exists {
			writeErrorNotFound(w)
			return
		}
		uType = mappingUpdaterType
		opts.mappingFile = v4FileName
	case fileName == "" && v != "":
		// If only version is provided, this is for Scanner V4 vuln file
		if version.GetVersionKind(v) == version.NightlyKind {
			// get dev for nightly at this moment
			v = "dev"
		}
		uType = vulnerabilityUpdaterType
		bundle := "vulns.json.zst"
		opts.urlPath = path.Join(v, bundle)
		opts.vulnVersion = v
		opts.vulnBundle = bundle
	default:
		writeErrorBadRequest(w)
		return
	}

	f, err := h.openMostRecentV4File(ctx, uType, opts)
	if err != nil {
		// Just using generic file name for this log...
		writeErrorForFile(w, err, "file")
		return
	}

	if uType == vulnerabilityUpdaterType {
		// http.ServeContent does not automatically detect zst files.
		w.Header().Set("Content-Type", "application/zstd")
	}

	defer utils.IgnoreError(f.Close)
	serveContent(w, r, f.Name(), f.modTime, f)
}

func (h *httpHandler) openMostRecentV4File(ctx context.Context, t updaterType, opts v4OpenOpts) (file *vulDefFile, err error) {
	log.Debugf("Fetching scanner V4 (online: %t): type %s: options: %#v", h.online, t, opts)

	file, err = h.openMostRecentV4OfflineFile(ctx, t, opts)
	if !h.online {
		return file, err
	}
	if err != nil {
		log.Debugf("Failed to access offline file (ignore the message if no "+
			"offline bundle has been uploaded): %v", err)
	}

	offlineFile := file

	defer func() {
		if offlineFile != nil {
			_ = offlineFile.Close()
		}
	}()

	file, err = h.openMostRecentV4OnlineFile(ctx, t, opts)
	if err != nil {
		return nil, err
	}

	// If the offline files are newer, return them instead.
	if offlineFile != nil && offlineFile.modTime.After(file.modTime) {
		_ = file.Close()
		// Set nil to protect the deferred close.
		file, offlineFile = offlineFile, nil
	}

	return file, err
}

// openMostRecentV4OfflineFile gets desired offline file from compressed bundle: offlineScannerV4DefinitionBlobName
func (h *httpHandler) openMostRecentV4OfflineFile(ctx context.Context, t updaterType, opts v4OpenOpts) (*vulDefFile, error) {
	log.Debugf("Getting v4 offline data for updater: type %s: options: %#v", t, opts)
	openedFile, err := h.openOfflineBlob(ctx, offlineScannerV4DefinitionBlobName)
	if err != nil {
		return nil, err
	}
	if openedFile == nil {
		log.Warnf("Blob %s does not exist", offlineScannerV4DefinitionBlobName)
		return nil, errors.New("No valid scanner V4 file in offline mode")
	}

	var offlineFile *vulDefFile
	defer utils.IgnoreError(openedFile.Close)
	switch t {
	case mappingUpdaterType:
		// search mapping file
		fileName := filepath.Base(opts.mappingFile)
		targetFile, cleanUp, err := openFromArchive(openedFile.Name(), fileName)
		if err != nil {
			return nil, err
		}
		defer cleanUp()
		offlineFile = &vulDefFile{File: targetFile, modTime: openedFile.modTime}
	case vulnerabilityUpdaterType:
		// check version information in manifest
		mf, cleanUp, err := openFromArchive(openedFile.Name(), "manifest.json")
		if err != nil {
			return nil, err
		}
		defer cleanUp()
		offlineV, err := getOfflineFileVersion(mf)
		if err != nil {
			return nil, err
		}
		defer utils.IgnoreError(mf.Close)

		if offlineV != minorVersionPattern.FindString(opts.vulnVersion) && (opts.vulnVersion != "dev" || buildinfo.ReleaseBuild) {
			msg := fmt.Sprintf("failed to get offline vuln file, uploaded file is version: %s and requested file version is: %s", offlineV, opts.vulnVersion)
			log.Errorf(msg)
			return nil, errors.New(msg)
		}

		vulns, cleanUp, err := openFromArchive(openedFile.Name(), opts.vulnBundle)
		if err != nil {
			return nil, err
		}
		defer cleanUp()
		offlineFile = &vulDefFile{File: vulns, modTime: openedFile.modTime}
	default:
		return nil, fmt.Errorf("unknown Scanner V4 updater type: %s", t)
	}

	return offlineFile, nil
}

func getOfflineFileVersion(mf *os.File) (string, error) {
	var m manifest
	err := json.NewDecoder(mf).Decode(&m)
	if err != nil {
		return "", err
	}
	return m.Version, nil
}

// openMostRecentV4OnlineFile gets desired "online" file, which is pulled and managed by the updater.
func (h *httpHandler) openMostRecentV4OnlineFile(_ context.Context, t updaterType, opts v4OpenOpts) (*vulDefFile, error) {
	u := h.getUpdater(t, opts.urlPath)
	// Ensure the updater is running.
	u.Start()
	openedFile, onlineTime, err := u.file.Open()
	if err != nil {
		return nil, err
	}
	if openedFile == nil {
		return nil, fmt.Errorf("scanner V4 %s file %s not found", t, opts.urlPath)
	}
	log.Debugf("Compressed data file is available: %s", openedFile.Name())
	switch t {
	case mappingUpdaterType:
		targetFile, cleanUp, err := openFromArchive(openedFile.Name(), opts.mappingFile)
		if err != nil {
			return nil, err
		}
		defer cleanUp()
		return &vulDefFile{File: targetFile, modTime: onlineTime}, nil
	case vulnerabilityUpdaterType:
		return &vulDefFile{File: openedFile, modTime: onlineTime}, nil
	default:
		return nil, fmt.Errorf("unknown Scanner V4 updater type: %s", t)
	}
}

func validateV4DefsVersion(zipPath string) error {
	zipR, err := zip.OpenReader(zipPath)
	if err != nil {
		return errors.Wrap(err, "couldn't open file as zip")
	}
	defer utils.IgnoreError(zipR.Close)

	for _, zipF := range zipR.File {
		if strings.HasPrefix(zipF.Name, scannerV4DefsPrefix) {
			defs, _, err := openFromArchive(zipPath, zipF.Name)
			if err != nil {
				return errors.Wrap(err, "couldn't open v4 offline defs manifest.json")
			}
			utils.IgnoreError(defs.Close)
			mf, removeDefs, err := openFromArchive(defs.Name(), "manifest.json")
			if err != nil {
				return errors.Wrap(err, "couldn't open v4 offline defs manifest.json")
			}
			offlineV, err := getOfflineFileVersion(mf)
			utils.IgnoreError(mf.Close)
			removeDefs()
			if err != nil {
				return errors.Wrap(err, "couldn't get v4 offline defs version")
			}
			v := minorVersionPattern.FindString(version.GetMainVersion())
			if offlineV != "dev" && offlineV != v {
				msg := fmt.Sprintf("failed to upload offline file bundle, uploaded file is version: %s and system version is: %s; "+
					"please upload an offline bundle version: %s, consider using command roxctl scanner download-db", offlineV, version.GetMainVersion(), v)
				log.Errorf(msg)
				return errors.New(msg)
			}
		}
	}
	return nil
}
