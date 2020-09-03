package generate

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	clusterValidation "github.com/stackrox/rox/pkg/cluster"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

const (
	warningDeprecatedAdmissionControllerFlagSet = `WARNING: The --admission-controller flag has been renamed to --create-admission-controller and will be removed
in future versions of roxctl. Please use --create-admission-controller to suppress this warning
text and avoid breakages in the future.`
	errorDeprecatedAndNewAdmissionControllerFlagSet = `It is illegal to specify both the --admission-controller and --create-admission-controller flags.
Please use --create-admission-controller exclusively in all invocations.`
)

func k8s() *cobra.Command {
	c := &cobra.Command{
		Use: "k8s",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			cluster.Type = storage.ClusterType_KUBERNETES_CLUSTER
			if err := clusterValidation.Validate(&cluster); err.ToError() != nil {
				return err.ToError()
			}
			return fullClusterCreation(flags.Timeout(c))
		}),
	}
	return c
}
