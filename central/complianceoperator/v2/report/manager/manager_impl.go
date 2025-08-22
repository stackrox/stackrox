package manager

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	checkResults "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	complianceIntegrationDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	reportGen "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/generator"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/helpers"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/watcher"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	bindingsDS "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore"
	suiteDS "github.com/stackrox/rox/central/complianceoperator/v2/suites/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/queue"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/uuid"
	"golang.org/x/sync/semaphore"
)

var (
	log         = logging.LoggerForModule()
	maxRequests = 100
)

type reportRequest struct {
	scanConfig         *storage.ComplianceOperatorScanConfigurationV2
	ctx                context.Context
	snapshotID         string
	notificationMethod storage.ComplianceOperatorReportStatus_NotificationMethod
	clusterData        map[string]*report.ClusterData
	numFailedClusters  int
}

type managerImpl struct {
	scanConfigDataStore  scanConfigurationDS.DataStore
	scanDataStore        scanDS.DataStore
	profileDataStore     profileDatastore.DataStore
	snapshotDataStore    snapshotDS.DataStore
	integrationDataStore complianceIntegrationDS.DataStore
	suiteDataStore       suiteDS.DataStore
	bindingsDataStore    bindingsDS.DataStore
	checkResultDataStore checkResults.DataStore

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
	// watchingScansStartTime records when a given watcher was started (for the metrics)
	watchingScansStartTime map[string]time.Time
	// readyQueue holds the scan that are ready to be reported
	readyQueue *queue.Queue[*watcher.ScanWatcherResults]

	automaticReportingCtx   context.Context
	watchingScanConfigsLock sync.Mutex
	// watchingScanConfigs a map holding the ScanConfigWatchers
	watchingScanConfigs map[string]watcher.ScanConfigWatcher
	// scanConfigReadyQueue holds the scan configurations that are ready to be reported
	scanConfigReadyQueue *queue.Queue[*watcher.ScanConfigWatcherResults]

	metricsTicker  *time.Ticker
	metricsTickerC <-chan time.Time
	// maxScansInParallel stores the maximum number of scans running in parallel between the ticks of metricsTicker.
	maxScansInParallel atomic.Int32
}

func New(scanConfigDS scanConfigurationDS.DataStore,
	scanDataStore scanDS.DataStore,
	profileDataStore profileDatastore.DataStore,
	snapshotDatastore snapshotDS.DataStore,
	complianceIntegration complianceIntegrationDS.DataStore,
	suiteDataStore suiteDS.DataStore,
	bindingsDataStore bindingsDS.DataStore,
	checkResultDataStore checkResults.DataStore,
	reportGen reportGen.ComplianceReportGenerator) Manager {
	gmt := time.NewTicker(env.ComplianceScansRunningInParallelMetricObservationPeriod.DurationSetting())
	return &managerImpl{
		scanConfigDataStore:    scanConfigDS,
		scanDataStore:          scanDataStore,
		profileDataStore:       profileDataStore,
		snapshotDataStore:      snapshotDatastore,
		integrationDataStore:   complianceIntegration,
		suiteDataStore:         suiteDataStore,
		bindingsDataStore:      bindingsDataStore,
		checkResultDataStore:   checkResultDataStore,
		stopper:                concurrency.NewStopper(),
		runningReportConfigs:   make(map[string]*reportRequest, maxRequests),
		reportRequests:         make(chan *reportRequest, maxRequests),
		concurrencySem:         semaphore.NewWeighted(int64(env.ReportExecutionMaxConcurrency.IntegerSetting())),
		reportGen:              reportGen,
		automaticReportingCtx:  sac.WithAllAccess(context.Background()),
		watchingScans:          make(map[string]watcher.ScanWatcher),
		watchingScansStartTime: make(map[string]time.Time),
		readyQueue:             queue.NewQueue[*watcher.ScanWatcherResults](),
		watchingScanConfigs:    make(map[string]watcher.ScanConfigWatcher),
		scanConfigReadyQueue:   queue.NewQueue[*watcher.ScanConfigWatcherResults](),
		metricsTicker:          gmt,
		metricsTickerC:         gmt.C,
	}
}

