package watcher

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	complianceIntegrationDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	scanConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"golang.org/x/mod/semver"
)

var (
	log                                              = logging.LoggerForModule()
	ErrScanAlreadyHandled                            = errors.New("the scan is already handled")
	ErrScanTimeout                                   = errors.New("scan watcher timed out")
	ErrScanContextCancelled                          = errors.New("scan watcher context cancelled")
	ErrScanRemoved                                   = errors.New("scan was removed")
	ErrComplianceOperatorNotInstalled                = errors.New("compliance operator is not installed")
	ErrComplianceOperatorVersion                     = errors.New("compliance operator version")
	ErrComplianceOperatorIntegrationDataStore        = errors.New("unable to retrieve compliance operator integration")
	ErrComplianceOperatorIntegrationZeroIntegrations = errors.New("no compliance operator integrations retrieved")
	ErrComplianceOperatorScanMissingLastStartedFiled = errors.New("scan is missing the LastStartedTime field")
	ErrComplianceOperatorReceivedOldCheckResult      = errors.New("the check result is older than the current scan")
)

const (
	minimumComplianceOperatorVersion = "v1.6.0"
	CheckCountAnnotationKey          = "compliance.openshift.io/check-count"
	LastScannedAnnotationKey         = "compliance.openshift.io/last-scanned-timestamp"
	defaultChanelSize                = 100
)

// ScanWatcher determines if a scan is running or has completed.
type ScanWatcher interface {
	PushScan(v2 *storage.ComplianceOperatorScanV2) error
	PushCheckResult(*storage.ComplianceOperatorCheckResultV2) error
	Finished() concurrency.ReadOnlySignal
	Stop(err error)
}

// ScanWatcherResults is returned when the watcher detects that the scan is completed.
type ScanWatcherResults struct {
	SensorCtx    context.Context
	WatcherID    string
	Scan         *storage.ComplianceOperatorScanV2
	CheckResults set.StringSet
	Error        error
}

// IsComplianceOperatorHealthy indicates whether Compliance Operator is ready for automatic reporting
func IsComplianceOperatorHealthy(ctx context.Context, clusterID string, complianceIntegrationDataStore complianceIntegrationDS.DataStore) (*storage.ComplianceIntegration, error) {
	coStatus, err := complianceIntegrationDataStore.GetComplianceIntegrationByCluster(ctx, clusterID)
	if err != nil {
		return nil, errors.Wrap(ErrComplianceOperatorIntegrationDataStore, err.Error())
	}
	if len(coStatus) == 0 {
		log.Errorf("No compliance integrations retrieved from cluster %s", clusterID)
		return nil, ErrComplianceOperatorIntegrationZeroIntegrations
	}
	if !coStatus[0].GetOperatorInstalled() {
		return coStatus[0], ErrComplianceOperatorNotInstalled
	}
	if semver.Compare(coStatus[0].GetVersion(), minimumComplianceOperatorVersion) < 0 {
		return coStatus[0], ErrComplianceOperatorVersion
	}
	return coStatus[0], nil
}

// GetWatcherIDFromScan given a Scan, returns a unique ID for the watcher
func GetWatcherIDFromScan(ctx context.Context, scan *storage.ComplianceOperatorScanV2, snapshotDataStore snapshotDS.DataStore, scanConfigDataStore scanConfigDS.DataStore, overrideTimestamp *protocompat.Timestamp) (string, error) {
	if scan == nil {
		return "", errors.New("nil scan")
	}
	clusterID := scan.GetClusterId()
	if clusterID == "" {
		return "", errors.New("missing cluster ID")
	}
	scanID := scan.GetId()
	if scanID == "" {
		return "", errors.New("missing scan ID")
	}
	startTime := scan.GetLastStartedTime()
	if startTime == nil && overrideTimestamp == nil {
		// This could happen if the scan is freshly created
		return "", ErrComplianceOperatorScanMissingLastStartedFiled
	}
	if overrideTimestamp != nil {
		startTime = overrideTimestamp
	}
	scanConfigQuery := search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfigName, scan.GetScanConfigName()).
		ProtoQuery()
	scanConfigs, err := scanConfigDataStore.GetScanConfigurations(ctx, scanConfigQuery)
	if err != nil {
		return "", errors.Wrap(err, "unable to retrieve the scan configuration from the store")
	}
	if len(scanConfigs) == 0 {
		return "", errors.New("this scan is not handled by any known scan configuration")
	}
	// If there is a snapshot with the same timestamp or newer we shouldn't handle this scan since we already handle a newer one
	query := search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanRef, scan.GetScanRefId()).
		AddTimeRangeField(search.ComplianceOperatorScanLastStartedTime, startTime.AsTime(), timestamp.InfiniteFuture.GoTime()).
		ProtoQuery()
	snapshots, err := snapshotDataStore.SearchSnapshots(ctx, query)
	if err != nil {
		return "", errors.Wrap(err, "unable to retrieve snapshots from the store")
	}
	if len(snapshots) > 0 {
		// If we have reports with the same start time or newer, then we do not handle this scan
		// as it is an old scan, because the check results in the db will not be the results of this scan.
		// We can land in this situation if CO sends a scan update after the watcher is done, which
		// could happen because the check-count annotation that we use to determine if the scan is done
		// is sent before the scan's last update (when it changes its status to COMPLETED).
		return "", ErrScanAlreadyHandled
	}
	return fmt.Sprintf("%s:%s", clusterID, scanID), nil
}

