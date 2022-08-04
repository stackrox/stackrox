package handler

import (
	"archive/zip"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/fetcher"
	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc/codes"
)

const (
	definitionsBaseDir = "scannerdefinitions"

	// scannerDefsSubZipName represents the offline zip bundle for CVEs for Scanner.
	scannerDefsSubZipName = "scanner-defs.zip"
	// K8sIstioCveZipName represents the zip bundle for k8s/istio CVEs.
	K8sIstioCveZipName = "k8s-istio.zip"

	// offlineScannerDefsName represents the offline/fallback zip bundle for CVEs for Scanner.
	offlineScannerDefsName = scannerDefsSubZipName

	scannerUpdateDomain    = "https://definitions.stackrox.io"
	scannerUpdateURLSuffix = "diff.zip"

	defaultCleanupInterval = 4 * time.Hour
	defaultCleanupAge      = 1 * time.Hour
)

var (
	client = &http.Client{
		Transport: proxy.RoundTripper(),
		Timeout:   5 * time.Minute,
	}

	log = logging.LoggerForModule()
)

type requestedUpdater struct {
	*updater
	lastRequestedTime time.Time
}

// httpHandler handles HTTP GET and POST requests for vulnerability data.
type httpHandler struct {
	cveManager fetcher.OrchestratorIstioCVEManager

	online        bool
	interval      time.Duration
	lock          sync.Mutex
	updaters      map[string]*requestedUpdater
	onlineVulnDir string
	offlineFile   *file.File
}

// New creates a new http.Handler to handle vulnerability data.
func New(cveManager fetcher.OrchestratorIstioCVEManager, opts handlerOpts) http.Handler {
	h := &httpHandler{
		cveManager: cveManager,

		online:   !env.OfflineModeEnv.BooleanSetting(),
		interval: env.ScannerVulnUpdateInterval.DurationSetting(),
	}

	h.initializeOfflineVulnDump(opts.offlineVulnDefsDir)

	if h.online {
		h.initializeUpdaters(opts.cleanupInterval, opts.cleanupAge)
	} else {
		log.Info("In offline mode: scanner definitions will not be updated automatically")
	}

	return h
}

func (h *httpHandler) initializeOfflineVulnDump(vulnDefsDir string) {
	if vulnDefsDir == "" {
		vulnDefsDir = filepath.Join(migrations.DBMountPath(), definitionsBaseDir)
	}

	h.offlineFile = file.New(filepath.Join(vulnDefsDir, offlineScannerDefsName))
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
	// Open the most recent definitions file for the provided `uuid`.
	uuid := r.URL.Query().Get(`uuid`)
	f, modTime, err := h.openMostRecentDefinitions(uuid)
	if err != nil {
		writeErrorForFile(w, err, uuid)
		return
	}

	// It is possible no offline Scanner definitions were uploaded, or Central cannot
	// reach the definitions object, or there is no definitions for the given
	// `uuid`; in any of those cases, `f` will be `nil`.
	if f == nil {
		writeErrorNotFound(w)
		return
	}

	defer utils.IgnoreError(f.Close)

	// If `file` was provided, extract from definitions' bundle to a
	// temporary file and serve that instead.
	fileName := r.URL.Query().Get(`file`)
	if fileName != "" {
		f, err = openFromArchive(f.Name(), fileName)
		if err != nil {
			writeErrorForFile(w, err, fileName)
			return
		}
		defer utils.IgnoreError(f.Close)
	}

	serveContent(w, r, f.Name(), modTime, f)
}

func writeErrorNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("No scanner definitions found"))
}

func writeErrorForFile(w http.ResponseWriter, err error, path string) {
	if errors.Is(err, fs.ErrNotExist) {
		writeErrorNotFound(w)
		return
	}

	httputil.WriteGRPCStyleErrorf(w, codes.Internal, "could not read file %s: %v", filepath.Base(path), err)
}