func (m *managerImpl) SubmitReportRequest(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2, method storage.ComplianceOperatorReportStatus_NotificationMethod) error {
	if !features.ComplianceReporting.Enabled() {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.runningReportConfigs[scanConfig.GetId()]; ok {
		return errors.New(fmt.Sprintf("Report request for scan configuration %q already in process", scanConfig.GetScanConfigName()))
	}

	req := &reportRequest{
		scanConfig:         scanConfig,
		ctx:                context.WithoutCancel(ctx),
		notificationMethod: method,
	}
	log.Infof("Submitting report for scan config %s at %s with method %s", scanConfig.GetScanConfigName(), time.Now().Format(time.RFC822), method.String())
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
	go m.updateMetrics()
}

func (m *managerImpl) Stop() {
	m.metricsTicker.Stop()
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
			scanWatcher.Stop(nil)
			<-scanWatcher.Finished().Done()
		}
		m.watchingScans = make(map[string]watcher.ScanWatcher)
		m.watchingScansStartTime = make(map[string]time.Time)
	})
	concurrency.WithLock(&m.watchingScanConfigsLock, func() {
		for _, scanConfigWatcher := range m.watchingScanConfigs {
			scanConfigWatcher.Stop()
		}
		m.watchingScanConfigs = make(map[string]watcher.ScanConfigWatcher)
	})

	m.reportGen.Stop()

	m.stopper.Client().Stop()
	err := m.stopper.Client().Stopped().Wait()
	if err != nil {
		logging.Errorf("Error stopping compliance report manager : %v", err)
	}

}

func (m *managerImpl) generateReportNoLock(req *reportRequest) {
	clusterIds := []string{}
	profiles := []string{}
	for _, cluster := range req.scanConfig.GetClusters() {
		clusterIds = append(clusterIds, cluster.GetClusterId())
	}

	for _, profile := range req.scanConfig.GetProfiles() {
		profiles = append(profiles, profile.GetProfileName())
	}

	repRequest := &report.Request{
		ScanConfigName:     req.scanConfig.GetScanConfigName(),
		ScanConfigID:       req.scanConfig.GetId(),
		Profiles:           profiles,
		ClusterIDs:         clusterIds,
		Notifiers:          req.scanConfig.GetNotifiers(),
		Ctx:                req.ctx,
		SnapshotID:         req.snapshotID,
		NotificationMethod: req.notificationMethod,
		ClusterData:        req.clusterData,
		NumFailedClusters:  req.numFailedClusters,
	}
	log.Infof("Executing report request for scan config %q", req.scanConfig.GetId())
	if err := m.reportGen.ProcessReportRequest(repRequest); err != nil {
		log.Errorf("unable to process the report request: %v", err)
	}
}

func (m *managerImpl) runReports() {
	defer m.stopper.Flow().ReportStopped()
	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			logging.Info("Signal received to stop compliance report manager")
			return
		case req := <-m.reportRequests:
			go func() {
				wasGenerated, err := m.handleReportRequest(req)
				// If the report was generated or the returned with an error we need to
				// delete the scan configuration entry from runningReportConfigs
				if err != nil || wasGenerated {
					concurrency.WithLock(&m.mu, func() {
						delete(m.runningReportConfigs, req.scanConfig.GetId())
					})
				}
				if err != nil {
					log.Errorf("unable to handle the report request: %v", err)
				}
			}()
		}
	}
}

