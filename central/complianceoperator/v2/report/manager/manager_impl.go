package manager

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	reportGen "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/complianceReportgenerator"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/utils"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/watcher"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/queue"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"golang.org/x/sync/semaphore"
)

var (
	log         = logging.LoggerForModule()
	maxRequests = 100
)

type reportRequest struct {
	scanConfig *storage.ComplianceOperatorScanConfigurationV2
	ctx        context.Context
	snapshotID string
}

type managerImpl struct {
	scanConfigDataStore scanConfigurationDS.DataStore
	scanDataStore       scanDS.DataStore
	profileDataStore    profileDatastore.DataStore
	snapshotDataStore   snapshotDS.DataStore

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

	watchingScanConfigsLock sync.Mutex
	// watchingScanConfigs a map holding the ScanConfigWatchers
	watchingScanConfigs map[string]watcher.ScanConfigWatcher
	// scanConfigReadyQueue holds the scan configurations that are ready to be reported
	scanConfigReadyQueue *queue.Queue[*watcher.ScanConfigWatcherResults]
}

func New(scanConfigDS scanConfigurationDS.DataStore, scanDataStore scanDS.DataStore, profileDataStore profileDatastore.DataStore, snapshotDatastore snapshotDS.DataStore, reportGen reportGen.ComplianceReportGenerator) Manager {
	return &managerImpl{
		scanConfigDataStore:  scanConfigDS,
		scanDataStore:        scanDataStore,
		profileDataStore:     profileDataStore,
		snapshotDataStore:    snapshotDatastore,
		stopper:              concurrency.NewStopper(),
		runningReportConfigs: make(map[string]*reportRequest, maxRequests),
		reportRequests:       make(chan *reportRequest, maxRequests),
		concurrencySem:       semaphore.NewWeighted(int64(env.ReportExecutionMaxConcurrency.IntegerSetting())),
		reportGen:            reportGen,
		watchingScans:        make(map[string]watcher.ScanWatcher),
		readyQueue:           queue.NewQueue[*watcher.ScanWatcherResults](),
		watchingScanConfigs:  make(map[string]watcher.ScanConfigWatcher),
		scanConfigReadyQueue: queue.NewQueue[*watcher.ScanConfigWatcherResults](),
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
	go m.handleReadyScanConfig()
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
			<-scanWatcher.Finished().Done()
		}
	})
	concurrency.WithLock(&m.watchingScanConfigsLock, func() {
		for _, scanConfigWatcher := range m.watchingScanConfigs {
			scanConfigWatcher.Stop()
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
		SnapshotID:     req.snapshotID,
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
			var w watcher.ScanConfigWatcher
			concurrency.WithLock(&m.watchingScanConfigsLock, func() {
				w = m.watchingScanConfigs[req.scanConfig.GetId()]
			})
			// These will be modified in a follow-up PR
			reportType := storage.ComplianceOperatorReportStatus_ON_DEMAND
			notificationMethod := storage.ComplianceOperatorReportStatus_EMAIL
			snapshot := &storage.ComplianceOperatorReportSnapshotV2{
				ReportId:            uuid.NewV4().String(),
				ScanConfigurationId: req.scanConfig.GetId(),
				Name:                req.scanConfig.GetScanConfigName(),
				Description:         req.scanConfig.GetDescription(),
				ReportStatus: &storage.ComplianceOperatorReportStatus{
					RunState:                 storage.ComplianceOperatorReportStatus_WAITING,
					StartedAt:                protocompat.TimestampNow(),
					ReportRequestType:        reportType,
					ReportNotificationMethod: notificationMethod,
				},
				User: req.scanConfig.GetModifiedBy(),
			}
			if w == nil {
				snapshot.GetReportStatus().RunState = storage.ComplianceOperatorReportStatus_PREPARING
				if err := m.snapshotDataStore.UpsertSnapshot(req.ctx, snapshot); err != nil {
					log.Errorf("Unable to upsert snapshot: %v", err)
					continue
				}
				req.snapshotID = snapshot.GetReportId()
				go m.generateReport(req)
				continue
			}
			if err := w.Subscribe(snapshot); err != nil {
				log.Errorf("Unable to subscribe to the scan configuration watcher")
				snapshot.GetReportStatus().RunState = storage.ComplianceOperatorReportStatus_FAILURE
				snapshot.GetReportStatus().CompletedAt = protocompat.TimestampNow()
				snapshot.GetReportStatus().ErrorMsg = utils.ErrUnableToSubscribeToWatcher.Error()
				if err := m.snapshotDataStore.UpsertSnapshot(req.ctx, snapshot); err != nil {
					log.Errorf("Unable to upsert snapshot: %v", err)
				}
				continue
			}
			if err := m.snapshotDataStore.UpsertSnapshot(req.ctx, snapshot); err != nil {
				log.Errorf("Unable to upsert snapshot: %v", err)
				continue
			}
		}
	}
}

