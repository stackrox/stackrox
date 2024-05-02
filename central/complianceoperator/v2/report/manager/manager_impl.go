package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	maxsizeQueue = 10
)

type reportJob struct {
	scanConfigID string
}

type managerImpl struct {
	datastore            scanConfigurationDS.DataStore
	runningReportConfigs map[string]*reportJob
	queuedReports        chan *reportJob // channel for queueing report job requests
	stopper              concurrency.Stopper
	mu                   sync.Mutex // Mutex to synchronize access to queuedReports map
	wg                   sync.WaitGroup
}

func New(scanConfigDS scanConfigurationDS.DataStore) Manager {
	manager := &managerImpl{
		datastore:            scanConfigDS,
		stopper:              concurrency.NewStopper(),
		runningReportConfigs: make(map[string]*reportJob),
	}
	manager.queuedReports = make(chan *reportJob, maxsizeQueue)
	return manager
}

func (m *managerImpl) SubmitReportRequest(ctx context.Context, scanConfigID string) error {
	// verify can be queued based on max size of queue and job from same config is not queued before
	// queue report job requests
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.runningReportConfigs[scanConfigID]; ok {
		return errors.New("Job running for report config")
	}
	// add job to queue
	return m.appendToReportsQueue(scanConfigID)
}

func (m *managerImpl) start() {
	go m.runReports()
}

func (m *managerImpl) Stop() {
	m.stopper.Client().Stop()
	err := m.stopper.Client().Stopped().Wait()
	if err != nil {
		logging.Errorf("Error stopping compliance report manager : %v", err)
	}
}

func (m *managerImpl) generateReport(job *reportJob) {
	defer m.wg.Done()
	// adding print statement for now. will implement logic to generate reports and call it generateReport in a follow up PR
	fmt.Printf("running job for scan config id %s", job)
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.runningReportConfigs, job.scanConfigID)
}

func (m *managerImpl) runReports() {
	defer m.stopper.Flow().ReportStopped()
	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			return
		case req := <-m.queuedReports:
			m.wg.Add(1)
			logging.Infof("Executing report '%s' at %v", req.scanConfigID, time.Now().Format(time.RFC822))
			m.generateReport(req)
			m.wg.Wait()
		}
	}
}

func (m *managerImpl) appendToReportsQueue(scanConfigID string) error {
	if len(m.queuedReports) == maxsizeQueue {
		return errors.New("Max number of jobs queued")
	}
	job := &reportJob{
		scanConfigID,
	}

	m.queuedReports <- job
	m.runningReportConfigs[scanConfigID] = job
	return nil
}
