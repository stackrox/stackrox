package watcher

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	complianceIntegrationDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"golang.org/x/mod/semver"
)

var (
	log                                               = logging.LoggerForModule()
	dbAccess                                          = sac.WithAllAccess(context.Background())
	ScanTimeoutError                                  = errors.New("scan watcher timed out")
	ComplianceOperatorNotInstalledError               = errors.New("compliance operator is not installed")
	ComplianceOperatorVersionError                    = errors.New("compliance operator version")
	ComplianceOperatorIntegrationDataStoreError       = errors.New("unable to retrieve compliance operator integration")
	ComplianceOperatorIntegrationZeroIngerationsError = errors.New("no compliance operator integrations retrieved")
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
	ID           string
	Scan         *storage.ComplianceOperatorScanV2
	CheckResults set.StringSet
	Error        error
}

// IsComplianceOperatorHealthy indicates whether Compliance Operator is ready for automatic reporting
func IsComplianceOperatorHealthy(clusterID string, complianceIntegrationDataStore complianceIntegrationDS.DataStore) error {
	coStatus, err := complianceIntegrationDataStore.GetComplianceIntegrationByCluster(dbAccess, clusterID)
	if err != nil {
		return errors.Wrap(err, ComplianceOperatorIntegrationDataStoreError.Error())
	}
	if len(coStatus) == 0 {
		log.Errorf("No compliance integrations retrieved from cluster %s", clusterID)
		return ComplianceOperatorIntegrationZeroIngerationsError
	}
	if !coStatus[0].GetOperatorInstalled() {
		return ComplianceOperatorNotInstalledError
	}
	if semver.Compare(coStatus[0].GetVersion(), minimumComplianceOperatorVersion) < 0 {
		return ComplianceOperatorVersionError
	}
	return nil
}

// GetWatcherIDFromScan given a Scan, returns a unique ID for the watcher
func GetWatcherIDFromScan(scan *storage.ComplianceOperatorScanV2) (string, error) {
	clusterID := scan.GetClusterId()
	if clusterID == "" {
		return "", errors.New("Missing cluster ID")
	}
	scanID := scan.GetId()
	if scanID == "" {
		return "", errors.New("Missing scan ID")
	}
	startTime := scan.GetLastStartedTime()
	if startTime == nil {
		// This could happen if the scan is freshly created
		log.Debug("Missing last stated time")
		return "", nil
	}
	return fmt.Sprintf("%s:%s:%s", clusterID, scanID, startTime.String()), nil
}

// GetWatcherIDFromCheckResult given a CheckResult, returns a unique ID for the watcher
func GetWatcherIDFromCheckResult(result *storage.ComplianceOperatorCheckResultV2, scanDataStore scanDS.DataStore) (string, error) {
	scanRefQuery := search.NewQueryBuilder().AddExactMatches(
		search.ComplianceOperatorScanRef,
		result.GetScanRefId(),
	).ProtoQuery()
	scans, err := scanDataStore.SearchScans(dbAccess, scanRefQuery)
	if err != nil {
		return "", errors.Errorf("Unable to retrieve scan : %v", err)
	}
	if len(scans) == 0 {
		return "", errors.Errorf("Scan not found for result %s", result.GetCheckName())
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
	if timestamp.String() != scans[0].GetLastStartedTime().String() {
		return "", errors.Errorf("The result and the scan do not have the same timestamp")
	}
	return GetWatcherIDFromScan(scans[0])
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

	watcherID    string
	readyQueue   readyQueue[*ScanWatcherResults]
	scan         *storage.ComplianceOperatorScanV2
	checkResults set.StringSet
	totalChecks  int
}

// NewScanWatcher creates a new ScanWatcher
func NewScanWatcher(ctx context.Context, watcherID string, queue readyQueue[*ScanWatcherResults]) *scanWatcherImpl {
	watcherCtx, cancel := context.WithCancel(ctx)
	finishedSignal := concurrency.NewSignal()
	ret := &scanWatcherImpl{
		ctx:          watcherCtx,
		cancel:       cancel,
		scanC:        make(chan *storage.ComplianceOperatorScanV2, defaultChanelSize),
		resultC:      make(chan *storage.ComplianceOperatorCheckResultV2, defaultChanelSize),
		stopped:      &finishedSignal,
		watcherID:    watcherID,
		readyQueue:   queue,
		checkResults: set.NewStringSet(),
	}
	timeout := time.NewTimer(defaultTimeout)
	go ret.run(timeout.C)
	return ret
}

// PushScan queues a Scan to be handled
func (s *scanWatcherImpl) PushScan(scan *storage.ComplianceOperatorScanV2) error {
	select {
	case <-s.ctx.Done():
		return errors.New("The watcher is stopped")
	case s.scanC <- scan:
		return nil
	}
}

// PushCheckResult queues a CheckResult to be handled
func (s *scanWatcherImpl) PushCheckResult(result *storage.ComplianceOperatorCheckResultV2) error {
	select {
	case <-s.ctx.Done():
		return errors.New("The watcher is stopped")
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

func (s *scanWatcherImpl) run(timerC <-chan time.Time) {
	defer func() {
		s.stopped.Signal()
		<-s.stopped.Done()
	}()
	for {
		select {
		case <-s.ctx.Done():
			log.Infof("Stopping scan watcher for scan")
			return
		case <-timerC:
			log.Warnf("Timeout waiting for the scan %s to finish", s.scan.GetScanName())
			s.readyQueue.Push(&ScanWatcherResults{
				ID:           s.watcherID,
				Scan:         s.scan,
				CheckResults: s.checkResults,
				Error:        ScanTimeoutError,
			})
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
		if s.totalChecks != 0 && s.totalChecks == len(s.checkResults) {
			s.readyQueue.Push(&ScanWatcherResults{
				ID:           s.watcherID,
				Scan:         s.scan,
				CheckResults: s.checkResults,
			})
			return
		}
	}
}

func (s *scanWatcherImpl) handleScan(scan *storage.ComplianceOperatorScanV2) error {
	if checkCountAnnotation, found := scan.GetAnnotations()[checkCountAnnotationKey]; found {
		var numChecks int
		var err error
		if numChecks, err = strconv.Atoi(checkCountAnnotation); err != nil {
			return err
		}
		s.totalChecks = numChecks
	}
	if s.scan == nil {
		s.scan = scan
	}
	return nil
}

func (s *scanWatcherImpl) handleResult(result *storage.ComplianceOperatorCheckResultV2) error {
	s.checkResults.Add(result.GetCheckId())
	return nil
}
