package runner

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sensorupgrader"
	"github.com/stackrox/rox/sensor/upgrader/bundle"
	"github.com/stackrox/rox/sensor/upgrader/k8sobjects"
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

var (
	log = logging.LoggerForModule()
)

type runner struct {
	ctx *upgradectx.UpgradeContext

	preUpgradeObjs  []k8sobjects.Object
	preUpgradeState map[k8sobjects.ObjectRef]k8sobjects.Object
	bundleContents  bundle.Contents
	postUpgradeObjs []k8sobjects.Object
	executionPlan   *plan.ExecutionPlan
}

func (r *runner) Run(workflow string) error {
	workflowStages := sensorupgrader.Workflows()[workflow]

	if workflowStages == nil {
		return errors.Errorf("invalid workflow %q", workflow)
	}

	log.Infof("====== Running workflow %s ======", workflow)

	stagesByID := r.Stages()
	for _, stageID := range workflowStages {
		stageDesc := stagesByID[stageID]
		log.Infof("---- %s ----", stageDesc.description)
		if err := stageDesc.run(); err != nil {
			log.Errorf(err.Error())
			return err
		}
	}

	log.Infof("====== Workflow %s terminated successfully ======", workflow)

	return nil
}
