package propagation

import (
	"sync/atomic"
	"time"
)

// VersionFetcher is a function that retrieves the current version.
type VersionFetcher func() (int64, error)

// OnChangeCallback is called when the version changes.
type OnChangeCallback func(oldVersion, newVersion int64)

// Poller polls for version changes at a regular interval.
type Poller struct {
	fetch    VersionFetcher
	onChange OnChangeCallback
	interval time.Duration

	lastKnownVersion atomic.Int64
	stopChan         chan struct{}
	stoppedChan      chan struct{}
}

// NewPoller creates a new version poller.
func NewPoller(fetch VersionFetcher, onChange OnChangeCallback, interval time.Duration) *Poller {
	return &Poller{
		fetch:       fetch,
		onChange:    onChange,
		interval:    interval,
		stopChan:    make(chan struct{}),
		stoppedChan: make(chan struct{}),
	}
}

// Start begins polling for version changes in a goroutine.
func (p *Poller) Start() {
	go p.run()
}

// Stop signals the poller to stop and waits for it to finish.
func (p *Poller) Stop() {
	close(p.stopChan)
	<-p.stoppedChan
}

func (p *Poller) run() {
	defer close(p.stoppedChan)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.checkVersion()
		case <-p.stopChan:
			return
		}
	}
}

func (p *Poller) checkVersion() {
	newVersion, err := p.fetch()
	if err != nil {
		log.Errorf("failed to fetch version: %v", err)
		return
	}

	oldVersion := p.lastKnownVersion.Load()
	if newVersion != oldVersion {
		p.lastKnownVersion.Store(newVersion)
		p.onChange(oldVersion, newVersion)
	}
}
