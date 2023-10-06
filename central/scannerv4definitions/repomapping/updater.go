package repomapping

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"path/filepath"

	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

type repoMappingUpdater struct {
	file *file.File

	client      *http.Client
	downloadURL string
	interval    time.Duration
	once        sync.Once
	stopSig     concurrency.Signal
}

const (
	baseURL        = "https://storage.googleapis.com/scanner-v4-test/redhat-repository-mappings/"
	container2Repo = "container-name-repos-map.json"
	repo2Cpe       = "repository-to-cpe.json"
)

var (
	log = logging.LoggerForModule()
)

// NewUpdater creates a new updater.
func NewUpdater(file *file.File, client *http.Client, downloadURL string, interval time.Duration) *repoMappingUpdater {
	return &repoMappingUpdater{
		file:        file,
		client:      client,
		downloadURL: downloadURL,
		interval:    interval,
		stopSig:     concurrency.NewSignal(),
	}
}

// Stop stops the updater.
func (u *repoMappingUpdater) Stop() {
	u.stopSig.Signal()
}

// Start starts the updater.
// The updater is only started once.
func (u *repoMappingUpdater) Start() {
	u.once.Do(func() {
		// Run the first update in a blocking-manner.
		u.update()
		go u.runForever()
	})
}

func (u *repoMappingUpdater) runForever() {
	t := time.NewTicker(u.interval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			u.update()
		case <-u.stopSig.Done():
			return
		}
	}
}

func (u *repoMappingUpdater) update() {
	if err := u.doUpdate(); err != nil {
		log.Errorf("Scanner vulnerability updater for endpoint %q failed: %v", u.downloadURL, err)
	}
}

func (u *repoMappingUpdater) doUpdate() error {

	// Downloading known files A.json and B.json
	for _, file := range []string{container2Repo, repo2Cpe} {
		err := downloadFromURL(baseURL+file, filePath, file)
		if err != nil {
			return fmt.Errorf("Failed to download %s: %v\n", file, err)
		} else {
			log.Infof("Successfully downloaded %s\n", file)
		}
	}
	return nil
}

func addFileToZip(zipWriter *zip.Writer, filename string) error {
	fileToZip, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := fileToZip.Close()
		if closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	// Get the file information
	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	// Create a header based on the file information
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileToZip)
	return err
}

// err := archiveFiles("mydir", "mydir/archive.zip")
//
//	if err != nil {
//		fmt.Println(err)
//	}
func archiveFiles(dirPath, zipFilePath string, files []string) error {
	if files == nil || len(files) < 1 {
		return fmt.Errorf("error: no repository mapping files available to archive")
	}
	// Create a new zip archive.
	outFile, err := os.Create(zipFilePath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	for _, file := range files {
		fileName := filepath.Join(dirPath, file)
		err = addFileToZip(zipWriter, fileName)
		if err != nil {
			return err
		}
	}

	return nil
}

func downloadFromURL(url, dir, filename string) error {
	const maxRetries = 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(url)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusOK { // Success
			out, err := os.Create(filepath.Join(dir, filename))
			if err != nil {
				lastErr = err
				resp.Body.Close()
				continue
			}

			_, err = io.Copy(out, resp.Body)
			if err != nil {
				lastErr = err
				out.Close()
				continue
			}
			resp.Body.Close()
			out.Close()
			return nil
		} else {
			resp.Body.Close()
			time.Sleep(time.Second * 3)
			continue
		}

		break
	}
	return lastErr
}
