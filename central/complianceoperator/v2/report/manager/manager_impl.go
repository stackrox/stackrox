package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log         logging.Logger
	maxRequests = 10
)

type reportRequest struct {
	scanConfig *storage.ComplianceOperatorScanConfigurationV2
}

type managerImpl struct {
	datastore            scanConfigurationDS.DataStore
	runningReportConfigs map[string]*reportRequest
	//channel for report job requests
	reportRequests chan *reportRequest
	stopper        concurrency.Stopper

	//Mutex to synchronize access to runningReportConfigs map
	mu sync.Mutex
}

func New(scanConfigDS scanConfigurationDS.DataStore) Manager {
	return &managerImpl{
		datastore:            scanConfigDS,
		stopper:              concurrency.NewStopper(),
		runningReportConfigs: make(map[string]*reportRequest, maxRequests),
		reportRequests:       make(chan *reportRequest, maxRequests),
	}
}

func (m *managerImpl) SubmitReportRequest(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.runningReportConfigs[scanConfig.GetId()]; ok {
		return errors.New(fmt.Sprintf("Report request for scan configuration %q already in process", scanConfig.GetScanConfigName()))
	}

	log.Infof("Submitting report for scan at %v for execution", scanConfig.GetScanConfigName(), time.Now().Format(time.RFC822))
	req := &reportRequest{
		scanConfig,
	}
	m.reportRequests <- req
	m.runningReportConfigs[scanConfig.GetId()] = req

	return nil
}

func (m *managerImpl) Start() {
	go m.runReports()
}

func (m *managerImpl) Stop() {
	m.stopper.Client().Stop()
	err := m.stopper.Client().Stopped().Wait()
	if err != nil {
		logging.Errorf("Error stopping compliance report manager : %v", err)
	}
}

func (m *managerImpl) generateReport(req *reportRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// TODO: Implement business logic for querying, formatting and sending report over email
	logging.Infof("Executing report request for scan config %q", req.scanConfig.GetId())
	delete(m.runningReportConfigs, req.scanConfig.GetId())
}

func (m *managerImpl) runReports() {
	defer m.stopper.Flow().ReportStopped()
	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			return
		case req := <-m.reportRequests:
			logging.Infof("Executing report %q at %v", req.scanConfig.GetId(), time.Now().Format(time.RFC822))
			go m.generateReport(req)
		}
	}
}
