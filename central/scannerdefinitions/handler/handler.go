package handler

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	blob "github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/central/blob/snapshot"
	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"google.golang.org/grpc/codes"
)

const (
	definitionsBaseDir = "scannerdefinitions"

	// scannerDefsSubZipName represents the offline zip bundle for CVEs for Scanner.
	scannerDefsSubZipName   = "scanner-defs.zip"
	scannerV4DefsSubZipName = "scanner-v4-defs.zip"
	// scannerV4DefsPrefix helps to search the v4 offline zip bundle for CVEs
	scannerV4DefsPrefix = "scanner-v4-defs"

	// offlineScannerDefinitionBlobName represents the blob name of offline/fallback zip bundle for CVEs for Scanner.
	offlineScannerDefinitionBlobName = "/offline/scanner/" + scannerDefsSubZipName

	// offlineScannerV4DefinitionBlobName represents the blob name of offline/fallback zip bundle for CVEs for Scanner V4.
	offlineScannerV4DefinitionBlobName = "/offline/scanner/v4/" + scannerV4DefsSubZipName

	scannerUpdateDomain    = "https://definitions.stackrox.io"
	scannerUpdateURLSuffix = "diff.zip"

	defaultCleanupInterval = 4 * time.Hour
	defaultCleanupAge      = 1 * time.Hour

	v4VulnSubDir      = "v4/vulnerability-bundles"
	v4MappingSubDir   = "v4/redhat-repository-mappings"
	mappingFile       = "mapping.zip"
	v4VulnFile        = "vulns.json.zst"
	mappingUpdaterKey = "mapping"
)

//go:generate stringer -type=updaterType
type updaterType int

const (
	mappingUpdaterType updaterType = iota
	vulnerabilityUpdaterType
	v2UpdaterType
)

var (
	client = &http.Client{
		Transport: proxy.RoundTripper(),
		Timeout:   5 * time.Minute,
	}

	log = logging.LoggerForModule()

	v4FileMapping = map[string]string{
		"name2repos": "repomapping/container-name-repos-map.json",
		"repo2cpe":   "repomapping/repository-to-cpe.json",
	}
	minorVersionPattern = regexp.MustCompile(`^\d+\.\d+`)
)

type requestedUpdater struct {
	*updater
	lastRequestedTime time.Time
}

type manifest struct {
	Version string `json:"version"`
}

// httpHandler handles HTTP GET and POST requests for vulnerability data.
type httpHandler struct {
	online        bool
	interval      time.Duration
	lock          sync.Mutex
	updaters      map[string]*requestedUpdater
	onlineVulnDir string
	blobStore     blob.Datastore
}

// New creates a new http.Handler to handle vulnerability data.
func New(blobStore blob.Datastore, opts handlerOpts) http.Handler {
	h := &httpHandler{
		online:    !env.OfflineModeEnv.BooleanSetting(),
		interval:  env.ScannerVulnUpdateInterval.DurationSetting(),
		blobStore: blobStore,
	}

	if h.online {
		h.initializeUpdaters(opts.cleanupInterval, opts.cleanupAge)
	} else {
		log.Info("In offline mode: scanner definitions will not be updated automatically")
	}

	return h
}

func (h *httpHandler) initializeUpdaters(cleanupInterval, cleanupAge *time.Duration) {
	var err error
	h.onlineVulnDir, err = os.MkdirTemp("", definitionsBaseDir)
	utils.CrashOnError(err) // Fundamental problem if we cannot create a temp directory.

	h.updaters = make(map[string]*requestedUpdater)
	go h.runCleanupUpdaters(cleanupInterval, cleanupAge)
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.get(w, r)
	case http.MethodPost:
		h.post(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *httpHandler) get(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get(`uuid`)
	fileName := r.URL.Query().Get(`file`)
	v := r.URL.Query().Get(`version`)
	// If only file is requested, then this is request for Scanner v4 mapping file.
	if fileName != "" && uuid == "" && v == "" {
		if v4FileName, exists := v4FileMapping[fileName]; exists {
			h.getV4(r.Context(), w, r, mappingUpdaterType, v4FileName)
			return
		}
		writeErrorNotFound(w)
		return
	}
	// If only version is provided, this is for Scanner V4 vuln file
	if v != "" && uuid == "" && fileName == "" {
		h.getV4(r.Context(), w, r, vulnerabilityUpdaterType, v)
		return
	}

	// At this point, we assume the request is from Scanner v2.
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
	namedFile, err := openFromArchive(f.Name(), fileName)
	if err != nil {
		writeErrorForFile(w, err, fileName)
		return
	}
	defer utils.IgnoreError(namedFile.Close)
	serveContent(w, r, namedFile.Name(), f.modTime, namedFile)
}

func writeErrorNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("No scanner definitions found"))
}