func (m *managerImpl) handleReportRequest(request *reportRequest) (bool, error) {
	if err := m.concurrencySem.Acquire(context.Background(), 1); err != nil {
		return false, errors.Wrap(err, "unable acquiring semaphore to run new report")
	}
	defer m.concurrencySem.Release(1)

	log.Infof("Executing report %q at %v", request.scanConfig.GetId(), time.Now().Format(time.RFC822))

	if !features.ScanScheduleReportJobs.Enabled() {
		m.generateReportNoLock(request)
		return true, nil
	}

	requesterID := authn.IdentityFromContextOrNil(request.ctx)
	if requesterID == nil {
		return false, errors.New("could not determine user identity from provided context")
	}

	var w watcher.ScanConfigWatcher
	concurrency.WithLock(&m.watchingScanConfigsLock, func() {
		w = m.watchingScanConfigs[request.scanConfig.GetId()]
	})
	snapshot := &storage.ComplianceOperatorReportSnapshotV2{
		ReportId:            uuid.NewV4().String(),
		ScanConfigurationId: request.scanConfig.GetId(),
		Name:                request.scanConfig.GetScanConfigName(),
		Description:         request.scanConfig.GetDescription(),
		ReportStatus: &storage.ComplianceOperatorReportStatus{
			RunState:                 storage.ComplianceOperatorReportStatus_WAITING,
			StartedAt:                protocompat.TimestampNow(),
			ReportRequestType:        storage.ComplianceOperatorReportStatus_ON_DEMAND,
			ReportNotificationMethod: request.notificationMethod,
		},
		ReportData: m.getReportData(request.scanConfig),
		User: &storage.SlimUser{
			Id:   requesterID.UID(),
			Name: stringutils.FirstNonEmpty(requesterID.FullName(), requesterID.FriendlyName()),
		},
	}
	if w == nil {
		// The report is going to be generated now
		snapshot.GetReportStatus().RunState = storage.ComplianceOperatorReportStatus_PREPARING
		if err := m.snapshotDataStore.UpsertSnapshot(request.ctx, snapshot); err != nil {
			return false, errors.Wrap(err, "unable to upsert snapshot on report preparation")
		}
		request.snapshotID = snapshot.GetReportId()
		failedClusters, err := helpers.GetFailedClusters(m.automaticReportingCtx, request.scanConfig.GetId(), m.snapshotDataStore, m.scanDataStore)
		if err != nil {
			log.Warnf("unable to retrieve failed clusters: %v", err)
		}
		request.numFailedClusters = len(failedClusters)
		request.clusterData, err = helpers.GetClusterData(m.automaticReportingCtx, snapshot.GetReportData(), failedClusters, m.scanDataStore)
		if err != nil {
			log.Errorf("unable to get clusters information: %v", err)
			if dbErr := helpers.UpdateSnapshotOnError(request.ctx, snapshot, report.ErrReportGeneration, m.snapshotDataStore); dbErr != nil {
				return false, errors.Wrap(dbErr, "unable to upsert snapshot on generation failure")
			}
			return false, errors.Wrap(err, "unable to get clusters information")
		}
		// Add failed clusters to the report snapshot
		if _, err = m.addFailedClustersToTheSnapshot(failedClusters, snapshot); err != nil {
			log.Errorf("unable to updata snapshot with failed clusters: %v", err)
			return false, err
		}
		m.generateReportNoLock(request)
		return true, nil
	}
	// There is a ScanConfigWatcher running for this Scan Configuration.
	// This means we cannot generate the report at this moment.
	// We subscribe to the watcher to later generate the report once it's finished.
	if err := w.Subscribe(snapshot); err != nil {
		if dbErr := helpers.UpdateSnapshotOnError(request.ctx, snapshot, report.ErrUnableToSubscribeToWatcher, m.snapshotDataStore); dbErr != nil {
			return false, errors.Wrap(dbErr, "unable to upsert snapshot on watcher subscription failure")
		}
		return false, errors.New("unable to subscribe to the scan configuration watcher")
	}
	if scans := w.GetScans(); len(scans) > 0 {
		snapshot.Scans = scans
	}
	if err := m.snapshotDataStore.UpsertSnapshot(request.ctx, snapshot); err != nil {
		return false, errors.Wrap(err, "unable to upsert snapshot on report waiting")
	}
	return false, nil
}

// HandleScan starts a new ScanWatcher if needed and pushes the scan to it
func (m *managerImpl) HandleScan(sensorCtx context.Context, scan *storage.ComplianceOperatorScanV2) error {
	if !features.ComplianceReporting.Enabled() || !features.ScanScheduleReportJobs.Enabled() {
		return nil
	}
	id, err := watcher.GetWatcherIDFromScan(m.automaticReportingCtx, scan, m.snapshotDataStore, m.scanConfigDataStore, nil)
	if err != nil {
		if errors.Is(err, watcher.ErrComplianceOperatorScanMissingLastStartedFiled) {
			log.Debug("The scan is missing the LastStartedTime field")
			return nil
		}
		if errors.Is(err, watcher.ErrScanAlreadyHandled) {
			log.Debugf("Scan %s was already handled", scan.GetScanName())
			return nil
		}
		return err
	}
	numChecks, err := watcher.GetExpectedNumChecks(scan)
	if err != nil {
		log.Warnf("Failed to get expected number of checks from annotations for %s: %v", scan.GetScanName(), err)
	}
	w := m.getWatcher(sensorCtx, id, numChecks)
	if w != nil {
		return w.PushScan(scan)
	}
	log.Debugf("Received scan update after removing the watcher %+v", scan)
	return nil
}

