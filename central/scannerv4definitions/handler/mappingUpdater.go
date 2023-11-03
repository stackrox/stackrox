package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	backoff "github.com/cenkalti/backoff/v3"
	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

type mappingUpdater struct {
	file *file.File

	client      *http.Client
	downloadURL string
	interval    time.Duration
	once        sync.Once
	stopSig     concurrency.Signal
}

const (
	// TODO(ROX-20481): Replace this URL 
	baseURL = "https://storage.googleapis.com/scanner-v4-test/redhat-repository-mappings/"

	mappingZip = "mapping.zip"
)

// NewMappingUpdater creates a new updater.
func NewMappingUpdater(file *file.File, client *http.Client, downloadURL string, interval time.Duration) *mappingUpdater {
	return &mappingUpdater{
		file:        file,
		client:      client,
		downloadURL: downloadURL,
		interval:    interval,
		stopSig:     concurrency.NewSignal(),
	}
}

// Stop stops the updater.
func (u *mappingUpdater) Stop() {
	u.stopSig.Signal()
}

// Start starts the updater.
// The updater is only started once.
func (u *mappingUpdater) Start() {
	u.once.Do(func() {
		// Run the first update in a blocking-manner.
		u.update()
		go u.runForever()
	})
}

func (u *mappingUpdater) runForever() {
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

func (u *mappingUpdater) update() error {
	if err := u.doUpdate(); err != nil {
		log.Errorf("Failed to update Scanner v4 repository mapping from endpoint %q: %v", u.downloadURL, err)
		return err
	}
	return nil
}

func (u *mappingUpdater) doUpdate() error {
	err := downloadFromURL(baseURL+mappingZip, u.file.Path())
	if err != nil {
		return fmt.Errorf("failed to download %s: %v", mappingZip, err)
	}

	log.Info("Finished downloading repo mapping data for Scanner V4")
	return nil
}

func downloadFromURL(url, pathToFile string) error {
	operation := func() error {
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			out, err := os.Create(pathToFile)
			if err != nil {
				return err
			}
			defer out.Close()

			_, err = io.Copy(out, resp.Body)
			return err
		}
		return fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	// Notify function will log the error and the backoff delay duration
	notify := func(err error, duration time.Duration) {
		fmt.Printf("Error: %v. Retrying in %v...\n", err, duration)
	}

	b := backoff.NewExponentialBackOff()
	backoff.WithMaxRetries(b, 3) // Set max retry attempts to 3

	return backoff.RetryNotify(operation, b, notify)
}

// The rest of your code...
