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
	"github.com/stackrox/rox/generated/storage"
	pkgStandards "github.com/stackrox/rox/pkg/compliance/checks/standards"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
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

	r.ctx, r.cancel = context.WithCancel(sac.WithAllAccess(context.Background()))

	return r
}

func (r *runInstance) updateStatus(s v1.ComplianceRun_State) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.status = s
}

func (r *runInstance) Start(dataPromise dataPromise, resultsStore complianceDS.DataStore) {
	go r.Run(dataPromise, resultsStore)
}

func (r *runInstance) Run(dataPromise dataPromise, resultsStore complianceDS.DataStore) {
	defer r.cancel()

	if r.schedule != nil {
		concurrency.WithLock(&r.schedule.mutex, func() {
			r.schedule.lastRun = r
		})
		defer concurrency.WithLock(&r.schedule.mutex, func() {
			r.schedule.lastFinishedRun = r
		})
	}

	run, nodeResults, err := r.doRun(dataPromise)
	defer concurrency.WithLock(&r.mutex, func() {
		r.finishTime = time.Now()
		r.status = v1.ComplianceRun_FINISHED
	})

	if err == nil {
		results := r.collectResults(run)

		if features.ComplianceInNodes.Enabled() {
			r.foldRemoteResults(results, nodeResults)
		}
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

func (r *runInstance) foldRemoteResults(results *storage.ComplianceRunResults, nodeResults map[string]map[string]*compliance.ComplianceStandardResult) {
	if results.NodeResults == nil {
		results.NodeResults = make(map[string]*storage.ComplianceRunResults_EntityResults)
	}

	mergedClusterResults := make(map[string]*storage.ComplianceResultValue)
	for _, node := range r.domain.Nodes() {
		standardResults := r.getStandardResults(node.Node().GetName(), nodeResults)
		if standardResults == nil {
			continue
		}

		// Merge the cluster-level results into a single map of check ID -> check result
		mergeClusterResults(mergedClusterResults, standardResults.GetClusterCheckResults())

		// Fold in each of the node-level results individually
		nodeID := node.ID()
		currentNode, ok := results.NodeResults[nodeID]
		if !ok {
			results.NodeResults[nodeID] = &storage.ComplianceRunResults_EntityResults{
				ControlResults: standardResults.GetNodeCheckResults(),
			}
			continue
		}
		combineResultSets(currentNode.GetControlResults(), standardResults.GetNodeCheckResults())
	}

	// Add notes for any missing cluster-level checks
	r.noteMissingNodeClusterChecks(mergedClusterResults)
	// Finally, combine all the cluster-level checks into the final result data
	combineResultSets(results.GetClusterResults().GetControlResults(), mergedClusterResults)
}

func mergeClusterResults(destination, source map[string]*storage.ComplianceResultValue) {
	for checkName, sourceComplianceResult := range source {
		destinationComplianceResult, ok := destination[checkName]
		if !ok {
			destination[checkName] = sourceComplianceResult
			continue
		}
		destinationComplianceResult.Evidence = append(destinationComplianceResult.GetEvidence(), sourceComplianceResult.GetEvidence()...)
		if sourceComplianceResult.GetOverallState() > destinationComplianceResult.GetOverallState() {
			destinationComplianceResult.OverallState = sourceComplianceResult.GetOverallState()
		}
	}
}

func (r *runInstance) getStandardResults(nodeName string, nodeResults map[string]map[string]*compliance.ComplianceStandardResult) *compliance.ComplianceStandardResult {
	perStandardNodeResults, ok := nodeResults[nodeName]
	if !ok {
		return nil
	}

	standardResults, ok := perStandardNodeResults[r.standard.ID]
	if !ok {
		log.Infof("no check results received from node %s for compliance standard %s", nodeName, r.standard.ID)
		return nil
	}
	return standardResults
}

func combineResultSets(destination, source map[string]*storage.ComplianceResultValue) {
	// Fold evidence in per-check.  We can't just override the results because some of the checks may have been run on Central
	for checkName, remoteComplianceResult := range source {
		destination[checkName] = remoteComplianceResult
	}
}

func (r *runInstance) noteMissingNodeClusterChecks(clusterResults map[string]*storage.ComplianceResultValue) {
	standard, ok := pkgStandards.NodeChecks[r.standard.ID]
	if !ok {
		return
	}

	for checkName, checkAndMetadata := range standard {
		if checkAndMetadata.Metadata.TargetKind != pkgFramework.ClusterKind {
			continue
		}

		// Only assign a value to a nil clusterResults after we know there is supposed to be evidence
		if clusterResults == nil {
			clusterResults = map[string]*storage.ComplianceResultValue{}
		}

		if evidence, ok := clusterResults[checkName]; !ok || len(evidence.GetEvidence()) == 0 {
			clusterResults[checkName] = &storage.ComplianceResultValue{
				Evidence: []*storage.ComplianceResultValue_Evidence{
					{
						State:   storage.ComplianceState_COMPLIANCE_STATE_NOTE,
						Message: "No evidence was received for this check. This can occur when using a managed Kubernetes service or if the compliance pods are not running on the master nodes.",
					},
				},
				OverallState: storage.ComplianceState_COMPLIANCE_STATE_NOTE,
			}
		}
	}
}

func (r *runInstance) doRun(dataPromise dataPromise) (framework.ComplianceRun, map[string]map[string]*compliance.ComplianceStandardResult, error) {
	concurrency.WithLock(&r.mutex, func() {
		r.startTime = time.Now()
		r.status = v1.ComplianceRun_STARTED
	})

	log.Infof("Starting compliance run %s for cluster %q and standard %q", r.id, r.domain.Cluster().Cluster().Name, r.standard.Standard.Name)

	r.updateStatus(v1.ComplianceRun_WAIT_FOR_DATA)
	data, err := dataPromise.WaitForResult(r.ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "waiting for compliance data")
	}

	run, err := framework.NewComplianceRun(r.standard.AllChecks()...)
	if err != nil {
		return nil, nil, errors.Wrap(err, "creating compliance run")
	}

	log.Infof("Starting evaluating checks for run %s for cluster %q and standard %q", r.id, r.domain.Cluster().Cluster().Name, r.standard.Standard.Name)
	r.updateStatus(v1.ComplianceRun_EVALUTING_CHECKS)

	if err := run.Run(r.ctx, r.domain, data); err != nil {
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
