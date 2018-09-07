package federationcontrollermanager

import (
	"github.com/stackrox/rox/benchmarks/checks"
	"github.com/stackrox/rox/benchmarks/checks/utils"
)

const process = "federation-controller-manager"

var configFunc = utils.GetKubeFederationControllerManagerConfig

func newKubernetesSchedulerCheck(check *utils.CommandCheck) utils.Check {
	check.Process = process
	check.ConfigGetter = configFunc
	return check
}

// NewProfiling implements CIS Kubernetes v1.2.0 3.2.1
func NewProfiling() utils.Check {
	return newKubernetesSchedulerCheck(&utils.CommandCheck{
		Name:        "CIS Kubernetes v1.2.0 - 3.2.1",
		Description: "Ensure that the --profiling argument is set to false",

		Field:        "profiling",
		Default:      "true",
		EvalFunc:     utils.Matches,
		DesiredValue: "false",
	})
}

func init() {
	checks.AddToRegistry(NewProfiling())
}
