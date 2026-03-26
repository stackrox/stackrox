package generate

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	clusterValidation "github.com/stackrox/rox/pkg/cluster"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

type sensorGenerateOpenShiftCommand struct {
	*sensorGenerateCommand

	openshiftVersion          int
	disableAuditLogCollection *bool
}

func (s *sensorGenerateOpenShiftCommand) ConstructOpenShift() error {
	s.cluster.Type = storage.ClusterType_OPENSHIFT4_CLUSTER
	switch s.openshiftVersion {
	case 0:
	case 4:
	default:
		return errox.InvalidArgs.Newf("invalid OpenShift version %d, currently only the value '4' is supported", s.openshiftVersion)
	}

	s.cluster.AdmissionControllerEvents = s.cluster.GetType() == storage.ClusterType_OPENSHIFT4_CLUSTER

	// This is intentionally NOT feature-flagged, because we always want to set the correct (auto) value,
	// even if we turn off the flag before shipping.
	if s.disableAuditLogCollection == nil {
		s.disableAuditLogCollection = pointers.Bool(s.cluster.GetType() != storage.ClusterType_OPENSHIFT4_CLUSTER)
	}

	s.cluster.DynamicConfig.DisableAuditLogs = *s.disableAuditLogCollection

	return nil
}

func openshift(generateCmd *sensorGenerateCommand) *cobra.Command {
	openshiftCommand := sensorGenerateOpenShiftCommand{sensorGenerateCommand: generateCmd}
	c := &cobra.Command{
		Use:   "openshift",
		Short: "Generate the required files to deploy StackRox services into an OpenShift cluster",
		Long:  "Generate the required YAML configuration files to deploy StackRox Sensor and Collector into an OpenShift cluster.",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			if err := openshiftCommand.ConstructOpenShift(); err != nil {
				return err
			}

			if err := clusterValidation.ValidatePartial(openshiftCommand.cluster).ToError(); err != nil {
				return errors.Wrap(err, "cluster validation failed")
			}
			return openshiftCommand.fullClusterCreation()
		}),
	}
	c.PersistentFlags().IntVar(&openshiftCommand.openshiftVersion, "openshift-version", 0, "OpenShift major version to generate deployment files for.")
	var ignoredBoolFlag bool
	c.PersistentFlags().BoolVar(&ignoredBoolFlag, "admission-controller-listen-on-events", true, "Enable admission controller webhook to listen on Kubernetes events.")
	utils.Must(c.PersistentFlags().MarkDeprecated("admission-controller-listen-on-events", WarningAdmissionControllerListenOnEventsSet))

	// Audit log collection should be enabled by default, disabled = false, as with the proto
	flags.OptBoolFlagVarPF(c.PersistentFlags(), &openshiftCommand.disableAuditLogCollection, "disable-audit-logs", "", "Disable audit log collection for runtime detection.", "false")

	return c
}