// HandleScan starts a new ScanWatcher if needed and pushes the scan to it
func (m *managerImpl) HandleScan(ctx context.Context, scan *storage.ComplianceOperatorScanV2) error {
	if !features.ComplianceReporting.Enabled() {
		return nil
	}
	id, err := watcher.GetWatcherIDFromScan(ctx, scan, m.snapshotDataStore, nil)
	if err != nil {
		if errors.Is(err, watcher.ErrComplianceOperatorScanMissingLastStartedFiled) {
			log.Debugf("The scan is missing the LastStartedField: %v", err)
			return nil
		}
		if errors.Is(err, watcher.ErrScanAlreadyHandled) {
			log.Debugf("Scan %s was already handled", scan.GetScanName())
			return nil
		}
		return err
	}
	err = m.getWatcher(ctx, id).PushScan(scan)
	return err
}

func (m *managerImpl) getWatcher(ctx context.Context, id string) watcher.ScanWatcher {
	var scanWatcher watcher.ScanWatcher
	concurrency.WithLock(&m.watchingScansLock, func() {
		var found bool
		if scanWatcher, found = m.watchingScans[id]; !found {
			scanWatcher = watcher.NewScanWatcher(ctx, id, m.readyQueue)
			m.watchingScans[id] = scanWatcher
		}
	})
	return scanWatcher
}

// HandleResult starts a new ScanWatcher if needed and pushes the checkResult to it
func (m *managerImpl) HandleResult(ctx context.Context, result *storage.ComplianceOperatorCheckResultV2) error {
	if !features.ComplianceReporting.Enabled() {
		return nil
	}
	id, err := watcher.GetWatcherIDFromCheckResult(ctx, result, m.scanDataStore, m.snapshotDataStore)
	if err != nil {
		if errors.Is(err, watcher.ErrComplianceOperatorReceivedOldCheckResult) {
			log.Debugf("The CheckResult is older than the current scan in the store")
			return nil
		}
		if errors.Is(err, watcher.ErrComplianceOperatorScanMissingLastStartedFiled) {
			log.Debugf("The scan is missing the LastStartedField: %v", err)
			return nil
		}
		if errors.Is(err, watcher.ErrScanAlreadyHandled) {
			log.Debugf("The scan linked to the check result %s is already handled", result.GetCheckName())
			return nil
		}
		return err
	}
	err = m.getWatcher(ctx, id).PushCheckResult(result)
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
				// At the moment we simply do not start the ScanConfigWatcher if there are errors in the ScanWatchers
				// There are many reason why a ScanWatcher might fail like for example the Scan was deleted mid-execution
				// If this happens we will generate many ReportSnapshots with timeouts. Until we implement a way to
				// distinguish legitimate failures (CO not reporting back) we log the error and avoid creating the Report.
				if scanResult.Error != nil {
					log.Errorf("The scanResults returned with an error: %v", scanResult.Error)
					continue
				}
				log.Debugf("Scan %s done with %d checks", scanResult.Scan.GetScanName(), len(scanResult.CheckResults))
				w, err := m.getScanConfigWatcher(scanResult.Ctx, scanResult, m.scanConfigDataStore, m.scanConfigReadyQueue)
				if err != nil {
					log.Errorf("Unable to create the ScanConfigWatcher: %v", err)
					continue
				}
				if err := w.PushScanResults(scanResult); err != nil {
					log.Errorf("Unable to push scan %s: %v", scanResult.Scan.GetScanName(), err)
				}
				concurrency.WithLock(&m.watchingScansLock, func() {
					delete(m.watchingScans, scanResult.WatcherID)
				})
			}
		}
	}
}