// GetWatcherIDFromCheckResult given a CheckResult, returns a unique ID for the watcher
func GetWatcherIDFromCheckResult(ctx context.Context, result *storage.ComplianceOperatorCheckResultV2, scanDataStore scanDS.DataStore, snapshotDataStore snapshotDS.DataStore, scanConfigDataStore scanConfigDS.DataStore) (string, error) {
	if result == nil {
		return "", errors.New("nil check result")
	}
	scanRefQuery := search.NewQueryBuilder().AddExactMatches(
		search.ComplianceOperatorScanRef,
		result.GetScanRefId(),
	).ProtoQuery()
	scans, err := scanDataStore.SearchScans(ctx, scanRefQuery)
	if err != nil {
		return "", errors.Wrap(err, "unable to retrieve scan")
	}
	// We should always receive one scan here since ComplianceOperatorScanRef should be unique
	if len(scans) != 1 {
		return "", errors.Errorf("scan not found for result %s", result.GetCheckName())
	}
	var startTime string
	var ok bool
	if startTime, ok = result.GetAnnotations()[LastScannedAnnotationKey]; !ok {
		return "", errors.Errorf("%s annotation not found", LastScannedAnnotationKey)
	}
	timestamp, err := protocompat.ParseRFC3339NanoTimestamp(startTime)
	if err != nil {
		return "", errors.Errorf("Unable to parse time: %v", err)
	}
	timestampCmpResult := protocompat.CompareTimestamps(timestamp, scans[0].GetLastStartedTime())
	if timestampCmpResult < 0 {
		return "", ErrComplianceOperatorReceivedOldCheckResult
	}
	if timestampCmpResult > 0 {
		// In this case the timestamp from the check is newer which means the scans has not yet arrived
		// to sensor's pipeline. We need to create the watcher with the new timestamp
		return GetWatcherIDFromScan(ctx, scans[0], snapshotDataStore, scanConfigDataStore, timestamp)
	}
	return GetWatcherIDFromScan(ctx, scans[0], snapshotDataStore, scanConfigDataStore, nil)
}

// readyQueue represents the expected queue interface to push the results
type readyQueue[T comparable] interface {
	Push(T)
}

type scanWatcherImpl struct {
	ctx       context.Context
	sensorCtx context.Context
	cancel    func()
	timeout   Timer
	scanC     chan *storage.ComplianceOperatorScanV2
	resultC   chan *storage.ComplianceOperatorCheckResultV2
	stopped   *concurrency.Signal

	readyQueue      readyQueue[*ScanWatcherResults]
	resultsLock     sync.Mutex
	scanResults     *ScanWatcherResults
	totalChecks     int
	lastStartedTime *protocompat.Timestamp
}

// NewScanWatcher creates a new ScanWatcher
func NewScanWatcher(ctx, sensorCtx context.Context, watcherID string, queue readyQueue[*ScanWatcherResults]) *scanWatcherImpl {
	log.Debugf("Creating new ScanWatcher with id %s", watcherID)
	watcherCtx, cancel := context.WithCancel(ctx)
	finishedSignal := concurrency.NewSignal()
	timeout := NewTimer(env.ComplianceScanWatcherTimeout.DurationSetting())
	ret := &scanWatcherImpl{
		ctx:        watcherCtx,
		sensorCtx:  sensorCtx,
		cancel:     cancel,
		timeout:    timeout,
		scanC:      make(chan *storage.ComplianceOperatorScanV2, defaultChanelSize),
		resultC:    make(chan *storage.ComplianceOperatorCheckResultV2, defaultChanelSize),
		stopped:    &finishedSignal,
		readyQueue: queue,
		scanResults: &ScanWatcherResults{
			SensorCtx:    sensorCtx,
			WatcherID:    watcherID,
			CheckResults: set.NewStringSet(),
		},
	}
	go ret.run()
	return ret
}

// PushScan queues a Scan to be handled
func (s *scanWatcherImpl) PushScan(scan *storage.ComplianceOperatorScanV2) error {
	select {
	case <-s.ctx.Done():
		return errors.New("the watcher is stopped")
	case s.scanC <- scan:
		return nil
	}
}

