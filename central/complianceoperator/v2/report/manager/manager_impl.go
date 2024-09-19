package manager

import (
	"container/list"
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	reportGen "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/complianceReportgenerator"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
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

//type scanResults struct {
//	runs map[string]*checkResults
//}

type checkResults struct {
	scan   *storage.ComplianceOperatorScanV2
	total  int
	checks map[string]*storage.ComplianceOperatorCheckResultV2
}

type managerImpl struct {
	datastore            scanConfigurationDS.DataStore
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

	scanLock sync.Mutex
	// cluster + scan -> results
	runningScans map[string]*checkResults
	// cluster + scan -> runs -> results
	readyScans *list.List

	watchingConfig map[string]map[string]struct{}
}

func New(scanConfigDS scanConfigurationDS.DataStore, reportGen reportGen.ComplianceReportGenerator) Manager {
	return &managerImpl{
		datastore:            scanConfigDS,
		stopper:              concurrency.NewStopper(),
		runningReportConfigs: make(map[string]*reportRequest, maxRequests),
		reportRequests:       make(chan *reportRequest, maxRequests),
		concurrencySem:       semaphore.NewWeighted(int64(env.ReportExecutionMaxConcurrency.IntegerSetting())),
		reportGen:            reportGen,
		runningScans:         make(map[string]*checkResults),
		readyScans:           list.New(),
		watchingConfig:       make(map[string]map[string]struct{}),
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
	go m.processReadyScan()
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

func (m *managerImpl) UpsertScan(scan *storage.ComplianceOperatorScanV2) error {
	m.scanLock.Lock()
	defer m.scanLock.Unlock()
	log.Infof("UpsertScan %s", scan.GetScanName())
	idx := fmt.Sprintf("%s:%s:%s", scan.GetClusterId(), scan.GetId(), scan.GetCreatedTime().String())
	log.Infof("Scan %s idx %s", scan.GetScanName(), idx)
	if results, ok := m.runningScans[idx]; ok {
		results.scan = scan
		if checks, ok := scan.GetAnnotations()["compliance.openshift.io/check-count"]; ok {
			if numChecks, err := strconv.Atoi(checks); err == nil && numChecks == len(results.checks) {
				// is ready
				log.Infof("Scan %s is ready", scan.GetScanName())
				m.readyScans.PushBack(results)
				delete(m.runningScans, idx)
			} else {
				if numChecks, err := strconv.Atoi(checks); err == nil {
					m.runningScans[idx].total = numChecks
				}
			}
		}
	} else {
		m.runningScans[idx] = &checkResults{
			scan:   scan,
			checks: make(map[string]*storage.ComplianceOperatorCheckResultV2, 0),
		}
	}
	return nil
}

func (m *managerImpl) UpsertResult(result *storage.ComplianceOperatorCheckResultV2) error {
	m.scanLock.Lock()
	defer m.scanLock.Unlock()
	log.Infof("UpsertResult %s", result.GetCheckName())
	var starttime string
	var ok bool
	if starttime, ok = result.GetAnnotations()["compliance.openshift.io/last-scanned-timestamp"]; !ok {
		return nil
	}
	timestamp, err := protocompat.ParseRFC3339NanoTimestamp(starttime)
	if err != nil {
		log.Errorf("unable to parse time: %v", err)
		return nil
	}
	scanDataStore := scanDS.Singleton()
	scanRefQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanRef, result.GetScanRefId()).
		ProtoQuery()
	scans, err := scanDataStore.SearchScans(sac.WithAllAccess(context.Background()), scanRefQuery)
	if err != nil {
		log.Errorf("unable to retrieve scan : %v", err)
		return nil
	}
	if len(scans) == 0 {
		log.Error("zero scans retrieved")
		return nil
	}
	id := scans[0].GetId()
	idx := fmt.Sprintf("%s:%s:%s", result.GetClusterId(), id, timestamp.String())
	log.Infof("Result %s idx %s", result.GetCheckName(), idx)
	if results, ok := m.runningScans[idx]; ok {
		results.checks[result.GetId()] = result
	} else {
		m.runningScans[idx] = &checkResults{checks: make(map[string]*storage.ComplianceOperatorCheckResultV2)}
		m.runningScans[idx].checks[result.GetId()] = result
	}
	if len(m.runningScans[idx].checks) == m.runningScans[idx].total {
		// is ready
		log.Infof("Scan %s is ready", m.runningScans[idx].scan.GetScanName())
		m.readyScans.PushBack(m.runningScans[idx])
		delete(m.runningScans, idx)
	}
	return nil
}

func (m *managerImpl) processReadyScan() {
	log.Info("starting process ready scan")
	for {
		time.Sleep(5 * time.Second)
		concurrency.WithLock(&m.scanLock, func() {
			scan := m.readyScans.Front()
			if scan == nil {
				return
			}
			results := m.readyScans.Remove(scan).(*checkResults)
			log.Infof("ready scan %s", results.scan.GetScanName())
			configDB := scanConfigurationDS.Singleton()
			config, _ := configDB.GetScanConfigurationByName(sac.WithAllAccess(context.Background()), results.scan.GetScanConfigName())
			if config != nil {
				log.Infof("scan config %s", config.GetScanConfigName())
			}
			if profiles, ok := m.watchingConfig[config.GetId()]; ok {
				log.Infof("config found %s", config.GetScanConfigName())
				// find profile
				profileDB := profileDatastore.Singleton()
				scanRefQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileRef, results.scan.GetProfile().GetProfileRefId()).
					ProtoQuery()

				profs, _ := profileDB.SearchProfiles(sac.WithAllAccess(context.Background()), scanRefQuery)
				if len(profs) == 0 {
					log.Error("no profiles found")
					return
				}
				log.Infof("config profiles %s %v", profs[0].GetName(), profiles)
				delete(profiles, fmt.Sprintf("%s:%s", results.scan.GetClusterId(), profs[0].GetName()))
			} else {
				log.Infof("config not found %s", config.GetScanConfigName())
				pros := make(map[string]struct{})
				for _, p := range config.GetProfiles() {
					pros[fmt.Sprintf("%s:%s", results.scan.GetClusterId(), p.GetProfileName())] = struct{}{}
				}
				profileDB := profileDatastore.Singleton()
				scanRefQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileRef, results.scan.GetProfile().GetProfileRefId()).
					ProtoQuery()

				profs, _ := profileDB.SearchProfiles(sac.WithAllAccess(context.Background()), scanRefQuery)
				if len(profs) == 0 {
					log.Error("no profiles found")
					return
				}
				log.Infof("config profiles %s %v", profs[0].GetName(), pros)
				delete(profiles, fmt.Sprintf("%s:%s", results.scan.GetClusterId(), profs[0].GetName()))
				m.watchingConfig[config.GetId()] = pros
			}
			if len(m.watchingConfig[config.GetId()]) == 0 {
				log.Info("The config is ready to be reported")
			}
		})
	}
}
