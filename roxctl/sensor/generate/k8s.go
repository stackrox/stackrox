package generate

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	clusterValidation "github.com/stackrox/rox/pkg/cluster"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/util"
)

const (
	WarningAdmissionControllerListenOnEventsSet = `The --admission-controller-listen-on-events flag has been deprecated and will be removed in future versions of roxctl. It will be ignored from version 4.9 onwards.`
)

type sensorGenerateK8sCommand struct {
	*sensorGenerateCommand
}

func (s *sensorGenerateK8sCommand) ConstructK8s() {
	s.cluster.SetType(storage.ClusterType_KUBERNETES_CLUSTER)
	s.cluster.GetDynamicConfig().SetDisableAuditLogs(true)
}

func k8s(generateCmd *sensorGenerateCommand) *cobra.Command {
	k8sCommand := sensorGenerateK8sCommand{sensorGenerateCommand: generateCmd}
	c := &cobra.Command{
		Use:   "k8s",
		Short: "Generate the required files to deploy StackRox services into a Kubernetes cluster",
		Long:  "Generate the required YAML configuration files to deploy StackRox Sensor, Collector, and Admission controller (optional) into a Kubernetes cluster.",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			k8sCommand.ConstructK8s()

			if err := clusterValidation.ValidatePartial(k8sCommand.cluster); err.ToError() != nil {
				return err.ToError()
			}
			return k8sCommand.fullClusterCreation()
		}),
	}

	var ignoredBoolFlag bool
	c.PersistentFlags().BoolVar(&ignoredBoolFlag, "admission-controller-listen-on-events", true, "Enable admission controller webhook to listen on Kubernetes events.")
	utils.Must(c.PersistentFlags().MarkDeprecated("admission-controller-listen-on-events", WarningAdmissionControllerListenOnEventsSet))

	return c
}
