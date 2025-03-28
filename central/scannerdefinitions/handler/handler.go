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
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	blob "github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/central/blob/snapshot"
	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"google.golang.org/grpc/codes"
)

const (
	// tmpDirPattern is the pattern for the directory in which all Scanner data is written.
	tmpDirPattern = "scannerdefinitions-*"

	// scannerV2DiffFile is the name of the file which contains Scanner v2 diff data.
	scannerV2DiffFile = "diff.zip"
	// scannerV2DefsFileis the name of the file which contains offline Scanner v2 data.
	scannerV2DefsFile = "scanner-defs.zip"
	// offlineScannerV2DefsBlobName represents the blob name of offline/fallback data file for Scanner v2.
	offlineScannerV2DefsBlobName = "/offline/scanner/scanner-defs.zip"

	// scannerV4DefsPrefix helps to search the v4 offline zip bundle for CVEs
	scannerV4DefsPrefix    = "v4-definitions-"
	scannerV4ManifestFile  = "manifest.json"
	scannerV4VulnSubDir    = "v4/vulnerability-bundles"
	scannerV4MappingSubDir = "v4/redhat-repository-mappings"
	scannerV4MappingFile   = "mapping.zip"
	// offlineScannerV4DefsBlobName represents the blob name of offline/fallback data file for Scanner V4.
	offlineScannerV4DefsBlobName = "/offline/scanner/v4/scanner-v4-defs.zip"

	// tmpUploadFile is the name of the file to which uploaded data is written, temporarily.
	tmpUploadFile = "offline-defs.zip"

	defaultCleanupInterval = 4 * time.Hour
	defaultCleanupAge      = 1 * time.Hour
)

//go:generate stringer -type=updaterType
type updaterType int

const (
	mappingUpdaterType updaterType = iota
	vulnerabilityUpdaterType
	v2UpdaterType
)

var (
	scannerUpdateBaseURL *url.URL

	client = &http.Client{
		Transport: proxy.RoundTripper(),
		Timeout:   5 * time.Minute,
	}

	log = logging.LoggerForModule()

	// v4FileMapping maps a URL query parameter to its associated
	// Scanner V4 map file.
	v4FileMapping = map[string]string{
		"name2repos": "repomapping/container-name-repos-map.json",
		"repo2cpe":   "repomapping/repository-to-cpe.json",
	}

	// mainVersionVariants is the set of all main version number variants. It allows
	// flexibility in specifying accepted versions in bundles. For example, is the
	// main version is `4.6.x-nightly-20241004`, and the offline bundle accepts
	// `4.6.x`, the variants would eventually allow us to accept the bundle when
	// comparing `4.6.x-nightly-20241004`, `4.6.x-nightly` and finally `4.6.x`.
	mainVersionVariants map[string]bool

	// errNotModified is used to inform the http handler that the requested file
	// has not be modified.
	errNotModified = errors.New("not modified")
)

type requestedUpdater struct {
	*updater
	lastRequestedTime time.Time
}

// manifest represents the manifest.json file
// containing Scanner V4 related metadata.
type manifest struct {
	VulnerabilityVersion string `json:"version"`
	ReleaseVersions      string `json:"release_versions"`
	releaseVersionsList  []string
}

// httpHandler handles HTTP GET and POST requests for vulnerability data.
type httpHandler struct {
	// online indicates if we are in online or offline mode.
	online bool
	// updaterInterval specifies the time period between subsequent updates, in online-mode.
	updaterInterval time.Duration
	// updatersLock protects updaters.
	updatersLock sync.Mutex
	// updaters stores the various updaters which may be required.
	updaters map[string]*requestedUpdater
	// dataDir is the root directory into which all data is downloaded.
	dataDir string
	// uploadPath is the file path to which "offline data" is uploaded prior to storing in blobStore.
	// This file will be under the dataDir directory.
	uploadPath string
	// blobStore provides access to the blob storage which stores the uploaded "offline data".
	blobStore blob.Datastore

	// uploadInProgress indicates when there is
	// a scanner definitions upload (POST) already in progress.
	// This is meant to protect from concurrent uploads which may overwrite each other.
	// Concurrent uploads are not expected nor supported.
	uploadInProgress atomic.Bool
}

