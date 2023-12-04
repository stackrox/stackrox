package handler

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	backoff "github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	_       RequestedUpdater = (*v4Updater)(nil)
	randGen                  = rand.New(rand.NewSource(time.Now().UnixNano()))
) /**/

type v4Updater struct {
	file *file.File

	client      *http.Client
	downloadURL string
	interval    time.Duration
	once        sync.Once
	stopSig     concurrency.Signal
}

const (
	// TODO(ROX-20481): Replace this URL with prod GCS bucket
	baseURL = "https://storage.googleapis.com/scanner-v4-test/redhat-repository-mappings/mapping.zip"
)

// newV4Updater creates a new updater for RH repository mapping data.
func newV4Updater(file *file.File, client *http.Client, downloadURL string, interval time.Duration) *v4Updater {
	if downloadURL == "" {
		downloadURL = baseURL
	}
	return &v4Updater{
		file:        file,
		client:      client,
		downloadURL: downloadURL,
		interval:    interval,
		stopSig:     concurrency.NewSignal(),
	}
}

// Stop stops the updater.
func (u *v4Updater) Stop() {
	u.stopSig.Signal()
}

// Start starts the updater.
// The updater is only started once.
func (u *v4Updater) Start() {
	u.once.Do(func() {
		// Run the first update in a blocking-manner.
		err := u.update()
		if err != nil {
			log.Errorf("Failed to start Scanner v4 repository mapping updater: %v", err)
		}
		go u.runForever()
	})
}

func (u *v4Updater) OpenFile() (*os.File, time.Time, error) {
	return u.file.Open()
}

func (u *v4Updater) runForever() {
	timer := time.NewTimer(u.interval)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			err := u.update()
			if err != nil {
				log.Errorf("Failed to update Scanner v4 repository mapping: %v", err)
			}
			// Reset the timer with a new interval
			timer.Reset(u.interval + nextInterval())
		case <-u.stopSig.Done():
			return
		}
	}
}

func (u *v4Updater) update() error {
	if err := u.doUpdate(); err != nil {
		log.Errorf("Failed to update Scanner v4 repository mapping from endpoint %q: %v", u.downloadURL, err)
		return err
	}
	return nil
}

func (u *v4Updater) doUpdate() error {
	err := u.downloadFromURL(u.downloadURL)
	if err != nil {
		return err
	}

	log.Info("Finished downloading repo mapping data for Scanner V4")
	return nil
}

func (u *v4Updater) downloadFromURL(url string) error {
	download := func() error {
		resp, err := http.Get(url)
		if err != nil {
			return err
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Errorf("Error closing response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
		}

		lastModified, err := time.Parse(time.RFC1123, resp.Header.Get(lastModifiedHeader))
		if err != nil {
			return errors.Errorf("unable to determine upstream definitions file's modified time: %v", err)
		}
		err = u.file.Write(resp.Body, lastModified)
		if err != nil {
			return err
		}

		return nil // Success case
	}

	// Notify function will log the error and the backoff delay duration
	notify := func(err error, duration time.Duration) {
		log.Errorf("Error: %v. Retrying in %v...\n", err, duration)
	}

	b := backoff.NewExponentialBackOff()
	backoff.WithMaxRetries(b, 3) // Set max retry attempts to 3

	return backoff.RetryNotify(download, b, notify)
}

func nextInterval() time.Duration {
	addMinutes := []int{10, 20, 30, 40}
	randomMinutes := addMinutes[randGen.Intn(len(addMinutes))] // pick a random number from addMinutes
	return time.Duration(randomMinutes) * time.Minute
}