func (m *managerImpl) updateMetrics() {
	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			return
		case <-m.metricsTickerC:
			nRunning := concurrency.WithLock1(&m.watchingScansLock, func() int {
				return len(m.watchingScans)
			})
			numWatchers.Set(float64(nRunning))
			// Reset the maximum value on tick and set to the current number
			prevVal := m.maxScansInParallel.Swap(int32(nRunning))
			log.Debugf("Updating maxScansInParallel from %d to %d (tick)", prevVal, nRunning)
			if prevVal > 0 {
				scansRunningInParallel.Observe(float64(prevVal))
			}
		}
	}
}

func (m *managerImpl) updateMaxNumScansRunningInParallelNoLock() {
	newVal := max(m.maxScansInParallel.Load(), int32(len(m.watchingScans)))
	prevVal := m.maxScansInParallel.Swap(newVal)
	log.Debugf("Updating maxScansInParallel from %d to %d", prevVal, newVal)
}

func (m *managerImpl) HandleScanRemove(scanID string) error {
	if !features.ComplianceReporting.Enabled() || !features.ScanScheduleReportJobs.Enabled() {
		return nil
	}
	scan, found, err := m.scanDataStore.GetScan(m.automaticReportingCtx, scanID)
	if err != nil {
		return errors.Wrapf(err, "unable to retrieve scan %s from the datastore", scanID)
	}
	if !found {
		return errors.Errorf("unable to find the scan %s in the datastore", scanID)
	}
	id := fmt.Sprintf("%s:%s", scan.GetClusterId(), scan.GetId())
	concurrency.WithLock(&m.watchingScansLock, func() {
		if scanWatcher, found := m.watchingScans[id]; found {
			scanWatcher.Stop(watcher.ErrScanRemoved)
		}
	})
	return nil
}

func (m *managerImpl) getWatcher(sensorCtx context.Context, id string, numChecks int) watcher.ScanWatcher {
	var scanWatcher watcher.ScanWatcher
	concurrency.WithLock(&m.watchingScansLock, func() {
		var found bool
		// The check for `numChecks == 0` is here to prevent starting a watcher twice per scan.
		// It may happen that additional status updates (e.g., state) from CO arrive
		// after the watcher is removed from the watchingScans (i.e., we have all the checks).
		// Not checking that would cause a new watcher to be created here and in some circumstances
		// (when no e-mail is provided for notification), the watcher would time-out and delete the data from DB.
		if scanWatcher, found = m.watchingScans[id]; !found && numChecks == 0 {
			scanWatcher = watcher.NewScanWatcher(m.automaticReportingCtx, sensorCtx, id, m.readyQueue)
			m.watchingScans[id] = scanWatcher
			m.watchingScansStartTime[id] = time.Now()
			m.updateMaxNumScansRunningInParallelNoLock()
		}
	})
	return scanWatcher
}

// HandleResult starts a new ScanWatcher if needed and pushes the checkResult to it
func (m *managerImpl) HandleResult(sensorCtx context.Context, result *storage.ComplianceOperatorCheckResultV2) error {
	if !features.ComplianceReporting.Enabled() || !features.ScanScheduleReportJobs.Enabled() {
		return nil
	}
	id, err := watcher.GetWatcherIDFromCheckResult(m.automaticReportingCtx, result, m.scanDataStore, m.snapshotDataStore, m.scanConfigDataStore)
	if err != nil {
		if errors.Is(err, watcher.ErrComplianceOperatorReceivedOldCheckResult) {
			log.Debugf("The CheckResult is older than the current scan in the store")
			return err
		}
		if errors.Is(err, watcher.ErrComplianceOperatorScanMissingLastStartedFiled) {
			log.Debug("The scan is missing the LastStartedTime field")
			return err
		}
		if errors.Is(err, watcher.ErrScanAlreadyHandled) {
			log.Debugf("The scan linked to the check result %s is already handled", result.GetCheckName())
			return err
		}
		return err
	}
	w := m.getWatcher(sensorCtx, id, 0)
	if w != nil {
		return w.PushCheckResult(result)
	}
	log.Debugf("Received check result update after removing the watcher %+v", result)
	return nil
}