// PushCheckResult queues a CheckResult to be handled
func (s *scanWatcherImpl) PushCheckResult(result *storage.ComplianceOperatorCheckResultV2) error {
	select {
	case <-s.ctx.Done():
		return errors.New("the watcher is stopped")
	case s.resultC <- result:
		return nil
	}
}

// Finished indicates whether the watcher is finished or not
func (s *scanWatcherImpl) Finished() concurrency.ReadOnlySignal {
	return s.stopped
}

// Stop the watcher
func (s *scanWatcherImpl) Stop(err error) {
	if err != nil {
		concurrency.WithLock(&s.resultsLock, func() {
			s.scanResults.Error = err
		})
	}
	s.cancel()
}

func (s *scanWatcherImpl) run() {
	defer func() {
		s.stopped.Signal()
		<-s.stopped.Done()
		s.timeout.Stop()
	}()
	for {
		select {
		case <-s.ctx.Done():
			concurrency.WithLock(&s.resultsLock, func() {
				log.Infof("Stopping scan watcher for scan %s", s.scanResults.Scan.GetScanName())
				if s.scanResults.Error == nil {
					s.scanResults.Error = ErrScanContextCancelled
				}
			})
			s.readyQueue.Push(s.scanResults)
			return
		case <-s.timeout.C():
			concurrency.WithLock(&s.resultsLock, func() {
				log.Warnf("Timeout waiting for the scan %s to finish", s.scanResults.Scan.GetScanName())
				s.scanResults.Error = ErrScanTimeout
			})
			s.readyQueue.Push(s.scanResults)
			return
		case scan := <-s.scanC:
			if err := s.handleScan(scan); err != nil {
				log.Errorf("Unable to handle scan %s: %v", scan.GetScanName(), err)
			}
		case result := <-s.resultC:
			if err := s.handleResult(result); err != nil {
				log.Errorf("Unable to handle result %s for scan %s: %v", result.GetCheckName(), result.GetScanName(), err)
			}
		}
		var numCheckResults int
		concurrency.WithLock(&s.resultsLock, func() {
			numCheckResults = len(s.scanResults.CheckResults)
		})
		if s.totalChecks != 0 && s.totalChecks == numCheckResults {
			s.readyQueue.Push(s.scanResults)
			return
		}
	}
}

func (s *scanWatcherImpl) handleScan(scan *storage.ComplianceOperatorScanV2) error {
	if checkCountAnnotation, found := scan.GetAnnotations()[CheckCountAnnotationKey]; found {
		var numChecks int
		var err error
		if numChecks, err = strconv.Atoi(checkCountAnnotation); err != nil {
			return errors.Wrap(err, "unable to convert the check count annotation to int")
		}
		s.totalChecks = numChecks
	}

	s.resultsLock.Lock()
	defer s.resultsLock.Unlock()

	if s.scanResults.Scan == nil {
		s.scanResults.Scan = scan
	}
	// If we received a newer timestamp we need to reset the watcher.
	if protocompat.CompareTimestamps(s.lastStartedTime, scan.GetLastStartedTime()) < 0 {
		s.lastStartedTime = scan.GetLastStartedTime()
		s.scanResults.Scan = scan
		s.scanResults.CheckResults = set.NewStringSet()
		s.timeout.Reset()
	}
	return nil
}

func (s *scanWatcherImpl) handleResult(result *storage.ComplianceOperatorCheckResultV2) error {
	var startTime string
	var found bool
	if startTime, found = result.GetAnnotations()[LastScannedAnnotationKey]; !found {
		return errors.Errorf("check result does not have the %s annotation", LastScannedAnnotationKey)
	}
	timestamp, err := protocompat.ParseRFC3339NanoTimestamp(startTime)
	if err != nil {
		return errors.Errorf("unable to parse time: %v", err)
	}

	s.resultsLock.Lock()
	defer s.resultsLock.Unlock()

	timestampCmpResult := protocompat.CompareTimestamps(timestamp, s.lastStartedTime)
	if timestampCmpResult < 0 {
		// This is an old CheckResult, so we do not processed it.
		// This could happen if a scan is reset mid-execution and there are old
		// CheckResult still in sensor's pipeline.
		return ErrComplianceOperatorReceivedOldCheckResult
	}
	// We received a CheckResult with a newer timestamp.
	// This means that a new scan is about to be received, so we reset the watcher.
	// This should never happen and is only here as a sanity check in case
	// there is a race in sensor's pipelines.
	if timestampCmpResult > 0 {
		s.lastStartedTime = timestamp
		s.scanResults.CheckResults = set.NewStringSet()
		s.timeout.Reset()
	}
	s.scanResults.CheckResults.Add(result.GetCheckId())
	return nil
}
