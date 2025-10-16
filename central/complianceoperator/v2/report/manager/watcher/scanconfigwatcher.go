package watcher

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	resultsDataStore "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	snapshotDataStore "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scanDataStore "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	ErrScanConfigTimeout          = errors.New("scan config watcher timed out")
	ErrScanConfigContextCancelled = errors.New("scan config watcher context cancelled")
)

// GetScanConfigFromScan returns the ScanConfiguration associated with the given scan
func GetScanConfigFromScan(ctx context.Context, scan *storage.ComplianceOperatorScanV2, scanConfigDS scanConfigurationDS.DataStore) (*storage.ComplianceOperatorScanConfigurationV2, error) {
	return scanConfigDS.GetScanConfigurationByName(ctx, scan.GetScanConfigName())
}

func DeleteOldResultsFromMissingScans(ctx context.Context, results *ScanConfigWatcherResults, profileDataStore profileDatastore.DataStore, scanDataStore scanDataStore.DataStore, resultsDataStore resultsDataStore.DataStore) error {
	if results == nil {
		return errors.New("unable to delete old CheckResults from an nil ScanConfigWatcherResults")
	}
	scans, err := GetScansFromScanConfiguration(ctx, results.ScanConfig, profileDataStore, scanDataStore)
	if err != nil {
		return err
	}
	for _, scanResults := range results.ScanResults {
		scans.Remove(fmt.Sprintf("%s:%s", scanResults.Scan.GetClusterId(), scanResults.Scan.GetId()))
	}
	errList := errorhelpers.NewErrorList("delete old CheckResults from missing scans")
	for scanWatcherID := range scans {
		parts := strings.Split(scanWatcherID, ":")
		if len(parts) != 2 {
			errList.AddError(errors.Errorf("unable to parse ScanID from %q", scanWatcherID))
			continue
		}
		scan, found, err := scanDataStore.GetScan(ctx, parts[1])
		if err != nil {
			errList.AddError(err)
			continue
		}
		if !found {
			errList.AddError(errors.Errorf("unable to find Scan with ID %q", parts[1]))
			continue
		}
		if err := resultsDataStore.DeleteOldResults(ctx, scan.GetLastStartedTime(), scan.GetScanRefId(), true); err != nil {
			errList.AddError(err)
		}
	}
	return errList.ToError()
}

// ScanConfigWatcher determines if a ScanConfiguration has running scans or has completed
type ScanConfigWatcher interface {
	PushScanResults(results *ScanWatcherResults) error
	Subscribe(snapshot *storage.ComplianceOperatorReportSnapshotV2) error
	GetScans() []*storage.ComplianceOperatorReportSnapshotV2_Scan
	Stop()
	Finished() concurrency.ReadOnlySignal
}

// ScanConfigWatcherResults is returned when the watcher detects all the scans are completed
type ScanConfigWatcherResults struct {
	SensorCtx      context.Context
	WatcherID      string
	ReportSnapshot []*storage.ComplianceOperatorReportSnapshotV2
	ScanConfig     *storage.ComplianceOperatorScanConfigurationV2
	ScanResults    map[string]*ScanWatcherResults
	Error          error
}

type scanConfigWatcherImpl struct {
	ctx                 context.Context
	sensorCtx           context.Context
	cancel              func()
	scanWatcherResoutsC chan *ScanWatcherResults
	stopped             *concurrency.Signal

	scanDS     scanDataStore.DataStore
	profileDS  profileDatastore.DataStore
	snapshotDS snapshotDataStore.DataStore

	resultsLock       sync.Mutex
	readyQueue        readyQueue[*ScanConfigWatcherResults]
	scanConfigResults *ScanConfigWatcherResults
	scansToWait       set.StringSet
	totalResults      int
}

