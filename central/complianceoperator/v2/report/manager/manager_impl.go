package manager

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	reportGen "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/complianceReportgenerator"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/watcher"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/queue"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/sync/semaphore"
)

var (
	log         = logging.LoggerForModule()
	maxRequests = 100
)

type reportRequest struct {
	scanConfig *storage.ComplianceOperatorScanConfigurationV2
	ctx        context.Context
}

type managerImpl struct {
	scanConfigDataStore scanConfigurationDS.DataStore
	scanDataStore       scanDS.DataStore

	runningReportConfigs map[string]*reportRequest
	// channel for report job requests
	reportRequests chan *reportRequest
	stopper        concurrency.Stopper
	// isStarted will make sure only one start routine runs for an instance of manager
	isStarted atomic.Bool

	// isStopped will prevent manager from being re-started once it is stopped
	isStopped atomic.Bool

	// Mutex to synchronize access to runningReportConfigs map
	mu             sync.Mutex
	concurrencySem *semaphore.Weighted
	reportGen      reportGen.ComplianceReportGenerator

	watchingScansLock sync.Mutex
	// watchingScans a map holding the ScanWatchers
	watchingScans map[string]watcher.ScanWatcher
	// readyQueue holds the scan that are ready to be reported
	readyQueue *queue.Queue[*watcher.ScanWatcherResults]
}

func New(scanConfigDS scanConfigurationDS.DataStore, scanDataStore scanDS.DataStore, reportGen reportGen.ComplianceReportGenerator) Manager {
	return &managerImpl{
		scanConfigDataStore:  scanConfigDS,
		scanDataStore:        scanDataStore,
		stopper:              concurrency.NewStopper(),
		runningReportConfigs: make(map[string]*reportRequest, maxRequests),
		reportRequests:       make(chan *reportRequest, maxRequests),
		concurrencySem:       semaphore.NewWeighted(int64(env.ReportExecutionMaxConcurrency.IntegerSetting())),
		reportGen:            reportGen,
		watchingScans:        make(map[string]watcher.ScanWatcher),
		readyQueue:           queue.NewQueue[*watcher.ScanWatcherResults](),
	}
}

func (m *managerImpl) SubmitReportRequest(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2) error {
	if !features.ComplianceReporting.Enabled() {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.runningReportConfigs[scanConfig.GetId()]; ok {
		return errors.New(fmt.Sprintf("Report request for scan configuration %q already in process", scanConfig.GetScanConfigName()))
	}

	req := &reportRequest{
		scanConfig: scanConfig,
		ctx:        context.WithoutCancel(ctx),
	}
	log.Infof("Submitting report for scan config %s at %v for execution with req %v.", scanConfig.GetScanConfigName(), time.Now().Format(time.RFC822), *req)
	select {
	case m.reportRequests <- req:
		m.runningReportConfigs[scanConfig.GetId()] = req
	default:
		return errors.New(fmt.Sprintf("Error submitting report request for for scan configuration %q, request limit reached",
			scanConfig.GetScanConfigName()))
	}

	return nil
}

func (m *managerImpl) Start() {
	if m.isStopped.Load() {
		log.Error("Compliance report manager already stopped. It cannot be re-started once stopped.")
		return
	}
	swapped := m.isStarted.CompareAndSwap(false, true)
	if !swapped {
		log.Error("Compliance report manager already running")
		return
	}
	log.Info("Starting compliance report manager")
	go m.runReports()
	go m.handleReadyScan()
}

func (m *managerImpl) Stop() {
	if m.isStarted.Load() {
		log.Error("Compliance report manager not started")
		return
	}
	swapped := m.isStopped.CompareAndSwap(false, true)
	if !swapped {
		log.Error("Compliance report manager already stopped")
		return
	}
	logging.Info("Stopping compliance report manager")
	concurrency.WithLock(&m.watchingScansLock, func() {
		for _, scanWatcher := range m.watchingScans {
			scanWatcher.Stop()
		}
	})
	m.stopper.Client().Stop()
	err := m.stopper.Client().Stopped().Wait()
	if err != nil {
		logging.Errorf("Error stopping compliance report manager : %v", err)
	}
}

func (m *managerImpl) generateReport(req *reportRequest) {
	defer m.concurrencySem.Release(1)

	clusterIds := []string{}
	profiles := []string{}
	for _, cluster := range req.scanConfig.GetClusters() {
		clusterIds = append(clusterIds, cluster.GetClusterId())
	}

	for _, profile := range req.scanConfig.GetProfiles() {
		profiles = append(profiles, profile.GetProfileName())
	}

	repRequest := &reportGen.ComplianceReportRequest{
		ScanConfigName: req.scanConfig.GetScanConfigName(),
		ScanConfigID:   req.scanConfig.GetId(),
		Profiles:       profiles,
		ClusterIDs:     clusterIds,
		Notifiers:      req.scanConfig.GetNotifiers(),
		Ctx:            req.ctx,
	}
	m.reportGen.ProcessReportRequest(repRequest)
	logging.Infof("Executing report request for scan config %q", req.scanConfig.GetId())

	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.runningReportConfigs, req.scanConfig.GetId())

}

