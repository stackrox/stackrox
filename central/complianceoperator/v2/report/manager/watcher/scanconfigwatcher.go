package watcher

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scan "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
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
	Subscribe(snapshotID string) error
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

	scanDS     scan.DataStore
	profileDS  profileDatastore.DataStore
	snapshotDS snapshotDS.DataStore

	resultsLock       sync.Mutex
	readyQueue        readyQueue[*ScanConfigWatcherResults]
	scanConfigResults *ScanConfigWatcherResults
	scansToWait       set.StringSet
	totalResults      int
}

// NewScanConfigWatcher creates a new ScanConfigWatcher
func NewScanConfigWatcher(ctx context.Context, watcherID string, sc *storage.ComplianceOperatorScanConfigurationV2, scanDS scan.DataStore, profileDS profileDatastore.DataStore, snapshotDS snapshotDS.DataStore, queue readyQueue[*ScanConfigWatcherResults], snapshotIDs ...string) *scanConfigWatcherImpl {
	watcherCtx, cancel := context.WithCancel(ctx)
	finishedSignal := concurrency.NewSignal()
	timeout := NewTimer(defaultScanConfigTimeout)
	ret := &scanConfigWatcherImpl{
		ctx:        watcherCtx,
		cancel:     cancel,
		stopped:    &finishedSignal,
		scanDS:     scanDS,
		profileDS:  profileDS,
		snapshotDS: snapshotDS,
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
	go ret.run(timeout)
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

// Subscribe snapshot to the watcher
func (w *scanConfigWatcherImpl) Subscribe(id string) error {
	var ctx context.Context
	concurrency.WithLock(&w.resultsLock, func() {
		if w.scanConfigResults == nil {
			return
		}
		ctx = w.scanConfigResults.Ctx
	})
	snapshot, found, err := w.snapshotDS.GetSnapshot(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return errors.Errorf("snapshot %s not found", id)
	}
	var scans []*storage.ComplianceOperatorReportSnapshotV2_Scan
	concurrency.WithLock(&w.resultsLock, func() {
		w.scanConfigResults.ReportSnapshotIDs = append(w.scanConfigResults.ReportSnapshotIDs, id)
		for _, scanResult := range w.scanConfigResults.ScanResults {
			scans = append(scans, &storage.ComplianceOperatorReportSnapshotV2_Scan{
				ScanRefId:       scanResult.Scan.GetScanRefId(),
				LastStartedTime: scanResult.Scan.GetLastStartedTime(),
			})
		}
	})
	if len(scans) == 0 {
		return nil
	}
	snapshot.Scans = scans
	return w.snapshotDS.UpsertSnapshot(ctx, snapshot)
}

// Stop the watcher
func (w *scanConfigWatcherImpl) Stop() {
	w.cancel()
}

// Finished indicates whether the watcher is finished or not
func (w *scanConfigWatcherImpl) Finished() concurrency.ReadOnlySignal {
	return w.stopped
}

func (w *scanConfigWatcherImpl) run(timer Timer) {
	defer func() {
		w.stopped.Signal()
		<-w.stopped.Done()
		timer.Stop()
	}()
	for {
		select {
		case <-w.ctx.Done():
			log.Infof("Stopping scan config watcher")
			return
		case <-timer.C():
			concurrency.WithLock(&w.resultsLock, func() {
				log.Warnf("Timeout waiting for the ScanConfiguration %s's scans to finish", w.scanConfigResults.ScanConfig.GetScanConfigName())
				w.scanConfigResults.Error = ScanConfigTimeoutError
				w.readyQueue.Push(w.scanConfigResults)
			})
			return
		case result := <-w.scanC:
			if err := w.handleScanResults(result); err != nil {
				log.Errorf("Unable to handle scan results %s: %v", result.Scan.GetScanName(), err)
				concurrency.WithLock(&w.resultsLock, func() {
					w.scanConfigResults.Error = err
					w.readyQueue.Push(w.scanConfigResults)
				})
				return
			}
		}
		if concurrency.WithLock1[bool](&w.resultsLock, func() bool {
			if w.totalResults != 0 && w.totalResults == len(w.scanConfigResults.ScanResults) {
				w.readyQueue.Push(w.scanConfigResults)
				return true
			}
			return false
		}) {
			return
		}
	}
}

func (w *scanConfigWatcherImpl) handleScanResults(result *ScanWatcherResults) error {
	// Here we have the scan config id and the scan
	if w.totalResults == 0 {
		scans, err := GetScansFromScanConfiguration(w.scanConfigResults.Ctx, w.scanConfigResults.ScanConfig, w.profileDS, w.scanDS)
		if err != nil {
			return err
		}
		w.scansToWait = scans
		w.totalResults = len(w.scansToWait)
		log.Infof("Scan config %s needs to wait for %d scans", w.scanConfigResults.ScanConfig.GetScanConfigName(), w.totalResults)
	}
	log.Infof("Scan to handle %s with id %s", result.Scan.GetScanName(), result.Scan.GetId())
	if found := w.scansToWait.Remove(fmt.Sprintf("%s:%s", result.Scan.GetClusterId(), result.Scan.GetId())); !found {
		return errors.Errorf("The scan %s should be handle by this watcher", result.Scan.GetId())
	}
	concurrency.WithLock(&w.resultsLock, func() {
		w.scanConfigResults.ScanResults[fmt.Sprintf("%s:%s", result.Scan.GetClusterId(), result.Scan.GetId())] = result
	})

	return w.appendScanToSnapshots(result.Ctx, result.Scan)
}

func (w *scanConfigWatcherImpl) appendScanToSnapshots(ctx context.Context, scan *storage.ComplianceOperatorScanV2) error {
	errList := errorhelpers.NewErrorList("for each snapshot")
	concurrency.WithLock(&w.resultsLock, func() {
		for _, id := range w.scanConfigResults.ReportSnapshotIDs {
			snapshot, found, err := w.snapshotDS.GetSnapshot(ctx, id)
			if err != nil {
				errList.AddError(err)
				continue
			}
			if !found {
				errList.AddError(errors.Errorf("snapshot %s not found", id))
				continue
			}
			snapshot.Scans = append(snapshot.Scans, &storage.ComplianceOperatorReportSnapshotV2_Scan{
				ScanRefId:       scan.GetScanRefId(),
				LastStartedTime: scan.GetLastStartedTime(),
			})
			if err := w.snapshotDS.UpsertSnapshot(ctx, snapshot); err != nil {
				errList.AddError(err)
			}
		}
	})
	return errList.ToError()
}

// GetScansFromScanConfiguration returns the scans associated with a given ScanConfiguration
func GetScansFromScanConfiguration(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2, profileDataStore profileDatastore.DataStore, scanDataStore scan.DataStore) (set.StringSet, error) {
	ret := set.NewStringSet()
	var profileNames []string
	for _, p := range scanConfig.GetProfiles() {
		profileNames = append(profileNames, p.GetProfileName())
	}
	var clusters []string
	for _, c := range scanConfig.GetClusters() {
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
	profiles, err := profileDataStore.SearchProfiles(ctx, profileQuery)
	if err != nil {
		return nil, errors.Wrap(err, "unable to search the profiles")
	}
	for _, p := range profiles {
		scanRefQuery := search.NewQueryBuilder().AddExactMatches(
			search.ComplianceOperatorProfileRef,
			p.GetProfileRefId(),
		).ProtoQuery()
		scans, err := scanDataStore.SearchScans(ctx, scanRefQuery)
		if err != nil {
			return nil, errors.Wrap(err, "unable to search the scans")
		}
		for _, s := range scans {
			log.Debugf("Adding scan to wait %s", s.GetScanName())
			ret.Add(fmt.Sprintf("%s:%s", s.GetClusterId(), s.GetId()))
		}
	}
	return ret, nil
}
