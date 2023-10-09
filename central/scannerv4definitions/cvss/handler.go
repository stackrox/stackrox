package cvss

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	blob "github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/central/blob/snapshot"
	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	definitionsBaseDir = "scannerv4definitions"

	// cvssDataZipName represents the offline zip bundle for CVEs for Scanner.
	cvssDataZipName = "cvss-data.zip"

	// offlineCvssDataBlobName represents the blob name of offline/fallback zip bundle for CVEs for Scanner.
	offlineCvssDataBlobName = "/offline/scannerV4/" + cvssDataZipName

	cvssDataURL = "https://storage.googleapis.com/scanner-v4-test/nvddata/"

	defaultUpdateInterval = 4 * time.Hour
)

var (
	client = &http.Client{
		Transport: proxy.RoundTripper(),
		Timeout:   5 * time.Minute,
	}

	log     = logging.LoggerForModule()
	randGen = rand.New(rand.NewSource(time.Now().UnixNano()))
)

type requestedUpdater struct {
	*cvssUpdater
	lastRequestedTime time.Time
}

// httpHandler handles HTTP GET and POST requests for vulnerability data.
type httpHandler struct {
	online      bool
	interval    time.Duration
	lock        sync.Mutex
	updater     *requestedUpdater
	cvssDataDir string
	blobStore   blob.Datastore
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.get(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// New creates a new http.Handler to handle vulnerability data.
func New(blobStore blob.Datastore) http.Handler {
	h := &httpHandler{
		interval:  env.CvssDataUpdateInterval.DurationSetting(),
		blobStore: blobStore,
	}
	h.initializeUpdater(context.Background())
	return h
}

func (h *httpHandler) initializeUpdater(ctx context.Context) {
	var err error
	utils.CrashOnError(err)

	h.updater = &requestedUpdater{}
	go h.fetchCvssData(ctx)
}

func (h *httpHandler) fetchCvssData(ctx context.Context) {
	ticker := time.NewTicker(env.CvssDataUpdateMaxInitialWait.DurationSetting())
	defer ticker.Stop()
	log.Infof("Starting the updater loop")
	h.getUpdater()

	for {
		select {
		case <-ctx.Done():
			log.Infof("Context done: %v", ctx.Err())
			return
		case <-ticker.C:
			err := h.updater.doUpdate(ctx)
			if err != nil {
				log.Errorf("Error updating CVSS data: %v", err)
			} else {
				err = h.handleCvssDataFile(ctx)
				if err != nil {
					log.Errorf("Error handling CVSS data file: %v", err)
				}
			}

			interval := nextInterval()
			ticker.Reset(interval)
		}
	}
}

func (h *httpHandler) get(w http.ResponseWriter, r *http.Request) {
	file, modTime, err := h.openOfflineBlob(context.Background(), offlineCvssDataBlobName)
	if err != nil || file == nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	if modTime == nil {
		http.Error(w, "modification time is not available", http.StatusInternalServerError)
		return
	}
	serveContent(w, r, file.Name(), *modTime, file)
}

func serveContent(w http.ResponseWriter, r *http.Request, name string, modTime time.Time, content io.ReadSeeker) {
	log.Debugf("Serving CVSS data from %s", filepath.Base(name))
	http.ServeContent(w, r, name, modTime, content)
}

// getUpdater gets or creates the updater for the scanner definitions
// identified by the given uuid.
// If the updater is created here, it is no started here, as it is a blocking operation.
func (h *httpHandler) getUpdater() {
	h.lock.Lock()
	defer h.lock.Unlock()

	var err error
	h.cvssDataDir, err = os.MkdirTemp("", definitionsBaseDir)
	if err != nil {
		log.Errorf("Error creating directory: %v", err)
		return
	}
	pathToFile := filepath.Join(h.cvssDataDir, "cvss.zip")

	if h.updater == nil || h.updater.cvssUpdater == nil {
		h.updater = &requestedUpdater{
			cvssUpdater: NewUpdaterWithCvssEnricher(
				file.New(pathToFile),
				client,
				cvssDataURL,
				h.interval,
			),
		}
		h.updater.lastRequestedTime = time.Now()
		log.Infof("Created CVSS data updater.")
	}
}

// handleCvssDataFile handles the CVSS data file by reading its contents,
// validating that there is exactly one file in the zip archive,
// and then processing the file contents to create and upsert a blob
// in the blob store.
func (h *httpHandler) handleCvssDataFile(ctx context.Context) error {
	log.Infof("Handling CVSS data file.") // Log start

	zipPath := filepath.Join(h.cvssDataDir, "cvss.zip")
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		if errors.As(err, &os.PathError{}) {
			// If error is of type *os.PathError, the file does not exist
			log.Errorf("Error: cvss.zip file does not exist: %v", err)
			return errors.Wrap(err, "cvss.zip file does not exist")
		}
		log.Errorf("Error opening cvss.zip: %v", err)
		return errors.Wrap(err, "error opening cvss.zip")
	}
	// Ensure the directory is removed after operations.
	defer utils.IgnoreError(r.Close) // Ensure the file is closed after operations.

	if len(r.File) != 1 {
		log.Errorf("Error: Expected exactly one consolidated CVSS data file, found %d", len(r.File))
		return fmt.Errorf("expected exactly one consolidated CVSS data file, found %d", len(r.File))
	}

	rc, err := r.File[0].Open()
	if err != nil {
		log.Errorf("Error opening cvss.zip by ZIP reader: %v", err)
		return errors.Wrap(err, "opening cvss.zip by ZIP reade")
	}
	defer func() {
		err := rc.Close()
		if err != nil {
			log.Errorf("Error closing ZIP reader: %v", err)
		}
	}()

	b := &storage.Blob{
		Name:         offlineCvssDataBlobName,
		LastUpdated:  timestamp.TimestampNow(),
		ModifiedTime: timestamp.TimestampNow(),
		Length:       int64(r.File[0].FileHeader.UncompressedSize64),
	}

	if err := h.blobStore.Upsert(sac.WithAllAccess(ctx), b, rc); err != nil {
		log.Errorf("Error writing cvss enrichment data: %v", err)
		return errors.Wrap(err, "writing cvss enrichment data")
	}

	// delete zip file only
	err = os.RemoveAll(zipPath)
	if err != nil {
		// log only, not returning error
		log.Errorf("Error removing cvss.zip file: %v", err)
	}
	log.Infof("Saved CVSS data file successfully.") // Log success
	return nil
}

func (h *httpHandler) openOfflineBlob(ctx context.Context, fileName string) (*os.File, *time.Time, error) {
	snap, err := snapshot.TakeBlobSnapshot(sac.WithAllAccess(ctx), h.blobStore, fileName)
	if err != nil {
		// If the blob does not exist, return no reader.
		if errors.Is(err, snapshot.ErrBlobNotExist) {
			return nil, nil, nil
		}
		log.Warnf("Cannnot take a snapshot of Blob %q: %v", fileName, err)
		return nil, nil, err
	}
	var modTime *time.Time
	if t := pgutils.NilOrTime(snap.GetBlob().ModifiedTime); t != nil {
		modTime = t
	}
	return snap.File, modTime, nil

}

func nextInterval() time.Duration {
	addMinutes := []int{10, 20, 30, 40}
	randomMinutes := addMinutes[randGen.Intn(len(addMinutes))] // pick a random number from addMinutes
	duration := defaultUpdateInterval + time.Duration(randomMinutes)*time.Minute
	return duration
}