func init() {
	var err error
	scannerUpdateBaseURL, err = url.Parse("https://definitions.stackrox.io")
	utils.CrashOnError(err) // This is very unexpected.

	// Parse the main version number variants, continue in a broken state if version
	// is not parseable.
	mainVersionVariants = make(map[string]bool)
	mainVersion := version.GetMainVersion()
	if mainVersion == "" {
		log.Error("v4 offline uploads are blocked: main version is empty")
		return
	}
	variants, err := version.Variants(mainVersion)
	if utils.ShouldErr(err) != nil {
		log.Errorf("v4 offline uploads are blocked: invalid main version format %q: %v",
			mainVersion, err)
		return
	}
	for _, v := range variants {
		mainVersionVariants[v] = true
	}
}

// New creates a new http.Handler to handle vulnerability data.
func New(blobStore blob.Datastore, opts handlerOpts) http.Handler {
	dataDir, err := os.MkdirTemp("", tmpDirPattern)
	utils.CrashOnError(err) // Fundamental problem if we cannot create a temp directory.

	h := &httpHandler{
		online:          !env.OfflineModeEnv.BooleanSetting(),
		updaterInterval: env.ScannerVulnUpdateInterval.DurationSetting(),
		dataDir:         dataDir,
		uploadPath:      filepath.Join(dataDir, tmpUploadFile),
		blobStore:       blobStore,
	}

	if !h.online {
		log.Info("In offline mode: scanner definitions will not be updated automatically")
		return h
	}

	log.Info("In online mode: scanner definitions will be updated automatically")

	h.updaters = make(map[string]*requestedUpdater)
	go h.cleanUpdatersPeriodic(opts.cleanupInterval, opts.cleanupAge)

	return h
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

// openOpts are options to open most recent V4 definition files.
type openOpts struct {
	// name is a generic name to refer to the definition bundle and its content.
	name string
	// urlPath specifies the update URL path when setting up online updaters.
	urlPath string
	// fileName specifies one file from within the scanner definition archive to
	// open, instead of returning the archive itself.
	fileName string
	// vulnVersion specifies the version of the vulnerability bundle for
	// vulnerability updaters.
	vulnVersion string
	// vulnBundle specifies the vulnerability bundle name for vulnerability updaters.
	vulnBundle string
	// offlineBlobName is the name of the offline blob to use.
	offlineBlobName string
	// modifiedSince is the parsed value of the If-Modified-Since header.
	modifiedSince *time.Time
}

func (h *httpHandler) get(w http.ResponseWriter, r *http.Request) {
	// URL parameters.
	uuid := r.URL.Query().Get(`uuid`)
	fileName := r.URL.Query().Get(`file`)
	v := r.URL.Query().Get(`version`)

	ctx := r.Context()

	var uType updaterType
	var opts openOpts
	opts.modifiedSince = h.parseModifiedSinceHeader(r)

	switch {
	case uuid != "":
		// Scanner V2 definitions.
		uType = v2UpdaterType
		opts.name = uuid
		opts.urlPath = uuid
		opts.fileName = fileName
		opts.offlineBlobName = offlineScannerV2DefsBlobName
	case fileName != "" && v == "":
		// If only file is requested, then this is request for Scanner V4 mapping file.
		v4FileName, exists := v4FileMapping[fileName]
		if !exists {
			writeErrorNotFound(w)
			return
		}
		uType = mappingUpdaterType
		opts.name = fileName
		opts.fileName = v4FileName
		opts.offlineBlobName = offlineScannerV4DefsBlobName
	case fileName == "" && v != "":
		// If only version is provided, this is for Scanner V4 vulnerabilities.
		uType = vulnerabilityUpdaterType
		bundle := "vulnerabilities.zip"
		opts.name = v
		opts.urlPath = path.Join(v, bundle)
		opts.vulnVersion = v
		opts.vulnBundle = bundle
		opts.offlineBlobName = offlineScannerV4DefsBlobName
	default:
		// No other parameter configurations are supported.
		writeErrorBadRequest(w)
		return
	}

	f, err := h.openDefinitions(ctx, uType, opts)
	if err != nil {
		if errors.Is(err, errNotModified) {
			writeNotModified(w)
			return
		}

		writeErrorForFile(w, err, opts.name)
		return
	}
	if f == nil {
		writeErrorNotFound(w)
		return
	}
	defer utils.IgnoreError(f.Close)
	serveContent(w, r, f.Name(), f.modTime, f)
}

// parseModifiedSinceHeader will extract a time from the If-Modified-Since header
// Logic inspired by the stdlib: https://cs.opensource.google/go/go/+/refs/tags/go1.23.7:src/net/http/fs.go;l=557
func (h *httpHandler) parseModifiedSinceHeader(r *http.Request) *time.Time {
	// Ignore the header if the request is not GET or HEAD.
	if r.Method != "GET" && r.Method != "HEAD" {
		return nil
	}

	ims := r.Header.Get("If-Modified-Since")
	if ims == "" {
		return nil
	}

	t, err := http.ParseTime(ims)
	if err != nil {
		return nil
	}

	return &t
}

func (h *httpHandler) openDefinitions(ctx context.Context, t updaterType, opts openOpts) (*vulDefFile, error) {
	log.Debugf("Fetching scanner data (online: %t): type %s: options: %#v", h.online, t, opts)

	// offline is nil if it does not exist.
	// This is ok, and we account for this.
	offline, err := h.openOfflineDefinitions(ctx, t, opts)
	if err != nil {
		if !h.online {
			// If offline mode, all errors should be returned.
			return nil, err
		}

		// Ignore not modified errors when in online mode, a newer file may be available.
		if !errors.Is(err, errNotModified) {
			return nil, err
		}
	}
	// If we are in offline-mode, do not bother fetching the latest online data.
	if !h.online {
		log.Debugf("offline bundle %s open result (`nil` if not found): %v", opts.offlineBlobName, offline)
		// Note: It is possible, and ok, the offline file is nil here.
		return offline, nil
	}

	online, err := h.openOnlineDefinitions(ctx, t, opts)
	if err != nil {
		return nil, err
	}

	newer, older := online, offline
	defer func() {
		// We serve the newest of the online and offline data, so close the older one.
		if older != nil {
			utils.IgnoreError(older.Close)
		}
	}()

	// If the offline files are newer, return them instead.
	if offline != nil && offline.modTime.After(online.modTime) {
		newer, older = offline, online
	}

	return newer, nil
}

// openOfflineDefinitions gets desired offline file from compressed bundle.
// If the offline file does not exist, it is not an error, and (nil, nil) is returned.
func (h *httpHandler) openOfflineDefinitions(ctx context.Context, t updaterType, opts openOpts) (*vulDefFile, error) {
	log.Debugf("Fetching offline data for updater: type %s: options: %#v", t, opts)
	offlineBlob, err := h.openOfflineBlob(ctx, opts.offlineBlobName, opts.modifiedSince)
	if err != nil {
		return nil, fmt.Errorf("opening offline definitions: %s: %w", opts.offlineBlobName, err)
	}
	if offlineBlob == nil {
		log.Debugf("Offline blob %s does not exist", opts.offlineBlobName)
		return nil, nil
	}

	var success bool
	offlineFile := offlineBlob
	defer func() {
		// If we fail from here, then we should close "offlineBlob". Similarly, if the client desired a file inside "offlineBlob",
		// then no need to keep "offlineBlob" open any longer.
		if !success || offlineFile != offlineBlob {
			utils.IgnoreError(offlineBlob.Close)
		}
	}()

	switch t {
	case v2UpdaterType:
		if opts.fileName == "" {
			break
		}
		// We want to find a specific file inside the toplevel online file.
		fallthrough
	case mappingUpdaterType:
		// Search mapping file.
		fileName := filepath.Base(opts.fileName)
		targetFile, _, err := h.openFromArchive(offlineBlob.Name(), fileName)
		if err != nil {
			return nil, err
		}
		offlineFile = &vulDefFile{File: targetFile, modTime: offlineBlob.modTime}
	case vulnerabilityUpdaterType:
		// Check version information in manifest.
		mf, _, err := h.openFromArchive(offlineBlob.Name(), scannerV4ManifestFile)
		if err != nil {
			log.Errorf("open manifest: %v", err)
			return nil, err
		}
		defer utils.IgnoreError(mf.Close)

		mv, err := readV4Manifest(mf)
		if err != nil {
			return nil, fmt.Errorf("reading v4 definitions manifest: %w", err)
		}

		// Only validate offline definition versions on release builds.
		if buildinfo.ReleaseBuild && mv.VulnerabilityVersion != opts.vulnVersion {
			log.Warnf("v4 offline bundle vulnerability version mismatch: "+
				"the stored version != requested vulnerability version (%s != %s)",
				mv.VulnerabilityVersion, opts.vulnVersion)
			return nil, nil
		}

		vulns, _, err := h.openFromArchive(offlineBlob.Name(), opts.vulnBundle)
		if err != nil {
			log.Errorf("open vulnerabilities: %s: %v", opts.vulnBundle, err)
			return nil, err
		}

		offlineFile = &vulDefFile{File: vulns, modTime: offlineBlob.modTime}
	default:
		return nil, fmt.Errorf("unknown updater type: %s", t)
	}

	success = true
	return offlineFile, nil
}

// openOfflineBlob opens the offline scanner data identified by the given blobName.
// If the blob does not exist, then no file nor error is returned (nil, nil).
// If the blob exists but has not been modified, errNotModified is returned.
func (h *httpHandler) openOfflineBlob(ctx context.Context, blobName string, modifiedSince *time.Time) (*vulDefFile, error) {
	allAccessCtx := sac.WithAllAccess(ctx)

	// Do not take blob snapshot if it has NOT been modified.
	if modifiedSince != nil {
		blob, exists, err := h.blobStore.GetMetadata(allAccessCtx, blobName)
		if err != nil {
			log.Warnf("Cannot test if Blob has been modified %q: %v", blobName, err)
			return nil, err
		}
		if !exists {
			return nil, nil
		}
		// Inspired by: https://cs.opensource.google/go/go/+/refs/tags/go1.23.7:src/net/http/fs.go;l=571
		if blobModifiedTime(blob).Truncate(time.Second).Compare(*modifiedSince) <= 0 {
			return nil, errNotModified
		}
	}

	snap, err := snapshot.TakeBlobSnapshot(allAccessCtx, h.blobStore, blobName)
	if err != nil {
		// If the blob does not exist, return no reader.
		if errors.Is(err, snapshot.ErrBlobNotExist) {
			return nil, nil
		}
		log.Warnf("Cannot take a snapshot of Blob %q: %v", blobName, err)
		return nil, err
	}

	return &vulDefFile{snap.File, blobModifiedTime(snap.GetBlob()), snap.Close}, nil
}

// blobModifiedTime will convert a blob modified time to a time.Time. If the
// blob's modified time is nil, will return the zero time.
func blobModifiedTime(blob *storage.Blob) time.Time {
	modTime := time.Time{}
	if blob == nil {
		return modTime
	}

	if t := protocompat.NilOrTime(blob.ModifiedTime); t != nil {
		modTime = *t
	}

	return modTime
}

// errNotExist is a wrapper meant to turn an error into a fs.ErrNotExist error.
type errNotExist struct {
	error
}

// Is informs errors.Is that errNotExist is equivalent to fs.ErrNotExist.
//
// See https://pkg.go.dev/errors#Is for more information.
func (e errNotExist) Is(target error) bool {
	return target == fs.ErrNotExist
}

// openOnlineDefinitions gets desired "online" file, which is pulled and managed
// by the updater.
func (h *httpHandler) openOnlineDefinitions(_ context.Context, t updaterType, opts openOpts) (*vulDefFile, error) {
	u, err := h.getUpdater(t, opts.urlPath)
	if err != nil {
		return nil, err
	}
	// Ensure the updater is running (if already running, nothing will happen).
	u.Start()
	online, modTime, err := u.file.Open()
	if err != nil {
		return nil, err
	}
	if online == nil {
		// Return an errNotExist so it's clear this is meant to be handled the same as fs.ErrNotExist errors.
		return nil, errNotExist{fmt.Errorf("scanner %s file %s not found", t, opts.urlPath)}
	}
	log.Debugf("Online data %s file %s is available: %s", t, opts.urlPath, online.Name())

	var success bool
	f := &vulDefFile{File: online, modTime: modTime}
	defer func() {
		// If we fail from here, then we should close "online". Similarly, if the client desired a file inside "online",
		// then no need to keep "online" open any longer.
		if !success || f.File != online {
			utils.IgnoreError(online.Close)
		}
	}()

	switch t {
	case v2UpdaterType:
		if opts.fileName == "" {
			break
		}
		// We want to find a specific file inside the toplevel online file.
		fallthrough
	case mappingUpdaterType:
		targetFile, _, err := h.openFromArchive(online.Name(), opts.fileName)
		if err != nil {
			return nil, err
		}
		f.File = targetFile
	case vulnerabilityUpdaterType:
		// Always return the toplevel online file.
	default:
		return nil, fmt.Errorf("unknown updater type: %s", t)
	}

	success = true
	return f, nil
}

// getUpdater gets or creates an updater for the scanner definitions identified
// by the given updater type and a URL path to the definitions file. If the
// updater was created, it is no started here, callers are expected to start it.
func (h *httpHandler) getUpdater(t updaterType, urlPath string) (*requestedUpdater, error) {
	h.updatersLock.Lock()
	defer h.updatersLock.Unlock()

	fileName := strings.ReplaceAll(filepath.Join(t.String(), urlPath), "/", "-")
	updater, exists := h.updaters[fileName]
	if !exists {
		var updateURL *url.URL
		switch t {
		case mappingUpdaterType:
			updateURL = scannerUpdateBaseURL.JoinPath(scannerV4MappingSubDir, scannerV4MappingFile)
		case vulnerabilityUpdaterType:
			updateURL = scannerUpdateBaseURL.JoinPath(scannerV4VulnSubDir, urlPath)
		case v2UpdaterType:
			updateURL = scannerUpdateBaseURL.JoinPath(urlPath, scannerV2DiffFile)
		default:
			return nil, fmt.Errorf("unknown updater type: %s", t)
		}
		filePath := filepath.Join(h.dataDir, fileName)
		updater = &requestedUpdater{
			updater: newUpdater(file.New(filePath), client, updateURL.String(), h.updaterInterval),
		}
		h.updaters[fileName] = updater
	}

	updater.lastRequestedTime = time.Now()
	return updater, nil
}

func (h *httpHandler) post(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("request missing body"))
		return
	}

	// Swap will set h.uploadInProgress to true and return the previous value.
	// If it was previously true, then there is already an upload in progress,
	// so we should abort the operation.
	if h.uploadInProgress.Swap(true) {
		httputil.WriteGRPCStyleError(w, codes.Aborted, errors.New("scanner definitions upload already in progress"))
		return
	}
	// There are no other uploads in progress at this point.
	// Once we exit this function, the upload is no longer in progress.
	defer h.uploadInProgress.Store(false)

	// Copy the request body into the filesystem.
	// If the file at h.uploadPath doesn't exist yet, it will be created.
	if err := fileutils.CopySrcToFile(h.uploadPath, r.Body); err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "copying uploaded scanner definitions"))
		return
	}

	if err := h.handleZipContentsFromVulnDump(r.Context()); err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, err)
		return
	}

	_, _ = w.Write([]byte("Successfully stored scanner vulnerability definitions"))
}