func (m *managerImpl) runReports() {
	defer m.stopper.Flow().ReportStopped()
	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			logging.Info("Signal received to stop compliance report manager")
			return
		case req := <-m.reportRequests:
			if err := m.concurrencySem.Acquire(context.Background(), 1); err != nil {
				log.Errorf("Error acquiring semaphore to run new report: %v", err)
				continue
			}
			logging.Infof("Executing report %q at %v", req.scanConfig.GetId(), time.Now().Format(time.RFC822))
			go m.generateReport(req)
		}
	}
}

// HandleScan starts a new ScanWatcher if needed and pushes the scan to it
func (m *managerImpl) HandleScan(ctx context.Context, scan *storage.ComplianceOperatorScanV2) error {
	if !features.ComplianceReporting.Enabled() {
		return nil
	}
	id, err := watcher.GetWatcherIDFromScan(scan, nil)
	if err != nil {
		if err == watcher.ComplianceOperatorScanMissingLastStartedFiledError {
			log.Debugf("The scan is missing the LastStartedField: %v", err)
			return nil
		}
		return err
	}
	concurrency.WithLock(&m.watchingScansLock, func() {
		var scanWatcher watcher.ScanWatcher
		var found bool
		if scanWatcher, found = m.watchingScans[id]; !found {
			scanWatcher = watcher.NewScanWatcher(ctx, id, m.readyQueue)
			m.watchingScans[id] = scanWatcher
		}
		err = scanWatcher.PushScan(scan)
	})
	return err
}

// HandleResult starts a new ScanWatcher if needed and pushes the checkResult to it
func (m *managerImpl) HandleResult(ctx context.Context, result *storage.ComplianceOperatorCheckResultV2) error {
	if !features.ComplianceReporting.Enabled() {
		return nil
	}
	id, err := watcher.GetWatcherIDFromCheckResult(ctx, result, m.scanDataStore)
	if err != nil {
		if err == watcher.ComplianceOperatorReceivedOldCheckResultError {
			log.Debugf("The CheckResult is older than the current scan in the store")
			return nil
		}
		if err == watcher.ComplianceOperatorScanMissingLastStartedFiledError {
			log.Debugf("The scan is missing the LastStartedField: %v", err)
			return nil
		}
		return err
	}
	concurrency.WithLock(&m.watchingScansLock, func() {
		var scanWatcher watcher.ScanWatcher
		var found bool
		if scanWatcher, found = m.watchingScans[id]; !found {
			scanWatcher = watcher.NewScanWatcher(ctx, id, m.readyQueue)
			m.watchingScans[id] = scanWatcher
		}
		err = scanWatcher.PushCheckResult(result)
	})
	return err
}

// handleReadyScan pulls scans that are ready to be reported
func (m *managerImpl) handleReadyScan() {
	if !features.ComplianceReporting.Enabled() {
		return
	}
	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			return
		default:
			if scanResult := m.readyQueue.PullBlocking(m.stopper.LowLevel().GetStopRequestSignal()); scanResult != nil {
				log.Infof("Scan %s done with %d checks", scanResult.Scan.GetScanName(), len(scanResult.CheckResults))
				concurrency.WithLock(&m.watchingScansLock, func() {
					delete(m.watchingScans, scanResult.WatcherID)
				})
			}
		}
	}
}