func serveContent(w http.ResponseWriter, r *http.Request, name string, modTime time.Time, content io.ReadSeeker) {
	log.Debugf("Serving vulnerability definitions from %s", filepath.Base(name))
	http.ServeContent(w, r, name, modTime, content)
}

// getUpdater gets or creates the updater for the scanner definitions
// identified by the given uuid.
// If the updater is created here, it is no started here, as it is a blocking operation.
func (h *httpHandler) getUpdater(uuid string) *requestedUpdater {
	h.lock.Lock()
	defer h.lock.Unlock()

	u, exists := h.updaters[uuid]
	if !exists {
		filePath := filepath.Join(h.onlineVulnDir, uuid+".zip")

		h.updaters[uuid] = &requestedUpdater{
			updater: newUpdater(
				file.New(filePath),
				client,
				strings.Join([]string{scannerUpdateDomain, uuid, scannerUpdateURLSuffix}, "/"),
				h.interval,
			),
		}

		u = h.updaters[uuid]
	}

	u.lastRequestedTime = time.Now()

	return u
}

func (h *httpHandler) updateK8sIstioCVEs(zipPath string) {
	if !h.online {
		h.cveManager.Update(zipPath, false)
	}
}

func (h *httpHandler) handleScannerDefsFile(zipF *zip.File) error {
	r, err := zipF.Open()
	if err != nil {
		return errors.Wrap(err, "opening ZIP reader")
	}
	defer utils.IgnoreError(r.Close)

	// POST requests only update the offline feed.
	if err := h.offlineFile.Write(r, zipF.Modified); err != nil {
		return errors.Wrap(err, "writing scanner definitions")
	}

	return nil
}

func (h *httpHandler) handleZipContentsFromVulnDump(zipPath string) error {
	zipR, err := zip.OpenReader(zipPath)
	if err != nil {
		return errors.Wrap(err, "couldn't open file as zip")
	}
	defer utils.IgnoreError(zipR.Close)

	var scannerDefsFileFound bool
	for _, zipF := range zipR.File {
		if zipF.Name == scannerDefsSubZipName {
			if err := h.handleScannerDefsFile(zipF); err != nil {
				return errors.Wrap(err, "couldn't handle scanner-defs sub file")
			}
			scannerDefsFileFound = true
			continue
		} else if zipF.Name == K8sIstioCveZipName {
			h.updateK8sIstioCVEs(zipPath)
		}
	}

	if !scannerDefsFileFound {
		return errors.New("scanner defs file not found in upload zip; wrong zip uploaded?")
	}
	return nil
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

	if err := h.handleZipContentsFromVulnDump(tempFile); err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
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

// openMostRecentDefinitions opens the latest Scanner Definitions based on
// modification time. It's either the one selected by `uuid` if present and
// online, otherwise fallback to the manually uploaded definitions. The file
// object can be `nil` if the definitions file does not exist, rather than
// returning an error.
func (h *httpHandler) openMostRecentDefinitions(uuid string) (file *os.File, modTime time.Time, err error) {
	// If in offline mode or uuid is not provided, default to the offline file.
	if !h.online || uuid == "" {
		file, modTime, err = h.offlineFile.Open()
		return
	}

	// Start the updater, can be called multiple times for the same uuid, but will
	// only start the updater once. The Start() call blocks if the definitions were
	// not downloaded yet.
	u := h.getUpdater(uuid)
	u.Start()

	// Open both the "online" and "offline", and save their modification times.
	onlineFile, onlineTime, err := u.file.Open()
	if err != nil {
		return
	}
	offlineFile, offlineTime, err := h.offlineFile.Open()
	if err != nil {
		utils.IgnoreError(onlineFile.Close)
		return
	}

	// Return the most recent file, notice that if both don't exist, nil is returned
	// since modification time will be zero.

	if offlineTime.After(onlineTime) {
		file, modTime = offlineFile, offlineTime
		utils.IgnoreError(onlineFile.Close)
	} else {
		file, modTime = onlineFile, onlineTime
		utils.IgnoreError(offlineFile.Close)
	}

	return
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
