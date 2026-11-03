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

	// sweeperCtx is elevated because the sweeper is a background loop with no user identity: it must
	// read every scan configuration and mark results stale across all clusters.
	sweeperCtx = sac.WithAllAccess(context.Background())
)

// Sweeper periodically marks compliance check results STALE when their scan configuration's
// schedule says they should have been refreshed by now but were not. It only ever transitions rows
// to STALE; CURRENT is owned by the ingest path and nil-timestamp rows stay UNKNOWN.
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

// New returns a Sweeper that ticks at the given interval.
func New(scanConfigDS scanConfigDS.DataStore, checkResultDS checkResultDS.DataStore, interval time.Duration) Sweeper {
	return &sweeperImpl{
		interval:      interval,
		scanConfigDS:  scanConfigDS,
		checkResultDS: checkResultDS,
		stopper:       concurrency.NewStopper(),
	}
}

// Start starts the freshness sweeper loop.
func (s *sweeperImpl) Start() {
	s.ticker = time.NewTicker(s.interval)
	go s.loop()
}

// Stop stops the freshness sweeper loop.
func (s *sweeperImpl) Stop() {
	s.stopper.Client().Stop()
	_ = s.stopper.Client().Stopped().Wait()
}

func (s *sweeperImpl) loop() {
	defer s.stopper.Flow().ReportStopped()
	defer s.ticker.Stop()

	go s.sweep()
	for {
		select {
		case <-s.stopper.Flow().StopRequested():
			return
		case <-s.ticker.C:
			s.sweep()
		}
	}
}

// sweep marks STALE the check results of every scan configuration whose schedule expected a refresh
// more than one grace window ago but that were not refreshed since.
func (s *sweeperImpl) sweep() {
	select {
	case <-s.stopper.Flow().StopRequested():
		return
	default:
	}

	now := time.Now()
	// grace is the combined budget the watchers allow for a scheduled scan to complete and have its
	// results ingested. A result older than (expected refresh - grace) is past due, hence stale.
	grace := env.ComplianceScanScheduleWatcherTimeout.DurationSetting() + env.ComplianceScanWatcherTimeout.DurationSetting()

	configs, err := s.scanConfigDS.GetScanConfigurations(sweeperCtx, search.EmptyQuery())
	if err != nil {
		log.Errorf("compliance freshness sweeper: listing scan configurations: %v", err)
		return
	}

	for _, cfg := range configs {
		exp, ok := expectedRefresh(cfg.GetSchedule(), now)
		if !ok {
			continue
		}
		olderThan := exp.Add(-grace)
		n, err := s.checkResultDS.MarkScanChecksStaleByConfig(sweeperCtx, cfg.GetScanConfigName(), olderThan)
		if err != nil {
			log.Errorf("compliance freshness sweeper: marking results stale for scan configuration %q: %v", cfg.GetScanConfigName(), err)
			continue
		}
		if n > 0 {
			log.Infof("compliance freshness sweeper: marked %d check result(s) stale for scan configuration %q", n, cfg.GetScanConfigName())
		}
	}
}

// expectedRefresh returns the most recent time strictly before now at which sched expected a scan to
// start, and true when that time could be computed. It returns false for a nil schedule, a schedule
// that does not convert to a cron expression (e.g. an UNSET interval), or an unreachable previous
// tick.
func expectedRefresh(sched *storage.Schedule, now time.Time) (time.Time, bool) {
	if sched == nil {
		return time.Time{}, false
	}
	cron, err := schedule.ConvertToCronTab(sched)
	if err != nil || cron == "" {
		return time.Time{}, false
	}
	// inclRefTime=false: we want the previous tick strictly before now, not now itself.
	prev, err := gronx.PrevTickBefore(cron, now, false)
	if err != nil {
		return time.Time{}, false
	}
	return prev, true
}