// handleReadyScan pulls scans that are ready to be reported
func (m *managerImpl) handleReadyScan() {
	if !features.ComplianceReporting.Enabled() || !features.ScanScheduleReportJobs.Enabled() {
		return
	}
	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			return
		default:
			if scanWatcherResult := m.readyQueue.PullBlocking(m.stopper.LowLevel().GetStopRequestSignal()); scanWatcherResult != nil {
				concurrency.WithLock(&m.watchingScansLock, func() {
					delete(m.watchingScans, scanWatcherResult.WatcherID)

					m.maxScansInParallel.Store(int32(len(m.watchingScans)))
					timeActive := time.Since(m.watchingScansStartTime[scanWatcherResult.WatcherID])
					scanWatcherActiveTimeMinutes.WithLabelValues(scanWatcherResult.Scan.GetScanName()).
						Observe(timeActive.Minutes())
					delete(m.watchingScansStartTime, scanWatcherResult.WatcherID)
				})
				if err := watcher.DeleteOldResults(m.automaticReportingCtx, scanWatcherResult, m.checkResultDataStore); err != nil {
					log.Errorf("unable to delete old CheckResults: %v", err)
				}
				if errors.Is(scanWatcherResult.Error, watcher.ErrScanRemoved) {
					log.Debugf("Scan %s was removed", scanWatcherResult.Scan.GetScanName())
					continue
				}
				log.Debugf("Scan %s done with %d checks", scanWatcherResult.Scan.GetScanName(), len(scanWatcherResult.CheckResults))
				w, scanConfig, wasAlreadyRunning, err := m.getOrCreateScanConfigWatcher(scanWatcherResult.SensorCtx, scanWatcherResult, m.scanConfigDataStore, m.scanConfigReadyQueue)
				if errors.Is(err, watcher.ErrScanAlreadyHandled) {
					continue
				}
				if err != nil {
					log.Errorf("Unable to create the ScanConfigWatcher: %v", err)
					continue
				}
				if !wasAlreadyRunning {
					// if there are no notifiers configured we need to still push the results as they might be on-demand request
					if err := m.createAutomaticSnapshotAndSubscribe(m.automaticReportingCtx, scanConfig, w); err != nil && !errors.Is(err, report.ErrNoNotifiersConfigured) {
						log.Errorf("Unable to create the snapshot: %v", err)
						continue
					}
				}
				if err := w.PushScanResults(scanWatcherResult); err != nil {
					log.Errorf("Unable to push scan %s: %v", scanWatcherResult.Scan.GetScanName(), err)
				}
			}
		}
	}
}

// getOrCreateScanConfigWatcher returns the ScanConfigWatcher of a given scan
func (m *managerImpl) getOrCreateScanConfigWatcher(ctx context.Context, results *watcher.ScanWatcherResults, ds scanConfigurationDS.DataStore, queue *queue.Queue[*watcher.ScanConfigWatcherResults]) (watcher.ScanConfigWatcher, *storage.ComplianceOperatorScanConfigurationV2, bool, error) {
	sc, err := watcher.GetScanConfigFromScan(m.automaticReportingCtx, results.Scan, ds)
	if err != nil {
		return nil, nil, false, errors.Errorf("unable to get scan config id: %v", err)
	}
	if sc == nil {
		return nil, nil, false, errors.Errorf("ScanConfiguration not found for scan %s", results.Scan.GetScanName())
	}
	w, watcherIsRunning := concurrency.WithLock2[watcher.ScanConfigWatcher, bool](&m.watchingScanConfigsLock, func() (watcher.ScanConfigWatcher, bool) {
		w, watcherIsRunning := m.watchingScanConfigs[sc.GetId()]
		return w, watcherIsRunning
	})
	if !watcherIsRunning {
		query := search.NewQueryBuilder().
			AddExactMatches(search.ComplianceOperatorScanRef, results.Scan.GetScanRefId()).
			AddTimeRangeField(search.ComplianceOperatorScanLastStartedTime, results.Scan.GetLastStartedTime().AsTime(), timestamp.InfiniteFuture.GoTime()).
			ProtoQuery()
		snapshot, err := m.snapshotDataStore.SearchSnapshots(m.automaticReportingCtx, query)
		if err != nil {
			return nil, nil, false, errors.Wrap(err, "unable to retrieve snapshots from the store")
		}
		if len(snapshot) > 0 {
			// We already handled a scan newer than this one, we ignore this scanResults
			return nil, nil, false, watcher.ErrScanAlreadyHandled
		}
		log.Debugf("Staring config watcher %s", sc.GetId())
		concurrency.WithLock(&m.watchingScanConfigsLock, func() {
			w = watcher.NewScanConfigWatcher(m.automaticReportingCtx, ctx, sc.GetId(), sc, m.scanDataStore, m.profileDataStore, m.snapshotDataStore, queue)
			m.watchingScanConfigs[sc.GetId()] = w
		})
	}
	return w, sc, watcherIsRunning, nil
}