func (h *httpHandler) handleZipContentsFromVulnDump(ctx context.Context) error {
	zipR, err := zip.OpenReader(h.uploadPath)
	if err != nil {
		return fmt.Errorf("failed to open offline zip: %w", err)
	}
	defer utils.IgnoreError(zipR.Close)

	checkV4Manifest := func(zipF *zip.File) (bool, error) {
		// Extract the Scanner V4 file out of the ZIP.
		defs, size, err := h.openFromArchive(h.uploadPath, zipF.Name)
		if err != nil {
			return false, fmt.Errorf("failed to open definitions zip: %w", err)
		}
		defer utils.IgnoreError(defs.Close)
		// Extract the manifest file out of the extracted Scanner V4 defs file.
		// Use readFromArchive, as the defs file was already closed via openFromArchive.
		mf, err := h.readFromArchive(defs, size, scannerV4ManifestFile)
		if err != nil {
			return false, fmt.Errorf("failed to open definitions manifest: %w", err)
		}
		defer utils.IgnoreError(mf.Close)

		mv, err := readV4Manifest(mf)
		if err != nil {
			return false, fmt.Errorf("failed to read definitions manifest: %w", err)
		}

		// Returns true if this manifest supports this release as provided by its main
		// version number.
		for _, v := range mv.releaseVersionsList {
			if mainVersionVariants[v] {
				return true, nil
			}
		}
		return false, nil
	}

	// Map of offline definitions to add.
	defs := make(map[string]*zip.File)

	// Currently, we expect the offline ZIP to contain both Scanner V2 and V4 offline
	// definitions. We want to keep the door open to support other file types, so
	// keep the following loop sensible to structural changes in the zip.
	for _, zipF := range zipR.File {
		switch {
		case zipF.Name == scannerV2DefsFile:
			if _, ok := defs[offlineScannerV2DefsBlobName]; ok {
				return fmt.Errorf("v2 definitions: found duplicate definition: %s", zipF.Name)
			}
			log.Infof("v2 definitions: found supported bundle: %s", zipF.Name)
			defs[offlineScannerV2DefsBlobName] = zipF
		case features.ScannerV4.Enabled() && strings.HasPrefix(zipF.Name, scannerV4DefsPrefix):
			ok, err := checkV4Manifest(zipF)
			if err != nil {
				// If we cannot check if the V4 definitions is supported, then we assume the
				// bundle is not good, which is not entirely accurate but simplifies error
				// handling.
				return fmt.Errorf("invalid v4 definition: %s: %w", zipF.Name, err)
			}
			if !ok {
				// Ignore unsupported bundles.
				log.Debugf("v4 definitions: ignoring unsupported bundle: %s", zipF.Name)
				continue
			}
			if prev, ok := defs[offlineScannerV4DefsBlobName]; ok {
				return fmt.Errorf("v4 definitions: found more than one supported definitions file: %s and %s",
					prev.Name, zipF.Name)
			}
			log.Infof("v4 definitions: found supported bundle: %s", zipF.Name)
			defs[offlineScannerV4DefsBlobName] = zipF
		default:
			// Ignore any other files which may be in the ZIP.
		}
	}

	// Check bundle compatibility.
	var incompatible bool
	if _, ok := defs[offlineScannerV4DefsBlobName]; features.ScannerV4.Enabled() && !ok {
		incompatible = true
		log.Debugf("offline bundle compatibility check: missing V4 definitions (and V4 is enabled)")
	}
	if _, ok := defs[offlineScannerV2DefsBlobName]; !ok {
		incompatible = true
		log.Debugf("offline bundle compatibility check: missing V2 definitions")
	}
	if incompatible {
		// If the expected definitions were not found, we assume the bundle is incompatible.
		return fmt.Errorf("the uploaded "+
			"bundle is incompatible with release version number '%s' "+
			"please upload an offline bundle that supports this "+
			"release, and consider using `roxctl scanner download-db`",
			version.GetMainVersion())
	}

	var errList []error
	for blobName := range defs {
		zipF := defs[blobName]
		if err := h.handleScannerDefsFile(ctx, zipF, blobName); err != nil {
			err = fmt.Errorf("failed to load %s: %w", zipF.Name, err)
			log.Error(err)
			errList = append(errList, err)
		}
		log.Infof("Successfully loaded offline definition: %s", zipF.Name)
	}

	if len(errList) > 0 {
		return fmt.Errorf("failed to load %d out of %d scanner defs file(s): %v",
			len(errList), len(defs), errList)
	}

	return nil
}

