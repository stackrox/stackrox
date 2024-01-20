package runner

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sensorupgrader"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/upgrader/bundle"
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	log = logging.LoggerForModule()
)

// A Runner runs a workflow.
type Runner interface {
	Err() error
	Finished() bool
	// MostRecentStage returns the most recent stage that the runner executed
	// -- either successfully, or unsuccessfully. If it was unsuccessful,
	// Err() is guaranteed to be non-nil.
	MostRecentStage() sensorupgrader.Stage
	// RunNextStage runs the next stage of the runner.
	// Callers MUST check r.Finished() and r.Error() before calling this.
	RunNextStage()
}

// A runner encapsulates the requisite state and logic for running a specific workflow.
type runner struct {
	// "Spec" fields.
	ctx      *upgradectx.UpgradeContext
	workflow string

	// Derived "spec" fields.
	stagesToExecute []sensorupgrader.Stage

	// "Status" fields.
	mostRecentlyExecutedStageIdx int
	err                          error
	preUpgradeObjs               []*unstructured.Unstructured
	preUpgradeState              map[k8sobjects.ObjectRef]*unstructured.Unstructured
	bundleContents               bundle.Contents
	postUpgradeObjs              []*unstructured.Unstructured
	executionPlan                *plan.ExecutionPlan
}

// New returns a new runner that is responsible for running the given workflow.
func New(ctx *upgradectx.UpgradeContext, workflow string) (Runner, error) {
	return newRunner(ctx, workflow)
}

func newRunner(ctx *upgradectx.UpgradeContext, workflow string) (*runner, error) {
	r := &runner{
		ctx:      ctx,
		workflow: workflow,
	}

	workflowStages := sensorupgrader.Workflows()[r.workflow]
	if workflowStages == nil {
		return nil, errors.Errorf("invalid workflow %q", r.workflow)
	}
	r.stagesToExecute = workflowStages
	r.mostRecentlyExecutedStageIdx = -1

	return r, nil
}

func (r *runner) runFullWorkflow() error {
	log.Infof("====== Running workflow %s ======", r.workflow)

	for !r.Finished() {
		r.RunNextStage()
		if err := r.Err(); err != nil {
			return err
		}
	}

	log.Infof("====== Workflow %s terminated successfully ======", r.workflow)
	return nil
}

func (r *runner) MostRecentStage() sensorupgrader.Stage {
	if r.mostRecentlyExecutedStageIdx < 0 {
		return sensorupgrader.UnsetStage
	}
	return r.stagesToExecute[r.mostRecentlyExecutedStageIdx]
}

func (r *runner) Finished() bool {
	return r.Err() == nil && r.mostRecentlyExecutedStageIdx >= len(r.stagesToExecute)-1
}

func (r *runner) Err() error {
	return r.err
}

func (r *runner) RunNextStage() {
	if r.Err() != nil {
		utils.Should(errors.Wrap(r.Err(), "cannot run next stage; runner is in error"))
		return
	}
	if r.Finished() {
		utils.Should(errors.New("cannot run next stage; runner is finished"))
		return
	}

	r.mostRecentlyExecutedStageIdx++
	stage := r.stagesToExecute[r.mostRecentlyExecutedStageIdx]
	stageDesc := r.Stages()[stage]
	log.Infof("---- %s ----", stageDesc.description)
	if err := stageDesc.run(); err != nil {
		r.err = errors.Wrapf(err, "executing stage %q", stageDesc.description)
		return
	}
}
