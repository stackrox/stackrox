package generate

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	clusterValidation "github.com/stackrox/rox/pkg/cluster"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/utils"
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
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			// Migration process for renaming "--admission-controller" parameter to "--create-admission-controller".
			// Can be removed in a future release.
			if c.PersistentFlags().Lookup("admission-controller").Changed && c.PersistentFlags().Lookup("create-admission-controller").Changed {
				// Add extra newline to delimit the warning from regular program output.
				fmt.Fprintln(os.Stderr, errorDeprecatedAndNewAdmissionControllerFlagSet)
				return errors.New("Specified deprecated flag --admission-controller and new flag --create-admission-controller at the same time")
			}
			if c.PersistentFlags().Lookup("admission-controller").Changed {
				// Add extra newline to delimit the warning from regular program output.
				fmt.Fprintf(os.Stderr, "%s\n\n", warningDeprecatedAdmissionControllerFlagSet)
			}

			return nil
		},
	}

	c.PersistentFlags().BoolVar(&cluster.AdmissionController, "admission-controller", false, "whether or not to use an admission controller for enforcement")
	c.PersistentFlags().BoolVar(&cluster.AdmissionController, "create-admission-controller", false, "whether or not to use an admission controller for enforcement")
	utils.Must(c.PersistentFlags().MarkHidden("admission-controller"))
	if features.AdmissionControlEnforceOnUpdate.Enabled() {
		c.PersistentFlags().BoolVar(&cluster.AdmissionControllerUpdates, "admission-controller-listen-on-updates", false, "whether or not to configure the admission controller webhook to listen on object updates")
	}

	// Admission controller config
	ac := cluster.DynamicConfig.AdmissionControllerConfig
	c.PersistentFlags().BoolVar(&ac.Enabled, "admission-controller-enabled", false, "dynamic enable for the admission controller")
	c.PersistentFlags().Int32Var(&ac.TimeoutSeconds, "admission-controller-timeout", 3, "timeout in seconds for the admission controller")
	c.PersistentFlags().BoolVar(&ac.ScanInline, "admission-controller-scan-inline", false, "get scans inline when using the admission controller")
	c.PersistentFlags().BoolVar(&ac.DisableBypass, "admission-controller-disable-bypass", false, "disable the bypass annotations for the admission controller")
	if features.AdmissionControlEnforceOnUpdate.Enabled() {
		c.PersistentFlags().BoolVar(&ac.EnforceOnUpdates, "admission-controller-enforce-on-updates", false, "dynamic enable for enforcing on object updates in the admission controller")
	}

	return c
}
