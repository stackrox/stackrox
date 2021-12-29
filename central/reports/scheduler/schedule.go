package scheduler

import (
	"context"

	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"gopkg.in/robfig/cron.v2"
)

var (
	log = logging.LoggerForModule()
)

// Scheduler maintains the schedules for reports
type Scheduler interface {
	UpsertReportSchedule(cronSpec string, reportConfig *storage.ReportConfiguration) error
	RemoveReportSchedule(reportConfigID string)
	SubmitReport(request *ReportRequest) error
	Start()
	Stop()
}

type scheduler struct {
	lock                   sync.Mutex
	cron                   *cron.Cron
	reportConfigToEntryIDs map[string]cron.EntryID

	reportConfigDatastore reportConfigDS.DataStore
	notifierDatastore     notifierDataStore.DataStore

	reportsToRun chan *ReportRequest

	stoppedSig concurrency.Signal
}

// ReportRequest is a request to the scheduler to run a scheduled or on demand report
type ReportRequest struct {
	ReportConfig *storage.ReportConfiguration
	OnDemand     bool
}

// New instantiates a new cron scheduler and supports adding and removing report configurations
func New() Scheduler {
	cronScheduler := cron.New()
	cronScheduler.Start()
	s := &scheduler{
		reportConfigToEntryIDs: make(map[string]cron.EntryID),
		cron:                   cronScheduler,
		reportConfigDatastore:  reportConfigDS.Singleton(),
		notifierDatastore:      notifierDataStore.Singleton(),
		reportsToRun:           make(chan *ReportRequest, 100),
	}
	go s.runReports()
	return s
}

func (s *scheduler) reportClosure(reportConfigID string) func() {
	return func() {
		reportConfig, found, err := s.reportConfigDatastore.GetReportConfiguration(context.Background(), reportConfigID)
		if !found {
			log.Errorf("Report config %s not found", reportConfigID)
			return
		}
		if err != nil {
			log.Errorf("error getting report config %s: %s", reportConfigID, err)
			return
		}
		log.Infof("Running report %s at %v", reportConfig.GetName(), timestamp.Now())
		if err := s.SubmitReport(&ReportRequest{
			ReportConfig: reportConfig,
			OnDemand:     false,
		}); err != nil {
			log.Error(err)
		}
	}
}

func (s *scheduler) UpsertReportSchedule(cronSpec string, reportConfig *storage.ReportConfiguration) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	entryID, err := s.cron.AddFunc(cronSpec, s.reportClosure(reportConfig.GetId()))
	if err != nil {
		return err
	}

	// Remove the old entry if this is an update
	if oldEntryID, ok := s.reportConfigToEntryIDs[reportConfig.GetId()]; ok {
		s.cron.Remove(oldEntryID)
	}
	s.reportConfigToEntryIDs[reportConfig.GetId()] = entryID
	return nil
}

func (s *scheduler) RemoveReportSchedule(id string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	oldEntryID := s.reportConfigToEntryIDs[id]
	s.cron.Remove(oldEntryID)
	delete(s.reportConfigToEntryIDs, id)
}

func (s *scheduler) SubmitReport(reportRequest *ReportRequest) error {
	s.reportsToRun <- reportRequest
	log.Infof("Submitted report %s at %v for execution", reportRequest.ReportConfig.GetName(), timestamp.Now())
	return nil
}

func (s *scheduler) updateReportStatus(reportConfig *storage.ReportConfiguration) error {
	return s.reportConfigDatastore.UpdateReportConfiguration(context.Background(), reportConfig)
}

func (s *scheduler) runReports() {
	select {
	case <-s.stoppedSig.Done():
		return
	case req := <-s.reportsToRun:
		log.Infof("Executing report %s at %v", req.ReportConfig.GetName(), timestamp.Now())
		err := sendReportResults(req.ReportConfig)
		if !req.OnDemand {
			// TODO: @khushboo for more accuracy, save timestamp when the vuln data is pulled aka the query is run
			if err != nil {
				req.ReportConfig.LastRunStatus = &storage.ReportLastRunStatus{
					ReportStatus: storage.ReportLastRunStatus_FAILURE,
					LastRunTime:  timestamp.Now().GogoProtobuf(),
					ErrorMsg:     err.Error(),
				}
			} else {
				req.ReportConfig.LastRunStatus = &storage.ReportLastRunStatus{
					ReportStatus: storage.ReportLastRunStatus_SUCCESS,
					LastRunTime:  timestamp.Now().GogoProtobuf(),
					ErrorMsg:     "",
				}
				req.ReportConfig.LastSuccessfulRunTime = timestamp.Now().GogoProtobuf()
			}
			if err := s.updateReportStatus(req.ReportConfig); err != nil {
				log.Errorf("unable to update last run status for report %s: %s", req.ReportConfig.GetName(), err)
			}
		}
	}
}

func sendReportResults(reportConfig *storage.ReportConfiguration) error {
	// TODO: To be implemented by hooking up the actual reporting (@khushboo)
	/*
	 1. Convert report config to v1.Query
	 2. Execute query using resolvers
	 3. Format results into CSV
	 4. Send CSV via email notifier
	*/
	return nil
}

func (s *scheduler) Start() {
	if !features.VulnReporting.Enabled() {
		return
	}
	s.stoppedSig.Reset()
}

func (s *scheduler) Stop() {
	s.stoppedSig.Signal()
}
