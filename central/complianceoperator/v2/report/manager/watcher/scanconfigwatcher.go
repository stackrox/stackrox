package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scan "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	ScanConfigTimeoutError   = errors.New("scan config watcher timed out")
	defaultScanConfigTimeout = defaultTimeout * 2
)

// GetScanConfigFromScan returns the ScanConfiguration associated with the given scan
func GetScanConfigFromScan(ctx context.Context, scan *storage.ComplianceOperatorScanV2, scanConfigDS scanConfigurationDS.DataStore) (*storage.ComplianceOperatorScanConfigurationV2, error) {
	return scanConfigDS.GetScanConfigurationByName(ctx, scan.GetScanConfigName())
}

// ScanConfigWatcher determines if a ScanConfiguration has running scans or has completed
type ScanConfigWatcher interface {
	PushScanResults(results *ScanWatcherResults) error
	Subscribe(snapshotID string)
	Stop()
	Finished() concurrency.ReadOnlySignal
}

// ScanConfigWatcherResults is returned when the watcher detects all the scans are completed
type ScanConfigWatcherResults struct {
	Ctx               context.Context
	WatcherID         string
	ReportSnapshotIDs []string
	ScanConfig        *storage.ComplianceOperatorScanConfigurationV2
	ScanResults       map[string]*ScanWatcherResults
	Error             error
}

type scanConfigWatcherImpl struct {
	ctx     context.Context
	cancel  func()
	scanC   chan *ScanWatcherResults
	stopped *concurrency.Signal

	scanDS    scan.DataStore
	profileDS profileDatastore.DataStore

	snapshotsLock     sync.Mutex
	readyQueue        readyQueue[*ScanConfigWatcherResults]
	scanConfigResults *ScanConfigWatcherResults
	scansToWait       set.StringSet
	totalResults      int
}

// NewScanConfigWatcher creates a new ScanConfigWatcher
func NewScanConfigWatcher(ctx context.Context, watcherID string, sc *storage.ComplianceOperatorScanConfigurationV2, scanDS scan.DataStore, profileDS profileDatastore.DataStore, queue readyQueue[*ScanConfigWatcherResults], snapshotIDs ...string) *scanConfigWatcherImpl {
	watcherCtx, cancel := context.WithCancel(ctx)
	finishedSignal := concurrency.NewSignal()
	ret := &scanConfigWatcherImpl{
		ctx:        watcherCtx,
		cancel:     cancel,
		stopped:    &finishedSignal,
		scanDS:     scanDS,
		profileDS:  profileDS,
		scanC:      make(chan *ScanWatcherResults),
		readyQueue: queue,
		scanConfigResults: &ScanConfigWatcherResults{
			Ctx:               ctx,
			WatcherID:         watcherID,
			ScanConfig:        sc,
			ReportSnapshotIDs: snapshotIDs,
			ScanResults:       make(map[string]*ScanWatcherResults),
		},
		scansToWait: set.NewStringSet(),
	}
	timeout := time.NewTicker(defaultScanConfigTimeout)
	go ret.run(timeout.C)
	return ret
}

// PushScanResults queues a ScanWatcherResults to be handled
func (w *scanConfigWatcherImpl) PushScanResults(results *ScanWatcherResults) error {
	select {
	case <-w.ctx.Done():
		return errors.New("The watcher is stopped")
	case w.scanC <- results:
		return nil
	}
}

// Subscribe a new readyQueue to the watcher
func (w *scanConfigWatcherImpl) Subscribe(id string) {
	concurrency.WithLock(&w.snapshotsLock, func() {
		w.scanConfigResults.ReportSnapshotIDs = append(w.scanConfigResults.ReportSnapshotIDs, id)
	})
}

// Stop the watcher
func (w *scanConfigWatcherImpl) Stop() {
	w.cancel()
}

// Finished indicates whether the watcher is finished or not
func (w *scanConfigWatcherImpl) Finished() concurrency.ReadOnlySignal {
	return w.stopped
}

func (w *scanConfigWatcherImpl) run(timerC <-chan time.Time) {
	defer func() {
		w.stopped.Signal()
		<-w.stopped.Done()
	}()
	for {
		select {
		case <-w.ctx.Done():
			log.Infof("Stopping scan config watcher")
			return
		case <-timerC:
			log.Warnf("Timeout waiting for the ScanConfiguration %s's scans to finish", w.scanConfigResults.ScanConfig.GetScanConfigName())
			concurrency.WithLock(&w.snapshotsLock, func() {
				w.scanConfigResults.Error = ScanConfigTimeoutError
				w.readyQueue.Push(w.scanConfigResults)
			})
			return
		case result := <-w.scanC:
			if err := w.handleScanResults(result); err != nil {
				log.Errorf("Unable to handle scan results %s: %v", result.Scan.GetScanName(), err)
			}
		}
		if w.totalResults != 0 && w.totalResults == len(w.scanConfigResults.ScanResults) {
			concurrency.WithLock(&w.snapshotsLock, func() {
				w.readyQueue.Push(w.scanConfigResults)
			})
		}
	}
}

func (w *scanConfigWatcherImpl) handleScanResults(result *ScanWatcherResults) error {
	// Here we have the scan config id and the scan
	if w.totalResults == 0 {
		var profileNames []string
		for _, p := range w.scanConfigResults.ScanConfig.GetProfiles() {
			profileNames = append(profileNames, p.GetProfileName())
		}
		log.Debugf("Profiles %v", profileNames)
		var clusters []string
		for _, c := range w.scanConfigResults.ScanConfig.GetClusters() {
			clusters = append(clusters, c.GetClusterId())
		}
		profileQuery := search.NewQueryBuilder().
			AddExactMatches(
				search.ComplianceOperatorProfileName,
				profileNames...,
			).
			AddExactMatches(
				search.ClusterID,
				clusters...,
			).ProtoQuery()
		profiles, err := w.profileDS.SearchProfiles(result.Ctx, profileQuery)
		if err != nil {
			return err
		}
		for _, p := range profiles {
			scanRefQuery := search.NewQueryBuilder().AddExactMatches(
				search.ComplianceOperatorProfileRef,
				p.GetProfileRefId(),
			).ProtoQuery()
			scans, err := w.scanDS.SearchScans(result.Ctx, scanRefQuery)
			if err != nil {
				log.Errorf("Unable to retrieve scans: %v", err)
				continue
			}
			for _, s := range scans {
				log.Debugf("Adding scan to wait %s", s.GetScanName())
				w.scansToWait.Add(fmt.Sprintf("%s:%s", s.GetClusterId(), s.GetId()))
			}
		}
		w.totalResults = len(w.scansToWait)
		log.Infof("Scan config %s needs to wait for %d scans", w.scanConfigResults.ScanConfig.GetScanConfigName(), w.totalResults)
	}
	log.Infof("Scan to handle %s with id %s", result.Scan.GetScanName(), result.Scan.GetId())
	if found := w.scansToWait.Remove(fmt.Sprintf("%s:%s", result.Scan.GetClusterId(), result.Scan.GetId())); !found {
		return errors.Errorf("The scan %s should be handle by this watcher", result.Scan.GetId())
	}
	w.scanConfigResults.ScanResults[fmt.Sprintf("%s:%s", result.Scan.GetClusterId(), result.Scan.GetId())] = result
	return nil
}
