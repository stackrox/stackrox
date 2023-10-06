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
	baseURL = "https://storage.googleapis.com/scanner-v4-test/redhat-repository-mappings/"

	container2Repo = "container-name-repos-map.json"
	repo2Cpe       = "repository-to-cpe.json"
	zipFileName    = "archive.zip"
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

func (u *repoMappingUpdater) update() error {
	if err := u.doUpdate(); err != nil {
		log.Errorf("Failed to update Scanner v4 repository mapping from endpoint %q: %v", u.downloadURL, err)
		return err
	}
	return nil
}

func (u *repoMappingUpdater) doUpdate() error {
	tempDir, err := os.MkdirTemp("", "repomapping")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filesToDownload := []string{container2Repo, repo2Cpe}
	for _, file := range filesToDownload {
		err := downloadFromURL(baseURL+file, tempDir, file)
		if err != nil {
			return fmt.Errorf("failed to download %s: %v", file, err)
		}
	}
	log.Info("Finished downloading repo mapping data for Scanner V4")

	archivePath := filepath.Join(tempDir, zipFileName)
	outZip, err := archiveFiles(tempDir, archivePath, filesToDownload)
	if err != nil {
		return fmt.Errorf("failed to archive files: %v", err)
	}
	log.Infof("Successfully generated zip file: %v", outZip.Name())

	// Seek to the beginning of the outZip
	_, err = outZip.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("error seeking to the beginning of zip outZip: %w", err)
	}
	u.file.WriteContent(outZip)
	log.Infof("Successfully write to the content: %v", u.file.Path())
	return nil
}

func addFileToZip(zipWriter *zip.Writer, filename string) error {
	fileToZip, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() {
		if err = fileToZip.Close(); err != nil {
			log.Errorf("Failed to close file %q: %v", filename, err)
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

func archiveFiles(dirPath, zipFilePath string, files []string) (*os.File, error) {
	if files == nil || len(files) < 1 {
		return nil, fmt.Errorf("error: no repository mapping files available to archive")
	}
	// Create a new zip archive.
	outFile, err := os.Create(zipFilePath)
	if err != nil {
		return nil, err
	}

	zipWriter := zip.NewWriter(outFile)

	for _, file := range files {
		fileName := filepath.Join(dirPath, file)
		err = addFileToZip(zipWriter, fileName)
		if err != nil {
			zipWriter.Close()
			outFile.Close()
			return nil, err
		}
	}

	if err := zipWriter.Close(); err != nil {
		outFile.Close()
		return nil, err
	}

	return outFile, nil
}

func downloadFromURL(url, dir, filename string) error {
	const maxRetries = 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(url)
		if err != nil {
			lastErr = err
			time.Sleep(time.Second * 3)
			continue
		}

		if resp.StatusCode == http.StatusOK { // Success
			out, err := os.Create(filepath.Join(dir, filename))
			if err != nil {
				resp.Body.Close()
				return err
			}

			_, err = io.Copy(out, resp.Body)
			if err != nil {
				out.Close()
				return err
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