// NewScanConfigWatcher creates a new ScanConfigWatcher
func NewScanConfigWatcher(ctx, sensorCtx context.Context, watcherID string, sc *storage.ComplianceOperatorScanConfigurationV2, scanDS scanDataStore.DataStore, profileDS profileDatastore.DataStore, snapshotDS snapshotDataStore.DataStore, queue readyQueue[*ScanConfigWatcherResults]) *scanConfigWatcherImpl {
	watcherCtx, cancel := context.WithCancel(ctx)
	finishedSignal := concurrency.NewSignal()
	timeout := NewTimer(env.ComplianceScanScheduleWatcherTimeout.DurationSetting())
	ret := &scanConfigWatcherImpl{
		ctx:                 watcherCtx,
		sensorCtx:           sensorCtx,
		cancel:              cancel,
		stopped:             &finishedSignal,
		scanDS:              scanDS,
		profileDS:           profileDS,
		snapshotDS:          snapshotDS,
		scanWatcherResoutsC: make(chan *ScanWatcherResults),
		readyQueue:          queue,
		scanConfigResults: &ScanConfigWatcherResults{
			SensorCtx:   sensorCtx,
			WatcherID:   watcherID,
			ScanConfig:  sc,
			ScanResults: make(map[string]*ScanWatcherResults),
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
	case w.scanWatcherResoutsC <- results:
		return nil
	}
}

// Subscribe snapshot to the watcher. A subscribed snapshot is a snapshots that
// needs to be updated from 'WAITING' to 'PREPARING' once the ScanConfigWatcher finishes.
func (w *scanConfigWatcherImpl) Subscribe(snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
	if w.scanConfigResults == nil {
		return errors.New("the scan config results are nil")
	}
	concurrency.WithLock(&w.resultsLock, func() {
		// Here we subscribe the snapshot to the watcher
		w.scanConfigResults.ReportSnapshot = append(w.scanConfigResults.ReportSnapshot, snapshot)
	})
	return nil
}

// GetScans returns all the scans that are handled by the watcher
func (w *scanConfigWatcherImpl) GetScans() []*storage.ComplianceOperatorReportSnapshotV2_Scan {
	var scans []*storage.ComplianceOperatorReportSnapshotV2_Scan
	if w.scanConfigResults == nil {
		return scans
	}
	concurrency.WithLock(&w.resultsLock, func() {
		// We return the current scans to be appended to the snapshot
		for _, scanResult := range w.scanConfigResults.ScanResults {
			cs := &storage.ComplianceOperatorReportSnapshotV2_Scan{}
			cs.SetScanRefId(scanResult.Scan.GetScanRefId())
			cs.SetLastStartedTime(scanResult.Scan.GetLastStartedTime())
			scans = append(scans, cs)
		}
	})
	return scans
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
			concurrency.WithLock(&w.resultsLock, func() {
				w.scanConfigResults.Error = ErrScanConfigContextCancelled
				w.readyQueue.Push(w.scanConfigResults)
			})
			return
		case <-timer.C():
			concurrency.WithLock(&w.resultsLock, func() {
				log.Warnf("Timeout waiting for the ScanConfiguration %s's scans to finish", w.scanConfigResults.ScanConfig.GetScanConfigName())
				w.scanConfigResults.Error = ErrScanConfigTimeout
				w.readyQueue.Push(w.scanConfigResults)
			})
			return
		case result := <-w.scanWatcherResoutsC:
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
		scans, err := GetScansFromScanConfiguration(w.ctx, w.scanConfigResults.ScanConfig, w.profileDS, w.scanDS)
		if err != nil {
			return err
		}
		w.scansToWait = scans
		w.totalResults = len(w.scansToWait)
		log.Debugf("Scan config %s needs to wait for %d scans", w.scanConfigResults.ScanConfig.GetScanConfigName(), w.totalResults)
	}
	log.Debugf("Scan to handle %s with id %s", result.Scan.GetScanName(), result.Scan.GetId())
	scanResultKey := fmt.Sprintf("%s:%s", result.Scan.GetClusterId(), result.Scan.GetId())
	if found := w.scansToWait.Remove(fmt.Sprintf("%s:%s", result.Scan.GetClusterId(), result.Scan.GetId())); !found {
		newScanResult := result.Scan
		var timestampCmpResult int
		concurrency.WithLock(&w.resultsLock, func() {
			if prevScanResult, ok := w.scanConfigResults.ScanResults[scanResultKey]; ok {
				timestampCmpResult = protocompat.CompareTimestamps(prevScanResult.Scan.GetLastStartedTime(), newScanResult.GetLastStartedTime())
			}
		})
		if timestampCmpResult > 0 {
			// We already handled a newer scan, so we can ignore this scan.
			return nil
		}
	}
	concurrency.WithLock(&w.resultsLock, func() {
		w.scanConfigResults.ScanResults[scanResultKey] = result
	})

	return w.appendScanToSnapshots(w.ctx, result.Scan)
}

func (w *scanConfigWatcherImpl) appendScanToSnapshots(ctx context.Context, scan *storage.ComplianceOperatorScanV2) error {
	errList := errorhelpers.NewErrorList("update snapshots' scans")
	concurrency.WithLock(&w.resultsLock, func() {
		for _, snapshot := range w.scanConfigResults.ReportSnapshot {
			// The new Scan is appended to the Snapshot. This allows us later to make sure
			// we do not generate duplicate Report Snapshots if we received an update in the Scan
			cs := &storage.ComplianceOperatorReportSnapshotV2_Scan{}
			cs.SetScanRefId(scan.GetScanRefId())
			cs.SetLastStartedTime(scan.GetLastStartedTime())
			snapshot.SetScans(append(snapshot.GetScans(), cs))
			if err := w.snapshotDS.UpsertSnapshot(ctx, snapshot); err != nil {
				errList.AddError(err)
			}
		}
	})
	return errList.ToError()
}

// GetScansFromScanConfiguration returns the scans associated with a given ScanConfiguration
func GetScansFromScanConfiguration(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2, profileDataStore profileDatastore.DataStore, scanDataStore scanDataStore.DataStore) (set.StringSet, error) {
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
