package cvss

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/central/scannerv4definitions/cvss/enricher"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

type cvssUpdater struct {
	file        *file.File
	client      *http.Client
	downloadURL string
	interval    time.Duration
	once        sync.Once
	stopSig     concurrency.Signal
	enricher    *enricher.Enricher
}

const tmpJson = "cvss.json"

var fp driver.Fingerprint

func NewUpdaterWithCvssEnricher(file *file.File, client *http.Client, downloadURL string, interval time.Duration) *cvssUpdater {
	e := &enricher.Enricher{}
	ctx := context.Background() // Or pass a context in if available.

	configFunc := func(cfg interface{}) error {
		c, ok := cfg.(*enricher.Config) // Type assertion for safety
		if !ok {
			return errors.New("invalid config type")
		}
		c.FeedRoot = &downloadURL
		return nil
	}

	err := e.Configure(ctx, configFunc, client)
	if err != nil {
		// TODO log config is bad
		return nil
	}

	return &cvssUpdater{
		file:        file,
		client:      client,
		downloadURL: downloadURL,
		interval:    interval,
		stopSig:     concurrency.NewSignal(),
		enricher:    e,
	}
}

func (u *cvssUpdater) Stop() {
	u.stopSig.Signal()
}

func (u *cvssUpdater) Start() {
	u.once.Do(func() {
		ctx := context.Background()
		u.doUpdate(ctx)
		go u.runForever()
	})
}

func (u *cvssUpdater) runForever() {
	t := time.NewTicker(u.interval)
	defer t.Stop()

	ctx := context.Background()

	for {
		select {
		case <-t.C:
			u.doUpdate(ctx)
		case <-u.stopSig.Done():
			return
		}
	}
}

func (u *cvssUpdater) doUpdate(ctx context.Context) error {
	// Run enrichment and gzip content directly into the temporary file
	log.Infof("Starting CVSS data enricher")
	zipFile, err := runEnricher(ctx, u.enricher)
	if err != nil {
		return err
	}

	defer func() {
		err := zipFile.Close()
		if err != nil {
			log.Errorf("Error closing the zip file: %v", err)
		}
	}()

	// Seek to the beginning of the file
	_, err = zipFile.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("error seeking to the beginning of zip file: %w", err)
	}

	// Call WriteContent to finalize the operation
	return u.file.WriteContent(zipFile)
}

func runEnricher(ctx context.Context, u *enricher.Enricher) (*os.File, error) {
	var err error
	var pathToJson string
	for i := 0; i < 5; i++ {
		pathToJson, _, err = u.FetchEnrichment(ctx, fp, tmpJson)
		if err == nil {
			break
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After((2 << i) * time.Second):
		}
	}

	if err != nil {
		return nil, err
	}
	if len(pathToJson) < 1 {
		return nil, err
	}

	zipFile, err := jsonToZip(pathToJson)
	if err != nil {
		return nil, err
	}

	// Not closing the zip file until it's been written to updater file.
	return zipFile, nil
}

// jsonToZip converts a given JSON file to a zip archive and returns its path.
func jsonToZip(jsonFilePath string) (*os.File, error) {
	// Creating a temporary zip file.
	zipFile, err := os.CreateTemp("", "archive-*.zip")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp zip file: %w", err)
	}

	// Create a new zip archive.
	zipWriter := zip.NewWriter(zipFile)

	// Open the JSON file for reading.
	jsonFile, err := os.Open(jsonFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open json file: %w", err)
	}
	defer func() {
		err := jsonFile.Close()
		if err != nil {
			log.Errorf("Error closing the zip file: %v", err)
		}
	}()

	// Add the JSON file to the zip archive.
	fileInfo, err := jsonFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	header, err := zip.FileInfoHeader(fileInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip file header: %w", err)
	}

	zipFileWriter, err := zipWriter.CreateHeader(header)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip file writer: %w", err)
	}

	_, err = io.Copy(zipFileWriter, jsonFile)
	if err != nil {
		return nil, fmt.Errorf("failed to write file to zip archive: %w", err)
	}

	err = zipWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close zip writer: %w", err)
	}

	err = os.RemoveAll(jsonFilePath)
	if err != nil {
		log.Errorf("Failed to delete json file: %w", err)
	}

	return zipFile, nil
}
