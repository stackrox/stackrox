package handler

import (
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	lastModifiedHeader    = "Last-Modified"
	ifModifiedSinceHeader = "If-Modified-Since"
)

var (
	_       RequestedUpdater = (*updater)(nil)
	randGen                  = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// updater periodically updates a file by downloading the contents from the downloadURL.
type updater struct {
	file *file.File

	client      *http.Client
	downloadURL string
	interval    time.Duration
	once        sync.Once
	stopSig     concurrency.Signal
}

// newUpdater creates a new updater.
func newUpdater(file *file.File, client *http.Client, downloadURL string, interval time.Duration) *updater {
	return &updater{
		file:        file,
		client:      client,
		downloadURL: downloadURL,
		interval:    interval,
		stopSig:     concurrency.NewSignal(),
	}
}

// Stop stops the updater.
func (u *updater) Stop() {
	u.stopSig.Signal()
}

// Start starts the updater.
// The updater is only started once.
func (u *updater) Start() {
	u.once.Do(func() {
		// Run the first update in a blocking-manner.
		u.update()
		go u.runForever()
	})
}

func (u *updater) OpenFile() (*os.File, time.Time, error) {
	return u.file.Open()
}

func (u *updater) runForever() {
	timer := time.NewTimer(u.interval)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			u.update()
			timer.Reset(u.interval + nextInterval())
		case <-u.stopSig.Done():
			return
		}
	}
}

func (u *updater) update() {
	if err := u.doUpdate(); err != nil {
		log.Errorf("Scanner vulnerability updater for endpoint %q failed: %v", u.downloadURL, err)
	}
}

func (u *updater) doUpdate() error {
	req, err := http.NewRequest(http.MethodGet, u.downloadURL, nil)
	if err != nil {
		return errors.Wrap(err, "constructing request")
	}

	// The returned *os.File is not used, but we need to be sure to close the file
	// to prevent leaking file descriptor.
	f, modTime, err := u.file.Open()
	if err != nil {
		return errors.Wrapf(err, "reading modified time of file %s", u.file.Path())
	}
	defer utils.IgnoreError(f.Close)

	req.Header.Set(ifModifiedSinceHeader, modTime.Format(http.TimeFormat))

	resp, err := u.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "executing request")
	}
	defer utils.IgnoreError(resp.Body.Close)

	if resp.StatusCode == http.StatusNotModified {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("invalid response from google storage; got code %d", resp.StatusCode)
	}

	lastModified, err := time.Parse(time.RFC1123, resp.Header.Get(lastModifiedHeader))
	if err != nil {
		return errors.Errorf("unable to determine upstream definitions file's modified time: %v", err)
	}

	return u.file.Write(resp.Body, lastModified)
}

func nextInterval() time.Duration {
	addMinutes := []int{10, 20, 30, 40}
	randomMinutes := addMinutes[randGen.Intn(len(addMinutes))] // pick a random number from addMinutes
	return time.Duration(randomMinutes) * time.Minute
}
