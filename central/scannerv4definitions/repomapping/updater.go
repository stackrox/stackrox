package repomapping

import (
	"net/http"
	"time"

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
	req, err := http.NewRequest(http.MethodGet, u.downloadURL, nil)
	if err != nil {
		return errors.Wrap(err, "constructing request")
	}
}
