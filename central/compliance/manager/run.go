package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/compliance/data"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/store"
	"github.com/stackrox/rox/central/scrape"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
)

type status int

const (
	// maxFinishedRunAge specifies the maximum age of a finished run before it will be flagged for deletion.
	maxFinishedRunAge = 12 * time.Hour

	ready status = iota
	started
	finished
)

// runInstance is a run managed by the ComplianceManager. It is different from a run in the compliance framework,
// which only encompasses the execution of the checks. Instead, a `runInstance` is a run from start to finish,
// including data collection and results storage.
type runInstance struct {
	mutex sync.RWMutex

	id         string
	standardID string

	domain framework.ComplianceDomain
	run    framework.ComplianceRun

	schedule *scheduleInstance

	ctx    context.Context
	cancel context.CancelFunc

	status                status
	startTime, finishTime time.Time
	err                   error
}

func createRun(id, standardID string, domain framework.ComplianceDomain, run framework.ComplianceRun) (*runInstance, error) {
	r := &runInstance{
		id:         id,
		standardID: standardID,

		domain: domain,
		run:    run,
	}

	r.ctx, r.cancel = context.WithCancel(context.Background())

	return r, nil
}

func (r *runInstance) Start(scrapeFactory scrape.Factory, dataRepoFactory data.RepositoryFactory, resultsStore store.Store) error {
	go r.Run(scrapeFactory, dataRepoFactory, resultsStore)
	return nil
}

func (r *runInstance) Run(scrapeFactory scrape.Factory, dataRepoFactory data.RepositoryFactory, resultsStore store.Store) {
	defer r.cancel()

	concurrency.WithLock(&r.mutex, func() {
		r.startTime = time.Now()
		r.status = started
	})
	defer concurrency.WithLock(&r.mutex, func() {
		r.finishTime = time.Now()
		r.status = finished
	})

	if r.schedule != nil {
		concurrency.WithLock(&r.schedule.mutex, func() {
			r.schedule.lastRun = r
		})
		defer concurrency.WithLock(&r.schedule.mutex, func() {
			r.schedule.lastFinishedRun = r
		})
	}

	err := r.doRun(scrapeFactory, dataRepoFactory, resultsStore)

	concurrency.WithLock(&r.mutex, func() {
		r.err = err
	})
}

func (r *runInstance) doRun(scrapeFactory scrape.Factory, dataRepoFactory data.RepositoryFactory, resultsStore store.Store) error {
	log.Infof("scraping results for standard %s and cluster %s", r.standardID, r.domain.Cluster().ID())
	scrapeResults, err := scrapeFactory.RunScrape(r.domain, r.ctx)
	if err != nil {
		return fmt.Errorf("scraping results: %v", err)
	}
	log.Infof("done scraping results for standard %s and cluster %s", r.standardID, r.domain.Cluster().ID())

	data, err := dataRepoFactory.CreateDataRepository(r.domain, scrapeResults)
	if err != nil {
		return fmt.Errorf("aggregating data: %v", err)
	}

	log.Infof("running checks for standard %s and cluster %s", r.standardID, r.domain.Cluster().ID())
	if err := r.run.Run(r.ctx, r.domain, data); err != nil {
		return err
	}
	log.Infof("done running checks for %s and cluster %s", r.standardID, r.domain.Cluster().ID())
	results := r.collectResults()
	return resultsStore.StoreRunResults(results)
}

func timeToProto(t time.Time) *types.Timestamp {
	if t.IsZero() {
		return nil
	}
	tspb, _ := types.TimestampProto(t)
	return tspb
}

func (r *runInstance) ToProto() *v1.ComplianceRun {
	if r == nil {
		return nil
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var errorMessage string
	var state v1.ComplianceRun_State
	switch r.status {
	case ready:
		state = v1.ComplianceRun_READY
	case started:
		state = v1.ComplianceRun_RUNNING
	case finished:
		state = v1.ComplianceRun_FINISHED
		if r.err != nil {
			errorMessage = r.err.Error()
		}
	}

	proto := &v1.ComplianceRun{
		Id:           r.id,
		ClusterId:    r.domain.Cluster().Cluster().GetId(),
		StandardId:   r.standardID,
		StartTime:    timeToProto(r.startTime),
		FinishTime:   timeToProto(r.finishTime),
		State:        state,
		ErrorMessage: errorMessage,
	}
	if r.schedule != nil {
		proto.ScheduleId = r.schedule.id
	}
	return proto
}

func (r *runInstance) shouldDelete() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.status != finished {
		return false
	}
	return time.Now().Sub(r.finishTime) > maxFinishedRunAge
}