// getScanConfigWatcher returns the ScanConfigWatcher of a given scan
func (m *managerImpl) getScanConfigWatcher(ctx context.Context, results *watcher.ScanWatcherResults, ds scanConfigurationDS.DataStore, queue *queue.Queue[*watcher.ScanConfigWatcherResults]) (watcher.ScanConfigWatcher, error) {
	sc, err := watcher.GetScanConfigFromScan(ctx, results.Scan, ds)
	if err != nil {
		return nil, errors.Errorf("unable to get scan config id: %v", err)
	}
	if sc == nil {
		return nil, errors.Errorf("ScanConfiguration not found for scan %s", results.Scan.GetScanName())
	}
	if len(sc.GetNotifiers()) == 0 {
		// If there aren't any notifiers configured we cannot report
		log.Warnf("The scan configuration %s has not configured notifiers", sc.GetScanConfigName())
		return nil, utils.ErrNoNotifiersConfigured
	}
	var w watcher.ScanConfigWatcher
	var watcherIsRunning bool
	concurrency.WithLock(&m.watchingScanConfigsLock, func() {
		if w, watcherIsRunning = m.watchingScanConfigs[sc.GetId()]; !watcherIsRunning {
			log.Debugf("Staring config watcher %s", sc.GetId())
			w = watcher.NewScanConfigWatcher(ctx, sc.GetId(), sc, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, queue)
			m.watchingScanConfigs[sc.GetId()] = w
		}
	})
	if !watcherIsRunning {
		snapshot := &storage.ComplianceOperatorReportSnapshotV2{
			ReportId:            uuid.NewV4().String(),
			ScanConfigurationId: sc.GetId(),
			Name:                sc.GetScanConfigName(),
			Description:         sc.GetDescription(),
			ReportStatus: &storage.ComplianceOperatorReportStatus{
				RunState:                 storage.ComplianceOperatorReportStatus_WAITING,
				StartedAt:                protocompat.TimestampNow(),
				ReportRequestType:        storage.ComplianceOperatorReportStatus_SCHEDULED,
				ReportNotificationMethod: storage.ComplianceOperatorReportStatus_EMAIL,
			},
			User: sc.GetModifiedBy(),
		}
		if err := m.snapshotDataStore.UpsertSnapshot(ctx, snapshot); err != nil {
			return nil, errors.Wrap(err, "unable to upsert the snapshot")
		}
		if err := w.Subscribe(snapshot); err != nil {
			log.Errorf("Unable to subscribe to the scan configuration watcher")
			snapshot.GetReportStatus().RunState = storage.ComplianceOperatorReportStatus_FAILURE
			snapshot.GetReportStatus().CompletedAt = protocompat.TimestampNow()
			snapshot.GetReportStatus().ErrorMsg = utils.ErrUnableToSubscribeToWatcher.Error()
			if err := m.snapshotDataStore.UpsertSnapshot(ctx, snapshot); err != nil {
				return nil, errors.Wrap(err, "unable to upsert the snapshot")
			}
			return nil, errors.Wrap(err, utils.ErrUnableToSubscribeToWatcher.Error())
		}
	}
	return w, nil
}

// handleReadyScanConfig pulls scan configs that are ready to be reported
func (m *managerImpl) handleReadyScanConfig() {
	if !features.ComplianceReporting.Enabled() {
		return
	}
	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			return
		default:
			if result := m.scanConfigReadyQueue.PullBlocking(m.stopper.LowLevel().GetStopRequestSignal()); result != nil {
				log.Debugf("Scan Config %s done with %d scans and %d reports", result.ScanConfig.GetScanConfigName(), len(result.ScanResults), len(result.ReportSnapshot))
				concurrency.WithLock(&m.watchingScanConfigsLock, func() {
					delete(m.watchingScanConfigs, result.WatcherID)
				})
				go m.generateReportsFromWatcherResults(result)
			}
		}
	}
}

func (m *managerImpl) generateReportsFromWatcherResults(result *watcher.ScanConfigWatcherResults) {
	for _, snapshot := range result.ReportSnapshot {
		if err := m.validateScanConfigResults(result); err != nil {
			snapshot.ReportStatus.RunState = storage.ComplianceOperatorReportStatus_FAILURE
			snapshot.GetReportStatus().CompletedAt = protocompat.TimestampNow()
			snapshot.GetReportStatus().ErrorMsg = utils.ErrScanWatchersFailed.Error()
			if err := m.snapshotDataStore.UpsertSnapshot(result.Ctx, snapshot); err != nil {
				log.Errorf("Unable to upsert the snapshot %s: %v", snapshot.GetReportId(), err)
			}
			continue
		}
		snapshot.ReportStatus.RunState = storage.ComplianceOperatorReportStatus_PREPARING
		if err := m.snapshotDataStore.UpsertSnapshot(result.Ctx, snapshot); err != nil {
			log.Errorf("Unable to upsert the snapshot %s: %v", snapshot.GetReportId(), err)
			continue
		}
		generateReportReq := &reportRequest{
			ctx:        result.Ctx,
			scanConfig: result.ScanConfig,
			snapshotID: snapshot.GetReportId(),
		}
		if err := m.concurrencySem.Acquire(context.Background(), 1); err != nil {
			log.Errorf("Error acquiring semaphore to run new report: %v", err)
			continue
		}
		m.generateReport(generateReportReq)
	}
}

func (m *managerImpl) validateScanConfigResults(result *watcher.ScanConfigWatcherResults) error {
	if result.Error != nil {
		return result.Error
	}

	errList := errorhelpers.NewErrorList("Scan result errors")
	for _, scanResult := range result.ScanResults {
		if scanResult.Error != nil {
			errList.AddErrors(scanResult.Error)
		}
	}

	return errList.ToError()
}
