package watcher

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	complianceIntegrationDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/timestamp"
	"golang.org/x/mod/semver"
)

var (
	log                                              = logging.LoggerForModule()
	ErrScanAlreadyHandled                            = errors.New("the scan is already handled")
	ErrScanTimeout                                   = errors.New("scan watcher timed out")
	ErrComplianceOperatorNotInstalled                = errors.New("compliance operator is not installed")
	ErrComplianceOperatorVersion                     = errors.New("compliance operator version")
	ErrComplianceOperatorIntegrationDataStore        = errors.New("unable to retrieve compliance operator integration")
	ErrComplianceOperatorIntegrationZeroIntegrations = errors.New("no compliance operator integrations retrieved")
	ErrComplianceOperatorScanMissingLastStartedFiled = errors.New("scan is missing the LastStartedTime field")
	ErrComplianceOperatorReceivedOldCheckResult      = errors.New("the check result is older than the current scan")
)

const (
	minimumComplianceOperatorVersion = "v1.6.0"
	checkCountAnnotationKey          = "compliance.openshift.io/check-count"
	lastScannedAnnotationKey         = "compliance.openshift.io/last-scanned-timestamp"
	defaultChanelSize                = 100
	defaultTimeout                   = 10 * time.Minute
)

// ScanWatcher determines if a scan is running or has completed.
type ScanWatcher interface {
	PushScan(v2 *storage.ComplianceOperatorScanV2) error
	PushCheckResult(*storage.ComplianceOperatorCheckResultV2) error
	Finished() concurrency.ReadOnlySignal
	Stop()
}

// ScanWatcherResults is returned when the watcher detects that the scan is completed.
type ScanWatcherResults struct {
	Ctx          context.Context
	WatcherID    string
	Scan         *storage.ComplianceOperatorScanV2
	CheckResults set.StringSet
	Error        error
}

// IsComplianceOperatorHealthy indicates whether Compliance Operator is ready for automatic reporting
func IsComplianceOperatorHealthy(ctx context.Context, clusterID string, complianceIntegrationDataStore complianceIntegrationDS.DataStore) error {
	coStatus, err := complianceIntegrationDataStore.GetComplianceIntegrationByCluster(ctx, clusterID)
	if err != nil {
		return errors.Wrap(err, ErrComplianceOperatorIntegrationDataStore.Error())
	}
	if len(coStatus) == 0 {
		log.Errorf("No compliance integrations retrieved from cluster %s", clusterID)
		return ErrComplianceOperatorIntegrationZeroIntegrations
	}
	if !coStatus[0].GetOperatorInstalled() {
		return ErrComplianceOperatorNotInstalled
	}
	if semver.Compare(coStatus[0].GetVersion(), minimumComplianceOperatorVersion) < 0 {
		return ErrComplianceOperatorVersion
	}
	return nil
}

// GetWatcherIDFromScan given a Scan, returns a unique ID for the watcher
func GetWatcherIDFromScan(ctx context.Context, scan *storage.ComplianceOperatorScanV2, snapshotDataStore snapshotDS.DataStore, overrideTimestamp *protocompat.Timestamp) (string, error) {
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
		// If we have reports with the same start time or newer we do not handle this scan
		// as it is an old scan, the check results in the db will not be the results of this scan
		// We can land in this situation if CO sends a scan update after the watcher is done, which
		// could happen because the check-count annotation that we use to determine if the scan is done
		// is sent before the scan's last update (when it changes its status to COMPLETED).
		return "", ErrScanAlreadyHandled
	}
	return fmt.Sprintf("%s:%s:%s", clusterID, scanID, startTime.String()), nil
}

// GetWatcherIDFromCheckResult given a CheckResult, returns a unique ID for the watcher
func GetWatcherIDFromCheckResult(ctx context.Context, result *storage.ComplianceOperatorCheckResultV2, scanDataStore scanDS.DataStore, snapshotDataStore snapshotDS.DataStore) (string, error) {
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
	if startTime, ok = result.GetAnnotations()[lastScannedAnnotationKey]; !ok {
		return "", errors.Errorf("%s annotation not found", lastScannedAnnotationKey)
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
		return GetWatcherIDFromScan(ctx, scans[0], snapshotDataStore, timestamp)
	}
	return GetWatcherIDFromScan(ctx, scans[0], snapshotDataStore, nil)
}

// readyQueue represents the expected queue interface to push the results
type readyQueue[T comparable] interface {
	Push(T)
}

type scanWatcherImpl struct {
	ctx     context.Context
	cancel  func()
	scanC   chan *storage.ComplianceOperatorScanV2
	resultC chan *storage.ComplianceOperatorCheckResultV2
	stopped *concurrency.Signal

	readyQueue  readyQueue[*ScanWatcherResults]
	scanResults *ScanWatcherResults
	totalChecks int
}

// NewScanWatcher creates a new ScanWatcher
func NewScanWatcher(ctx context.Context, watcherID string, queue readyQueue[*ScanWatcherResults]) *scanWatcherImpl {
	log.Infof("Creating new ScanWatcher with id %s", watcherID)
	watcherCtx, cancel := context.WithCancel(ctx)
	finishedSignal := concurrency.NewSignal()
	timeout := NewTimer(defaultTimeout)
	ret := &scanWatcherImpl{
		ctx:        watcherCtx,
		cancel:     cancel,
		scanC:      make(chan *storage.ComplianceOperatorScanV2, defaultChanelSize),
		resultC:    make(chan *storage.ComplianceOperatorCheckResultV2, defaultChanelSize),
		stopped:    &finishedSignal,
		readyQueue: queue,
		scanResults: &ScanWatcherResults{
			Ctx:          ctx,
			WatcherID:    watcherID,
			CheckResults: set.NewStringSet(),
		},
	}
	go ret.run(timeout)
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
func (s *scanWatcherImpl) Stop() {
	s.cancel()
}

func (s *scanWatcherImpl) run(timer Timer) {
	defer func() {
		s.stopped.Signal()
		<-s.stopped.Done()
		timer.Stop()
	}()
	for {
		select {
		case <-s.ctx.Done():
			log.Infof("Stopping scan watcher for scan")
			return
		case <-timer.C():
			log.Warnf("Timeout waiting for the scan %s to finish", s.scanResults.Scan.GetScanName())
			s.scanResults.Error = ErrScanTimeout
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
		if s.totalChecks != 0 && s.totalChecks == len(s.scanResults.CheckResults) {
			s.readyQueue.Push(s.scanResults)
			return
		}
	}
}

func (s *scanWatcherImpl) handleScan(scan *storage.ComplianceOperatorScanV2) error {
	if checkCountAnnotation, found := scan.GetAnnotations()[checkCountAnnotationKey]; found {
		var numChecks int
		var err error
		if numChecks, err = strconv.Atoi(checkCountAnnotation); err != nil {
			return errors.Wrap(err, "unable to convert the check count annotation to int")
		}
		s.totalChecks = numChecks
	}
	if s.scanResults.Scan == nil {
		s.scanResults.Scan = scan
	}
	return nil
}

func (s *scanWatcherImpl) handleResult(result *storage.ComplianceOperatorCheckResultV2) error {
	s.scanResults.CheckResults.Add(result.GetCheckId())
	return nil
}