func (h *httpHandler) handleScannerDefsFile(ctx context.Context, zipF *zip.File, blobName string) error {
	r, err := zipF.Open()
	if err != nil {
		return errors.Wrap(err, "opening compressed file")
	}
	defer utils.IgnoreError(r.Close)

	b := &storage.Blob{
		Name:         blobName,
		LastUpdated:  protocompat.TimestampNow(),
		ModifiedTime: protocompat.TimestampNow(),
		Length:       zipF.FileInfo().Size(),
	}

	if err := h.blobStore.Upsert(sac.WithAllAccess(ctx), b, r); err != nil {
		return errors.Wrap(err, "writing scanner definitions")
	}

	return nil
}

// openFromArchive returns the associated file for the given name within the ZIP archiveFile
// along with the file size.
//
// The returned file struct has a file descriptor allocated on the filesystem outside the ZIP, but
// its name is removed. Meaning: as soon as the file struct is closed, the data will be
// freed in filesystem by the OS. That also means there should not be any OS operations
// done on the returned file.
func (h *httpHandler) openFromArchive(archiveFile string, fileName string) (*os.File, int64, error) {
	zipReader, err := zip.OpenReader(archiveFile)
	if err != nil {
		return nil, 0, errors.Wrap(err, "opening zip archive")
	}
	defer utils.IgnoreError(zipReader.Close)

	return h.openFromZipReader(&zipReader.Reader, fileName)
}