// createAutomaticSnapshotAndSubscribe creates a snapshot for an automatic report (not on-demand)
func (m *managerImpl) createAutomaticSnapshotAndSubscribe(ctx context.Context, sc *storage.ComplianceOperatorScanConfigurationV2, w watcher.ScanConfigWatcher) error {
	// If there aren't any notifiers configured we cannot report
	if len(sc.GetNotifiers()) == 0 {
		log.Warnf("The scan configuration %s has not configured notifiers", sc.GetScanConfigName())
		return report.ErrNoNotifiersConfigured
	}
	// If the watcher is not running we need to create a new snapshot
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
		ReportData: m.getReportData(sc),
		User:       sc.GetModifiedBy(),
	}
	if err := w.Subscribe(snapshot); err != nil {
		log.Errorf("Unable to subscribe to the scan configuration watcher")
		if dbErr := helpers.UpdateSnapshotOnError(ctx, snapshot, report.ErrUnableToSubscribeToWatcher, m.snapshotDataStore); dbErr != nil {
			return errors.Wrap(dbErr, "unable to upsert the snapshot")
		}
		return errors.Wrap(err, report.ErrUnableToSubscribeToWatcher.Error())
	}
	if scans := w.GetScans(); len(scans) > 0 {
		snapshot.Scans = scans
	}
	if err := m.snapshotDataStore.UpsertSnapshot(ctx, snapshot); err != nil {
		return errors.Wrap(err, "unable to upsert the snapshot")
	}
	return nil
}

// handleReadyScanConfig pulls scan configs that are ready to be reported
func (m *managerImpl) handleReadyScanConfig() {
	if !features.ComplianceReporting.Enabled() || !features.ScanScheduleReportJobs.Enabled() {
		return
	}
	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			return
		default:
			if scanConfigWatcherResult := m.scanConfigReadyQueue.PullBlocking(m.stopper.LowLevel().GetStopRequestSignal()); scanConfigWatcherResult != nil {
				log.Debugf("Scan Config %s done with %d scans and %d reports", scanConfigWatcherResult.ScanConfig.GetScanConfigName(), len(scanConfigWatcherResult.ScanResults), len(scanConfigWatcherResult.ReportSnapshot))
				concurrency.WithLock(&m.watchingScanConfigsLock, func() {
					delete(m.watchingScanConfigs, scanConfigWatcherResult.WatcherID)
				})
				if err := watcher.DeleteOldResultsFromMissingScans(m.automaticReportingCtx, scanConfigWatcherResult, m.profileDataStore, m.scanDataStore, m.checkResultDataStore); err != nil {
					log.Errorf("unable to delete old CheckResults: %v", err)
				}
				m.generateReportsFromWatcherResults(scanConfigWatcherResult)
			}
		}
	}
}

func (m *managerImpl) generateReportsFromWatcherResults(result *watcher.ScanConfigWatcherResults) {
	for _, snapshot := range result.ReportSnapshot {
		if err := m.generateSingleReportFromWatcherResults(result, snapshot); err != nil {
			// if there is an error we need to free the on-demand request from runningReportConfigs
			// if there are no error the map will be cleared in the success path
			if snapshot.GetReportStatus().GetReportRequestType() == storage.ComplianceOperatorReportStatus_ON_DEMAND {
				concurrency.WithLock(&m.mu, func() {
					delete(m.runningReportConfigs, result.ScanConfig.GetId())
				})
			}
			log.Errorf("unable to generate report: %v", err)
		}
	}
}

