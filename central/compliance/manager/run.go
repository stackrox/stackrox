package manager

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/compliance/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
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

	domain   framework.ComplianceDomain
	standard *standards.Standard

	schedule *scheduleInstance

	ctx    context.Context
	cancel context.CancelFunc

	status                v1.ComplianceRun_State
	startTime, finishTime time.Time
	err                   error
}

func createRun(id string, domain framework.ComplianceDomain, standard *standards.Standard) *runInstance {
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

func (r *runInstance) Start(dataPromise dataPromise, resultsStore store.Store) {
	go r.Run(dataPromise, resultsStore)
}

func (r *runInstance) Run(dataPromise dataPromise, resultsStore store.Store) {
	defer r.cancel()

	if r.schedule != nil {
		concurrency.WithLock(&r.schedule.mutex, func() {
			r.schedule.lastRun = r
		})
		defer concurrency.WithLock(&r.schedule.mutex, func() {
			r.schedule.lastFinishedRun = r
		})
	}

	run, err := r.doRun(dataPromise)

	if err == nil {
		results := r.collectResults(run)
		if storeErr := resultsStore.StoreRunResults(results); storeErr != nil {
			err = errors.Wrap(err, "storing results")
		}
	}
	if err != nil {
		concurrency.WithLock(&r.mutex, func() {
			r.err = err
		})
		metadata := r.metadataProto(true)
		if storeErr := resultsStore.StoreFailure(metadata); storeErr != nil {
			log.Errorf("Failed to store metadata for failed compliance run: %v", storeErr)
		}
	}
}

func (r *runInstance) doRun(dataPromise dataPromise) (framework.ComplianceRun, error) {
	concurrency.WithLock(&r.mutex, func() {
		r.startTime = time.Now()
		r.status = v1.ComplianceRun_STARTED
	})
	defer concurrency.WithLock(&r.mutex, func() {
		r.finishTime = time.Now()
		r.status = v1.ComplianceRun_FINISHED
	})

	log.Infof("Starting compliance run %s for cluster %q and standard %q", r.id, r.domain.Cluster().Cluster().Name, r.standard.Standard.Name)

	r.updateStatus(v1.ComplianceRun_WAIT_FOR_DATA)
	data, err := dataPromise.WaitForResult(r.ctx)
	if err != nil {
		return nil, errors.Wrap(err, "waiting for compliance data")
	}

	run, err := framework.NewComplianceRun(r.standard.AllChecks()...)
	if err != nil {
		return nil, errors.Wrap(err, "creating compliance run")
	}

	r.updateStatus(v1.ComplianceRun_EVALUTING_CHECKS)
	return run, run.Run(r.ctx, r.domain, data)
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
	return time.Since(r.finishTime) > maxFinishedRunAge
}