// readFromArchive returns the associated file for the given name within the ZIP archive.
//
// The returned file struct has a file descriptor allocated on the filesystem outside the ZIP, but
// its name is removed. Meaning: as soon as the file struct is closed, the data will be
// freed in filesystem by the OS. That also means there should not be any OS operations
// done on the returned file.
func (h *httpHandler) readFromArchive(archive io.ReaderAt, size int64, fileName string) (*os.File, error) {
	zipReader, err := zip.NewReader(archive, size)
	if err != nil {
		return nil, errors.Wrap(err, "reading zip archive")
	}

	f, _, err := h.openFromZipReader(zipReader, fileName)
	return f, err
}

// openFromZipReader does the work for readFromArchive and openFromArchive.
// It should **not** be used outside of those functions.
func (h *httpHandler) openFromZipReader(zipReader *zip.Reader, fileName string) (*os.File, int64, error) {
	zipFile, err := zipReader.Open(fileName)
	if err != nil {
		return nil, 0, errors.Wrap(err, "extracting file")
	}
	defer utils.IgnoreError(zipFile.Close)

	// Create a temporary file and remove it for the OS to clean up once the
	// struct is closed.
	//
	// Ensure the file extension stays intact (via the *- prefix) so the HTTP server
	// can automatically pick up the Content-Type.
	//
	// Also, replace / with - to account for the mapping files, as
	// forward slash is invalid in the pattern accepted by os.CreateTemp.
	tmpFilePattern := "*-" + strings.ReplaceAll(fileName, "/", "-")
	tmpFile, err := os.CreateTemp(h.dataDir, tmpFilePattern)
	if err != nil {
		return nil, 0, errors.Wrap(err, "opening temporary file")
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()
	var success bool
	defer func() {
		// If this function is unsuccessful, then close the struct.
		if !success {
			utils.IgnoreError(tmpFile.Close)
		}
	}()

	// Extract the file and copy contents to the temporary file, notice we
	// intentionally don't Sync(), to benefit from filesystem caching.
	size, err := io.Copy(tmpFile, zipFile)
	if err != nil {
		return nil, 0, errors.Wrap(err, "writing to temporary file")
	}

	// Reset for caller's convenience.
	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		return nil, 0, errors.Wrap(err, "setting offset for temporary file")
	}

	success = true
	return tmpFile, size, nil
}

