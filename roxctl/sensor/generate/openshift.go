package generate

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	clusterValidation "github.com/stackrox/rox/pkg/cluster"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

const (
	errorAdmCntrlNotSupportedOnOpenShift3x  = `ERROR: The --admission-controller-listen-on-events flag is not supported for OpenShift 3.11. Set --openshift-version=4 to indicate that you are deploying on OpenShift 4.x in order to use this flag.`
	errorAuditLogsNotSupportedOnOpenShift3x = `ERROR: The --disable-audit-logs flag is not supported for OpenShift 3.11. Set --openshift-version=4 to indicate that you are deploying on OpenShift 4.x in order to use this flag.`
	noteOpenShift3xCompatibilityMode        = `NOTE: Deployment files are generated in OpenShift 3.x compatibility mode. Set the --openshift-version flag to 3 to suppress this note, or to 4 to take advantage of OpenShift 4.x features.`
)

func openshift() *cobra.Command {
	var openshiftVersion int
	var admissionControllerEvents *bool
	var disableAuditLogCollection *bool

	c := &cobra.Command{
		Use: "openshift",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			cluster.Type = storage.ClusterType_OPENSHIFT_CLUSTER
			switch openshiftVersion {
			case 0:
				logger.WarnfLn(noteOpenShift3xCompatibilityMode)
			case 3:
			case 4:
				cluster.Type = storage.ClusterType_OPENSHIFT4_CLUSTER
			default:
				return errors.Errorf("invalid OpenShift version %d, supported values are '3' and '4'", openshiftVersion)
			}

			if admissionControllerEvents == nil {
				admissionControllerEvents = pointers.Bool(cluster.Type == storage.ClusterType_OPENSHIFT4_CLUSTER) // enable for OpenShift 4 only
			} else if *admissionControllerEvents && cluster.Type == storage.ClusterType_OPENSHIFT_CLUSTER {
				// The below `Validate` call would also catch this, but catching it here allows us to print more
				// CLI-relevant error messages that reference flag names.
				logger.ErrfLn(errorAdmCntrlNotSupportedOnOpenShift3x)
				return errors.New("incompatible flag settings")
			}
			cluster.AdmissionControllerEvents = *admissionControllerEvents

			// This is intentionally NOT feature-flagged, because we always want to set the correct (auto) value,
			// even if we turn off the flag before shipping.
			if disableAuditLogCollection == nil {
				disableAuditLogCollection = pointers.Bool(cluster.Type != storage.ClusterType_OPENSHIFT4_CLUSTER)
			} else if !*disableAuditLogCollection && cluster.Type != storage.ClusterType_OPENSHIFT4_CLUSTER {
				logger.ErrfLn(errorAuditLogsNotSupportedOnOpenShift3x)
				return errors.New("incompatible flag settings")
			}
			cluster.DynamicConfig.DisableAuditLogs = *disableAuditLogCollection

			if err := clusterValidation.ValidatePartial(&cluster).ToError(); err != nil {
				return err
			}
			return fullClusterCreation(flags.Timeout(c))
		}),
	}
	c.PersistentFlags().IntVar(&openshiftVersion, "openshift-version", 0, "OpenShift major version to generate deployment files for")
	flags.OptBoolFlagVarPF(c.PersistentFlags(), &admissionControllerEvents, "admission-controller-listen-on-events", "", "enable admission controller webhook to listen on Kubernetes events", "auto")
	flags.OptBoolFlagVarPF(c.PersistentFlags(), &disableAuditLogCollection, "disable-audit-logs", "", "disable audit log collection for runtime detection", "auto")

	return c
}