func writeErrorBadRequest(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte("at least one of file or uuid must be specified"))
}

func writeErrorForFile(w http.ResponseWriter, err error, path string) {
	if errorhelpers.IsAny(err, fs.ErrNotExist, snapshot.ErrBlobNotExist) {
		writeErrorNotFound(w)
		return
	}

	httputil.WriteGRPCStyleErrorf(w, codes.Internal, "could not read vulnerability definition %s: %v", filepath.Base(path), err)
}

func serveContent(w http.ResponseWriter, r *http.Request, name string, modTime time.Time, content io.ReadSeeker) {
	log.Debugf("Serving vulnerability definitions from %s", filepath.Base(name))
	http.ServeContent(w, r, name, modTime, content)
}

// getUpdater gets or creates the updater for the scanner definitions
// identified by the given key/uuid.
// If the updater is created here, it is no started here, as it is a blocking operation.
func (h *httpHandler) getUpdater(t updaterType, key string) *requestedUpdater {
	h.lock.Lock()
	defer h.lock.Unlock()

	updater, exists := h.updaters[key]
	if !exists {
		filePath := filepath.Join(h.onlineVulnDir, key)

		var urlStr string
		switch t {
		case mappingUpdaterType:
			urlStr, _ = url.JoinPath(scannerUpdateDomain, v4MappingSubDir, mappingFile)
			filePath += ".zip"
		case vulnerabilityUpdaterType:
			urlStr, _ = url.JoinPath(scannerUpdateDomain, v4VulnSubDir, key, v4VulnFile)
			filePath += ".json.zst"
		default: // uuid
			urlStr, _ = url.JoinPath(scannerUpdateDomain, key, scannerUpdateURLSuffix)
			filePath += ".zip"
		}

		h.updaters[key] = &requestedUpdater{
			updater: newUpdater(file.New(filePath), client, urlStr, h.interval),
		}
		updater = h.updaters[key]
	}

	updater.lastRequestedTime = time.Now()
	return updater
}

func (h *httpHandler) handleScannerDefsFile(ctx context.Context, zipF *zip.File, blobName string) error {
	r, err := zipF.Open()
	if err != nil {
		return errors.Wrap(err, "opening ZIP reader")
	}
	defer utils.IgnoreError(r.Close)

	// POST requests only update the offline feed.
	b := &storage.Blob{
		Name:         blobName,
		LastUpdated:  timestamp.TimestampNow(),
		ModifiedTime: timestamp.TimestampNow(),
		Length:       zipF.FileInfo().Size(),
	}

	if err := h.blobStore.Upsert(sac.WithAllAccess(ctx), b, r); err != nil {
		return errors.Wrap(err, "writing scanner definitions")
	}

	return nil
}

func (h *httpHandler) handleZipContentsFromVulnDump(ctx context.Context, zipPath string) error {
	zipR, err := zip.OpenReader(zipPath)
	if err != nil {
		return errors.Wrap(err, "couldn't open file as zip")
	}
	defer utils.IgnoreError(zipR.Close)
	var count int
	// It is expected a ZIP file be uploaded with both Scanner V2 and V4 vulnerability definitions.
	// scanner-defs.zip contains data required by Scanner V2.
	// scanner-v4-defs-*.zip contains data required by Scanner v4.
	// In the future, we may decide to support other files (like we have in the past), which is why we
	// expect this ZIP of a single ZIP.
	for _, zipF := range zipR.File {
		if zipF.Name == scannerDefsSubZipName {
			if err := h.handleScannerDefsFile(ctx, zipF, offlineScannerDefinitionBlobName); err != nil {
				return errors.Wrap(err, "couldn't handle scanner-defs sub file")
			}
			count++
			continue
		}
		if strings.HasPrefix(zipF.Name, scannerV4DefsPrefix) {
			if err := h.handleScannerDefsFile(ctx, zipF, offlineScannerV4DefinitionBlobName); err != nil {
				return errors.Wrap(err, "couldn't handle scanner-v4-defs sub file")
			}
			log.Debugf("Successfully processed file: %s", zipF.Name)
			count++
		}
		// Ignore any other files which may be in the ZIP.
	}
	if count > 0 {
		return nil
	}
	return errors.New("scanner defs file not found in upload zip; wrong zip uploaded?")
}