func readV4Manifest(mf io.Reader) (*manifest, error) {
	var m manifest
	err := json.NewDecoder(mf).Decode(&m)
	if err != nil {
		return nil, err
	}
	m.releaseVersionsList = strings.Fields(strings.TrimSpace(m.ReleaseVersions))
	return &m, nil
}

func serveContent(w http.ResponseWriter, r *http.Request, name string, modTime time.Time, content io.ReadSeeker) {
	log.Debugf("Serving vulnerability definitions from %s", filepath.Base(name))
	http.ServeContent(w, r, name, modTime, content)
}

// writeNotModified inspired by https://cs.opensource.google/go/go/+/refs/tags/go1.23.7:src/net/http/fs.go;l=622;
func writeNotModified(w http.ResponseWriter) {
	// RFC 7232 section 4.1:
	// a sender SHOULD NOT generate representation metadata other than the
	// above listed fields unless said metadata exists for the purpose of
	// guiding cache updates (e.g., Last-Modified might be useful if the
	// response does not have an ETag field).
	h := w.Header()
	delete(h, "Content-Type")
	delete(h, "Content-Length")
	delete(h, "Content-Encoding")
	if h.Get("Etag") != "" {
		delete(h, "Last-Modified")
	}
	w.WriteHeader(http.StatusNotModified)
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
	if errox.IsAny(err, fs.ErrNotExist, snapshot.ErrBlobNotExist) {
		writeErrorNotFound(w)
		return
	}

	httputil.WriteGRPCStyleErrorf(w, codes.Internal, "could not read vulnerability definition %s: %v", filepath.Base(path), err)
}

func (h *httpHandler) cleanUpdatersPeriodic(cleanupInterval, cleanupAge *time.Duration) {
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

	h.updatersLock.Lock()
	defer h.updatersLock.Unlock()

	for id, updatingHandler := range h.updaters {
		if now.Sub(updatingHandler.lastRequestedTime) > cleanupAge {
			// Updater has not been requested for a long time.
			// Clean it up.
			updatingHandler.Stop()
			delete(h.updaters, id)
		}
	}
}