func (m *managerImpl) generateSingleReportFromWatcherResults(result *watcher.ScanConfigWatcherResults, snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
	failedClusters, err := watcher.ValidateScanConfigResults(m.automaticReportingCtx, result, m.integrationDataStore)
	snapshot.GetReportStatus().RunState = storage.ComplianceOperatorReportStatus_PREPARING
	if err != nil {
		snapshot.GetReportStatus().ErrorMsg = err.Error()
	}
	log.Infof("Snapshot for ScanConfig %s: %+v -- %+v", result.ScanConfig.GetScanConfigName(), snapshot.GetReportStatus(), snapshot.GetFailedClusters())
	// Update ReportData
	snapshot.ReportData = m.getReportData(result.ScanConfig)
	// Populate ClusterData
	clusterData, err := helpers.GetClusterData(m.automaticReportingCtx, snapshot.ReportData, failedClusters, m.scanDataStore)
	if err != nil {
		log.Errorf("unable to populate cluster data: %v", err)
		if dbErr := helpers.UpdateSnapshotOnError(m.automaticReportingCtx, snapshot, report.ErrReportGeneration, m.snapshotDataStore); dbErr != nil {
			return errors.Wrap(dbErr, "unable to update snapshot on populate cluster data error")
		}
		return errors.Wrap(err, "unable to populate cluster data")
	}
	// Add failed clusters to the report snapshot
	snapshot, err = m.addFailedClustersToTheSnapshot(failedClusters, snapshot)
	if err != nil {
		return err
	}
	generateReportReq := &reportRequest{
		ctx:                m.automaticReportingCtx,
		scanConfig:         result.ScanConfig,
		snapshotID:         snapshot.GetReportId(),
		notificationMethod: snapshot.GetReportStatus().GetReportNotificationMethod(),
		numFailedClusters:  len(failedClusters),
		clusterData:        clusterData,
	}
	isOnDemand := snapshot.GetReportStatus().GetReportRequestType() == storage.ComplianceOperatorReportStatus_ON_DEMAND
	if err := m.handleReportScheduled(generateReportReq, isOnDemand); err != nil {
		return errors.Wrap(err, "unable to handle the report")
	}
	return nil
}

func (m *managerImpl) addFailedClustersToTheSnapshot(failedClusters map[string]*report.FailedCluster, snapshot *storage.ComplianceOperatorReportSnapshotV2) (*storage.ComplianceOperatorReportSnapshotV2, error) {
	if len(failedClusters) == 0 {
		return snapshot, nil
	}
	failedClustersSlice := make([]*storage.ComplianceOperatorReportSnapshotV2_FailedCluster, 0, len(failedClusters))
	for _, failedCluster := range failedClusters {
		scans := make([]string, 0, len(failedCluster.FailedScans))
		for _, scan := range failedCluster.FailedScans {
			scans = append(scans, scan.GetScanName())
		}
		failedClustersSlice = append(failedClustersSlice, &storage.ComplianceOperatorReportSnapshotV2_FailedCluster{
			ClusterId:       failedCluster.ClusterId,
			ClusterName:     failedCluster.ClusterName,
			OperatorVersion: failedCluster.OperatorVersion,
			Reasons:         failedCluster.Reasons,
			ScanNames:       scans,
		})
	}
	snapshot.FailedClusters = failedClustersSlice
	if err := m.snapshotDataStore.UpsertSnapshot(m.automaticReportingCtx, snapshot); err != nil {
		return snapshot, errors.Wrapf(err, "unable to upsert the snapshot %s", snapshot.GetReportId())
	}
	return snapshot, nil
}

func (m *managerImpl) handleReportScheduled(request *reportRequest, isOnDemand bool) error {
	if err := m.concurrencySem.Acquire(context.Background(), 1); err != nil {
		return errors.Wrap(err, "Error acquiring semaphore to run new report")
	}
	go func() {
		defer m.concurrencySem.Release(1)
		m.generateReportNoLock(request)
		// Only delete from the runningReportConfigs if the snapshots was generated through the UI
		if isOnDemand {
			concurrency.WithLock(&m.mu, func() {
				delete(m.runningReportConfigs, request.scanConfig.GetId())
			})
		}
	}()
	return nil
}

func (m *managerImpl) getReportData(scanConfig *storage.ComplianceOperatorScanConfigurationV2) *storage.ComplianceOperatorReportData {
	reportData, err := helpers.ConvertScanConfigurationToReportData(m.automaticReportingCtx, scanConfig, m.scanConfigDataStore, m.suiteDataStore, m.bindingsDataStore)
	if err != nil {
		log.Warnf("Unable to convert ScanConfiguration %s to ReportData: %v", scanConfig.GetId(), err)
		return nil
	}
	return reportData
}