func (h *httpHandler) post(w http.ResponseWriter, r *http.Request) {
	tempDir, err := os.MkdirTemp("", "scanner-definitions-handler")
	if err != nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "failed to create temp dir: %v", err)
		return
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			log.Warnf("Failed to remove temp dir for scanner defs: %v", err)
		}
	}()

	tempFile := filepath.Join(tempDir, "tempfile.zip")
	if err := fileutils.CopySrcToFile(tempFile, r.Body); err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrapf(err, "copying HTTP POST body to %s", tempFile))
		return
	}

	if err := h.handleZipContentsFromVulnDump(r.Context(), tempFile); err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, err)
		return
	}

	_, _ = w.Write([]byte("Successfully stored the offline vulnerability definitions"))
}

func (h *httpHandler) runCleanupUpdaters(cleanupInterval, cleanupAge *time.Duration) {
	interval := defaultCleanupInterval
	if cleanupInterval != nil {
		interval = *cleanupInterval
	}
	age := defaultCleanupAge
	if cleanupAge != nil {
		age = *cleanupAge
	}

	t := time.NewTicker(interval)
	for range t.C {
		h.cleanupUpdaters(age)
	}
}

func (h *httpHandler) cleanupUpdaters(cleanupAge time.Duration) {
	now := time.Now()

	h.lock.Lock()
	defer h.lock.Unlock()

	for id, updatingHandler := range h.updaters {
		if now.Sub(updatingHandler.lastRequestedTime) > cleanupAge {
			// Updater has not been requested for a long time.
			// Clean it up.
			updatingHandler.Stop()
			delete(h.updaters, id)
		}
	}
}

func (h *httpHandler) openOfflineBlob(ctx context.Context, blobName string) (*vulDefFile, error) {
	snap, err := snapshot.TakeBlobSnapshot(sac.WithAllAccess(ctx), h.blobStore, blobName)
	if err != nil {
		// If the blob does not exist, return no reader.
		if errors.Is(err, snapshot.ErrBlobNotExist) {
			return nil, nil
		}
		log.Warnf("Cannnot take a snapshot of Blob %q: %v", blobName, err)
		return nil, err
	}
	modTime := time.Time{}
	if t := pgutils.NilOrTime(snap.GetBlob().ModifiedTime); t != nil {
		modTime = *t
	}
	return &vulDefFile{snap.File, modTime, snap.Close}, nil
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

func (h *httpHandler) openMostRecentV4File(ctx context.Context, t updaterType, updaterKey, fileName string) (file *vulDefFile, err error) {
	if !h.online {
		return h.openMostRecentV4OfflineFile(ctx, t, updaterKey, fileName)
	}
	log.Debugf("Getting v4 data for updater key: %s", updaterKey)
	u := h.getUpdater(t, updaterKey)
	var onlineFile *vulDefFile
	// Ensure the updater is running.
	u.Start()
	openedFile, onlineTime, err := u.file.Open()
	if err != nil {
		return nil, err
	}
	if openedFile == nil {
		return nil, fmt.Errorf("Scanner V4 %s file %s not found", t, updaterKey)
	}
	log.Debugf("Compressed data file is available: %s", openedFile.Name())
	toClose := func(f *vulDefFile) {
		if file != f && f != nil {
			utils.IgnoreError(f.Close)
		}
	}
	switch t {
	case mappingUpdaterType:
		targetFile, err := openFromArchive(openedFile.Name(), fileName)
		if err != nil {
			return nil, err
		}
		onlineFile = &vulDefFile{File: targetFile, modTime: onlineTime}
	case vulnerabilityUpdaterType:
		onlineFile = &vulDefFile{File: openedFile, modTime: onlineTime}
	default:
		return nil, fmt.Errorf("unknown Scanner V4 updater type: %s", t)
	}
	defer toClose(onlineFile)
	file = onlineFile

	offlineFile, err := h.openMostRecentV4OfflineFile(ctx, t, updaterKey, fileName)
	if err != nil {
		log.Errorf("failed to access offline file: %v", err)
	}
	defer toClose(offlineFile)

	if offlineFile != nil && offlineFile.modTime.After(onlineTime) {
		file = offlineFile
	}
	return file, nil
}

// openMostRecentV4OfflineFile gets desired offline file from compressed bundle: offlineScannerV4DefinitionBlobName
func (h *httpHandler) openMostRecentV4OfflineFile(ctx context.Context, t updaterType, updaterKey, fileName string) (*vulDefFile, error) {
	log.Debugf("Getting v4 offline data for updater key: %s", updaterKey)
	openedFile, err := h.openOfflineBlob(ctx, offlineScannerV4DefinitionBlobName)
	if err == nil && openedFile == nil {
		log.Warnf("Blob %s does not exist", offlineScannerV4DefinitionBlobName)
		return nil, errors.New("No valid scanner V4 file in offline mode")
	}

	var offlineFile *vulDefFile
	defer utils.IgnoreError(openedFile.Close)
	switch t {
	case mappingUpdaterType:
		// search mapping file
		fileName = filepath.Base(fileName)
		targetFile, err := openFromArchive(openedFile.Name(), fileName)
		if err != nil {
			return nil, err
		}
		offlineFile = &vulDefFile{File: targetFile, modTime: openedFile.modTime}
	case vulnerabilityUpdaterType:
		// check version information in manifest
		mf, err := openFromArchive(openedFile.Name(), "manifest.json")
		if err != nil {
			return nil, err
		}

		offlineV, err := getOfflineFileVersion(mf)
		if err != nil {
			return nil, err
		}
		defer utils.IgnoreError(mf.Close)

		if (updaterKey != "dev" && offlineV != minorVersionPattern.FindString(updaterKey)) ||
			(updaterKey == "dev" && offlineV != "dev") {
			msg := fmt.Sprintf("failed to get offline vuln file, uploaded file is version: %s and requested file version is: %s", offlineV, updaterKey)
			log.Errorf(msg)
			return nil, errors.New(msg)
		}

		vulns, err := openFromArchive(openedFile.Name(), "vulns.json.zst")
		if err != nil {
			return nil, err
		}
		offlineFile = &vulDefFile{File: vulns, modTime: openedFile.modTime}
	default:
		return nil, fmt.Errorf("unknown Scanner V4 updater type: %s", t)
	}

	return offlineFile, nil
}

// openFromArchive returns a file object for a name within the definitions
// bundle. The file object has a file descriptor allocated on the filesystem, but
// its name is removed. Meaning once the file object is closed, the data will be
// freed in filesystem by the OS.
func openFromArchive(archiveFile string, fileName string) (*os.File, error) {
	// Open zip archive and extract the fileName.
	zipReader, err := zip.OpenReader(archiveFile)
	if err != nil {
		return nil, errors.Wrap(err, "opening zip archive")
	}
	defer utils.IgnoreError(zipReader.Close)
	fileReader, err := zipReader.Open(fileName)
	if err != nil {
		return nil, errors.Wrap(err, "extracting")
	}
	defer utils.IgnoreError(fileReader.Close)

	// Create a temporary file and remove it, keeping the file descriptor.
	tmpDir, err := os.MkdirTemp("", definitionsBaseDir)
	if err != nil {
		return nil, errors.Wrap(err, "creating temporary directory")
	}
	tmpFile, err := os.Create(filepath.Join(tmpDir, path.Base(fileName)))
	if err != nil {
		// Best effort to clean.
		_ = os.RemoveAll(tmpDir)
		return nil, errors.Wrap(err, "opening temporary file")
	}
	defer func() {
		if err != nil {
			_ = tmpFile.Close()
		}
	}()
	err = os.RemoveAll(tmpDir)
	if err != nil {
		return nil, errors.Wrap(err, "removing temporary file")
	}

	// Extract the file and copy contents to the temporary file, notice we
	// intentionally don't Sync(), to benefit from filesystem caching.
	_, err = io.Copy(tmpFile, fileReader)
	if err != nil {
		return nil, errors.Wrap(err, "writing to temporary file")
	}

	// Reset for caller's convenience.
	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		return nil, errors.Wrap(err, "writing to temporary file")
	}
	return tmpFile, nil
}

