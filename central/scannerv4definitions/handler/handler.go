package handler

import (
	"archive/zip"
	"context"
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
	baseDir = "scannerv4mappings"

	// mappingFileName represents the offline zip bundle.
	mappingFileName = "mapping-data.zip"

	// mappingfileBlobName represents the blob name of offline/fallback zip bundle for repo mapping data for Scanner.
	mappingfileBlobName = "/offline/scannerV4/" + mappingFileName

	repoMappingURL = "https://storage.googleapis.com/scanner-v4-test/redhat-repository-mappings/"

	defaultUpdateInterval = 4 * time.Hour

	name2cpe = "container-name-repos-map.json"

	repo2cpe = "repository-to-cpe.json"
)

var (
	client = &http.Client{
		Transport: proxy.RoundTripper(),
		Timeout:   5 * time.Minute,
	}

	log     = logging.LoggerForModule()
	randGen = rand.New(rand.NewSource(time.Now().UnixNano()))

	fileMapping = map[string]string{
		"name2cpe": name2cpe,
		"repo2cpe": repo2cpe,
	}
)

type requestedUpdater struct {
	*mappingUpdater
	lastRequestedTime time.Time
}

// httpHandler handles HTTP GET and POST requests for vulnerability data.
type httpHandler struct {
	interval  time.Duration
	lock      sync.Mutex
	updater   *requestedUpdater
	dataDir   string
	blobStore blob.Datastore
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.get(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// New creates a new http.Handler to handle repo mapping data.
func New(blobStore blob.Datastore) http.Handler {
	h := &httpHandler{
		interval:  env.RepoMappingUpdateMaxInitialWait.DurationSetting(),
		blobStore: blobStore,
	}
	h.initializeUpdater(context.Background())
	return h
}

func (h *httpHandler) initializeUpdater(ctx context.Context) {
	var err error
	utils.CrashOnError(err)

	h.updater = &requestedUpdater{}
	go h.fetchMappingData(ctx)
}

func (h *httpHandler) fetchMappingData(ctx context.Context) {
	ticker := time.NewTicker(env.RepoMappingUpdateMaxInitialWait.DurationSetting())
	defer ticker.Stop()
	log.Infof("Starting the updater loop")
	h.getUpdater()

	for {
		select {
		case <-ctx.Done():
			log.Infof("Context done: %v", ctx.Err())
			return
		case <-ticker.C:
			err := h.updater.update()
			if err != nil {
				log.Errorf("Error updating repo mapping data: %v", err)
			} else {
				err = h.handleRepoMappingFile(ctx)
				if err != nil {
					log.Errorf("Error handling repo mapping data file: %v", err)
				}
			}

			interval := nextInterval()
			ticker.Reset(interval)
		case <-h.updater.stopSig.Done():
			return
		}
	}
}

func (h *httpHandler) get(w http.ResponseWriter, r *http.Request) {
	// Get all URL query parameters
	fileType := r.URL.Query().Get("file")
	fileName, exists := fileMapping[fileType]
	if !exists || fileType == "" {
		http.Error(w, "Invalid file name", http.StatusBadRequest)
		return
	}

	offlineFile, modTime, err := h.openOfflineFile(context.Background(), mappingfileBlobName)
	if err != nil || offlineFile == nil {
		http.Error(w, "offline file not found", http.StatusNotFound)
		return
	}
	if modTime == nil {
		http.Error(w, "modification time is not available", http.StatusInternalServerError)
		return
	}

	targetFile, err := ExtractJSONFromZip(offlineFile.Name(), h.dataDir, fileName)
	if err != nil || targetFile == nil {
		log.Errorf("Failed to find JSON file: %v", err)
		http.Error(w, "offline file not found", http.StatusNotFound)
		return
	}

	http.ServeContent(w, r, filepath.Join(h.dataDir, fileName), *modTime, targetFile)
}

func (h *httpHandler) getUpdater() {
	h.lock.Lock()
	defer h.lock.Unlock()

	var err error
	h.dataDir, err = os.MkdirTemp("", baseDir)
	if err != nil {
		log.Errorf("Error creating directory: %v", err)
		return
	}
	pathToFile := filepath.Join(h.dataDir, "mapping.zip")

	if h.updater == nil || h.updater.mappingUpdater == nil {
		h.updater = &requestedUpdater{
			mappingUpdater: NewMappingUpdater(
				file.New(pathToFile),
				client,
				repoMappingURL,
				h.interval,
			),
		}
		h.updater.lastRequestedTime = time.Now()
		log.Infof("Created repo mapping data updater.")
	}
}

func (h *httpHandler) handleRepoMappingFile(ctx context.Context) error {
	file, err := os.Open(h.updater.file.Path())
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file info
	info, err := file.Stat()
	if err != nil {
		return err
	}

	// POST requests only update the offline feed.
	b := &storage.Blob{
		Name:         mappingfileBlobName,
		LastUpdated:  timestamp.TimestampNow(),
		ModifiedTime: timestamp.TimestampNow(),
		Length:       info.Size(),
	}

	if err := h.blobStore.Upsert(sac.WithAllAccess(ctx), b, file); err != nil {
		return errors.Wrap(err, "writing scanner definitions")
	}

	return nil
}

func (h *httpHandler) openOfflineFile(ctx context.Context, fileName string) (*os.File, *time.Time, error) {
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

// ExtractJSONFromZip ExtractExampleJSONFromZip ExtractJSONFromZip extracts the json file from the provided ZIP file
// and writes it to the specified directory.
func ExtractJSONFromZip(zipFilePath, outputDir string, fileName string) (*os.File, error) {
	r, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	for _, f := range r.File {
		fName := filepath.Base(f.Name)
		if fName == fileName {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			outputFile, err := os.Create(filepath.Join(outputDir, fileName)) // keep the file name same
			if err != nil {
				return nil, err
			}

			_, err = io.Copy(outputFile, rc)
			if err != nil {
				outputFile.Close()
				return nil, err
			}
			return outputFile, nil
		}
	}

	return nil, nil // Return nil if file not found in the zip
}
