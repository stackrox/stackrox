package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
)

const (
	// maxFinishedRunAge specifies the maximum age of a finished run before it will be flagged for deletion.
	maxFinishedRunAge = 12 * time.Hour
)

// runInstance is a run managed by the ComplianceManager. It is different from a run in the compliance framework,
// which only encompasses the execution of the checks. Instead, a `runInstance` is a run from start to finish,
// including data collection and results storage.
type runInstance struct {
	mutex sync.RWMutex

	id string

	domain        framework.ComplianceDomain
	standard      StandardImplementation
	scrapePromise *scrapePromise

	schedule *scheduleInstance

	ctx    context.Context
	cancel context.CancelFunc

	status                v1.ComplianceRun_State
	startTime, finishTime time.Time
	err                   error
}

func createRun(id string, domain framework.ComplianceDomain, standard StandardImplementation) *runInstance {
	r := &runInstance{
		id:       id,
		domain:   domain,
		standard: standard,
		status:   v1.ComplianceRun_READY,
	}

	r.ctx, r.cancel = context.WithCancel(context.Background())

	return r
}

func (r *runInstance) updateStatus(s v1.ComplianceRun_State) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.status = s
}

func (r *runInstance) Start(scrapePromise *scrapePromise, resultsStore store.Store) {
	go r.Run(scrapePromise, resultsStore)
}

func (r *runInstance) Run(scrapePromise *scrapePromise, resultsStore store.Store) {
	defer r.cancel()

	concurrency.WithLock(&r.mutex, func() {
		r.startTime = time.Now()
		r.status = v1.ComplianceRun_STARTED
	})
	defer concurrency.WithLock(&r.mutex, func() {
		r.finishTime = time.Now()
		r.status = v1.ComplianceRun_FINISHED
	})

	if r.schedule != nil {
		concurrency.WithLock(&r.schedule.mutex, func() {
			r.schedule.lastRun = r
		})
		defer concurrency.WithLock(&r.schedule.mutex, func() {
			r.schedule.lastFinishedRun = r
		})
	}

	err := r.doRun(scrapePromise, resultsStore)

	concurrency.WithLock(&r.mutex, func() {
		r.err = err
	})
}

func (r *runInstance) doRun(scrapePromise *scrapePromise, resultsStore store.Store) error {
	log.Infof("Starting compliance run %s for cluster %q and standard %q", r.id, r.domain.Cluster().Cluster().Name, r.standard.Standard.Name)

	r.updateStatus(v1.ComplianceRun_WAIT_FOR_DATA)
	data, err := scrapePromise.WaitForResult(r.ctx)
	if err != nil {
		return fmt.Errorf("waiting for compliance data: %v", err)
	}

	run, err := framework.NewComplianceRun(r.standard.Checks...)
	if err != nil {
		return fmt.Errorf("creating compliance run: %v", err)
	}

	r.updateStatus(v1.ComplianceRun_EVALUTING_CHECKS)
	if err := run.Run(r.ctx, r.domain, data); err != nil {
		return err
	}

	r.updateStatus(v1.ComplianceRun_STORING_RESULTS)
	results := r.collectResults(run)

	return resultsStore.StoreRunResults(results)
}

func timeToProto(t time.Time) *types.Timestamp {
	if t.IsZero() {
		return nil
	}
	tspb, _ := types.TimestampProto(t)
	return tspb
}

func (r *runInstance) standardID() string {
	return r.standard.Standard.ID
}

func (r *runInstance) ToProto() *v1.ComplianceRun {
	if r == nil {
		return nil
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var errorMessage string
	if r.status == v1.ComplianceRun_FINISHED && r.err != nil {
		errorMessage = r.err.Error()
	}

	proto := &v1.ComplianceRun{
		Id:           r.id,
		ClusterId:    r.domain.Cluster().Cluster().GetId(),
		StandardId:   r.standardID(),
		StartTime:    timeToProto(r.startTime),
		FinishTime:   timeToProto(r.finishTime),
		State:        r.status,
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

	if r.status != v1.ComplianceRun_FINISHED {
		return false
	}
	return time.Now().Sub(r.finishTime) > maxFinishedRunAge
}
