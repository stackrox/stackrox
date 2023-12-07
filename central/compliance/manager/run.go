package manager

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	complianceDS "github.com/stackrox/rox/central/compliance/datastore"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

// runInstance is a run managed by the ComplianceManager. It is different from a run in the compliance framework,
// which only encompasses the execution of the checks. Instead, a `runInstance` is a run from start to finish,
// including data collection and results storage.
type runInstance struct {
	mutex sync.RWMutex

	id string

	domain   framework.ComplianceDomain
	standard *standards.Standard

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

	r.ctx, r.cancel = context.WithCancel(sac.WithAllAccess(context.Background()))

	return r
}

func (r *runInstance) updateStatus(s v1.ComplianceRun_State) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.status = s
}

func (r *runInstance) Start(perClusterSemaphore *clusterBasedSemaphore, dataPromise dataPromise, resultsStore complianceDS.DataStore) {
	go r.Run(perClusterSemaphore, dataPromise, resultsStore)
}

func (r *runInstance) Run(perClusterSemaphore *clusterBasedSemaphore, dataPromise dataPromise, resultsStore complianceDS.DataStore) {
	defer r.cancel()

	run, nodeResults, err := r.doRun(perClusterSemaphore, dataPromise)
	defer concurrency.WithLock(&r.mutex, func() {
		r.finishTime = time.Now()
		r.status = v1.ComplianceRun_FINISHED
	})

	if err == nil {
		results := r.collectResults(run, nodeResults)

		if storeErr := resultsStore.StoreRunResults(r.ctx, results); storeErr != nil {
			err = errors.Wrap(storeErr, "storing results")
		}
	}
	if err != nil {
		concurrency.WithLock(&r.mutex, func() {
			r.err = err
		})
		metadata := r.metadataProto(true)
		if storeErr := resultsStore.StoreFailure(r.ctx, metadata); storeErr != nil {
			log.Errorf("Failed to store metadata for failed compliance run: %v", storeErr)
		}
		log.Errorf("Compliance run %s for standard %s on cluster %s failed: %v", r.id, r.standard.Name, r.domain.Cluster().ID(), err)
	}
	log.Infof("Completed compliance run %s", r.id)
}

func (r *runInstance) doRun(perClusterSemaphore *clusterBasedSemaphore, dataPromise dataPromise) (framework.ComplianceRun, map[string]map[string]*compliance.ComplianceStandardResult, error) {
	if err := perClusterSemaphore.Acquire(r.ctx, 1); err != nil {
		return nil, nil, err
	}
	defer perClusterSemaphore.Release(1)

	concurrency.WithLock(&r.mutex, func() {
		r.startTime = time.Now()
		r.status = v1.ComplianceRun_STARTED
	})

	log.Infof("Starting compliance run %s for cluster %q and standard %q", r.id, r.domain.Cluster().Cluster().Name, r.standard.Standard.Name)

	log.Infof("Sleeping for 1 minute to emulate slowness")
	time.Sleep(1 * time.Minute)

	r.updateStatus(v1.ComplianceRun_WAIT_FOR_DATA)
	data, err := dataPromise.WaitForResult(r.ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "waiting for compliance data")
	}

	run, err := framework.NewComplianceRun(r.standard.AllChecks()...)
	if err != nil {
		return nil, nil, errors.Wrap(err, "creating compliance run")
	}

	log.Infof("Starting evaluating checks (%d checks total) for run %s for cluster %q and standard %q", len(r.standard.AllChecks()), r.id, r.domain.Cluster().Cluster().Name, r.standard.Standard.Name)
	r.updateStatus(v1.ComplianceRun_EVALUTING_CHECKS)

	if err := run.Run(r.ctx, r.standard.Name, r.domain, data); err != nil {
		log.Errorf("Error evaluating checks for run %s for cluster %q and standard %q: %v", r.id, r.domain.Cluster().Cluster().Name, r.standard.Standard.Name, err)
		return nil, nil, err
	}
	log.Infof("Successfully evaluated checks for run %s for cluster %q and standard %q", r.id, r.domain.Cluster().Cluster().Name, r.standard.Standard.Name)
	return run, data.NodeResults(), nil
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
	return proto
}
