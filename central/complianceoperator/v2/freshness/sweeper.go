package freshness

import (
	"context"
	"time"

	"github.com/adhocore/gronx"
	checkResultDS "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	scanConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()

	// This cannot be tested without building complete graph, hence this elevated context
	sweeperCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
)

// Sweeper periodically re-evaluates the freshness of compliance check results.
type Sweeper interface {
	Start()
	Stop()
}

type sweeperImpl struct {
	interval time.Duration
	ticker   *time.Ticker

	scanConfigDS  scanConfigDS.DataStore
	checkResultDS checkResultDS.DataStore

	stopper concurrency.Stopper
}

// New returns a new instance of a Sweeper.
func New(scanConfigDS scanConfigDS.DataStore, checkResultDS checkResultDS.DataStore, interval time.Duration) Sweeper {
	return &sweeperImpl{
		interval: interval,

		scanConfigDS:  scanConfigDS,
		checkResultDS: checkResultDS,

		stopper: concurrency.NewStopper(),
	}
}

// Start starts the compliance freshness sweeper loop.
func (s *sweeperImpl) Start() {
	s.ticker = time.NewTicker(s.interval)
	go s.loop()
}

// Stop stops the compliance freshness sweeper loop.
func (s *sweeperImpl) Stop() {
	s.stopper.Client().Stop()
	_ = s.stopper.Client().Stopped().Wait()
}

func (s *sweeperImpl) loop() {
	defer s.stopper.Flow().ReportStopped()
	defer s.ticker.Stop()

	// Run sweeps synchronously in this single goroutine. loop() already runs in its own
	// goroutine (see Start), so the first sweep does not block the caller. Keeping sweeps
	// serial means a tick cannot start a second sweep while the previous one is still running
	// on a large database. Stop() waits for this goroutine to return, so no sweep keeps
	// writing after shutdown.
	s.sweep()
	for {
		select {
		case <-s.stopper.Flow().StopRequested():
			return
		case <-s.ticker.C:
			s.sweep()
		}
	}
}

// expectedRefresh returns the most recent time, before now, at which sched was expected to fire.
// The second return value is false if sched is nil or invalid.
func expectedRefresh(sched *storage.Schedule, now time.Time) (time.Time, bool) {
	if sched == nil {
		return time.Time{}, false
	}

	cron, err := schedule.ConvertToCronTab(sched)
	if err != nil || cron == "" {
		return time.Time{}, false
	}

	prev, err := gronx.PrevTickBefore(cron, now, true)
	if err != nil {
		return time.Time{}, false
	}
	return prev, true
}

func (s *sweeperImpl) sweep() {
	select {
	case <-s.stopper.Flow().StopRequested():
		return
	default:
	}

	now := time.Now()
	grace := env.ComplianceScanScheduleWatcherTimeout.DurationSetting() + env.ComplianceScanWatcherTimeout.DurationSetting()

	configs, err := s.scanConfigDS.GetScanConfigurations(sweeperCtx, search.EmptyQuery())
	if err != nil {
		log.Errorf("error retrieving compliance scan configurations: %v", err)
		return
	}

	for _, cfg := range configs {
		exp, ok := expectedRefresh(cfg.GetSchedule(), now)
		if !ok {
			continue
		}
		staleBefore := exp.Add(-grace)

		n, err := s.checkResultDS.UpdateFreshnessByScanConfig(sweeperCtx, cfg.GetScanConfigName(), staleBefore)
		if err != nil {
			log.Errorf("error updating freshness for scan configuration %q: %v", cfg.GetScanConfigName(), err)
			continue
		}
		if n > 0 {
			log.Infof("compliance freshness sweep: config %q updated %d results", cfg.GetScanConfigName(), n)
		}
	}
}