func (h *httpHandler) getV4(ctx context.Context, w http.ResponseWriter, r *http.Request, t updaterType, key string) {
	log.Debugf("Fetching scanner V4 %s file: %s", t, key)

	var err error
	var f *vulDefFile

	switch t {
	case mappingUpdaterType:
		f, err = h.openMostRecentV4File(ctx, t, mappingUpdaterKey, key)
	case vulnerabilityUpdaterType:
		if version.GetVersionKind(key) == version.NightlyKind {
			// get dev for nightly at this moment
			key = "dev"
		}
		f, err = h.openMostRecentV4File(ctx, t, key, "")
		if err == nil {
			w.Header().Set("Content-Type", "application/zstd")
		}
	default:
		errMsg := fmt.Sprintf("unknown Scanner V4 updater type: %s", t)
		log.Error(errMsg)
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, errMsg)
		return
	}

	if err != nil {
		errMsg := fmt.Sprintf("could not read %s file %q: %v", t, key, err)
		log.Error(errMsg)
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, errMsg)
		return
	}
	defer utils.IgnoreError(f.Close)
	http.ServeContent(w, r, f.Name(), f.modTime, f)
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

func getOfflineFileVersion(mf *os.File) (string, error) {
	var m manifest
	err := json.NewDecoder(mf).Decode(&m)
	if err != nil {
		return "", err
	}
	return m.Version, nil
}
