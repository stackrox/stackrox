package generate

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	clusterValidation "github.com/stackrox/rox/pkg/cluster"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

const (
	warningNotSupportedOnAllOSVersions = `WARNING: The --admission-controller-listen-on-events is not supported for OpenShift 3.11, please ensure you are using OpenShift 4.0 or higher.`
)

func openshift() *cobra.Command {
	c := &cobra.Command{
		Use: "openshift",
		PersistentPreRunE: func(c *cobra.Command, _ []string) error {
			if c.PersistentFlags().Lookup("admission-controller-listen-on-events").Changed {
				fmt.Fprintf(os.Stderr, "%s\n\n", warningNotSupportedOnAllOSVersions)
			}
			return nil
		},
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			cluster.Type = storage.ClusterType_OPENSHIFT_CLUSTER
			if err := clusterValidation.Validate(&cluster).ToError(); err != nil {
				return err
			}
			return fullClusterCreation(flags.Timeout(c))
		}),
	}
	c.PersistentFlags().BoolVar(&cluster.AdmissionControllerEvents, "admission-controller-listen-on-events", false, "enable admission controller webhook to listen on Kubernetes events")
	return c
}
