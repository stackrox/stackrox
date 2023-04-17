package generate

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	clusterValidation "github.com/stackrox/rox/pkg/cluster"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

const (
	errorAdmCntrlNotSupportedOnOpenShift3x  = "The --admission-controller-listen-on-events flag is not supported for OpenShift 3.11. Set --openshift-version=4 to indicate that you are deploying on OpenShift 4.x in order to use this flag."
	errorAuditLogsNotSupportedOnOpenShift3x = "The --disable-audit-logs flag is not supported for OpenShift 3.11. Set --openshift-version=4 to indicate that you are deploying on OpenShift 4.x in order to use this flag."
	noteOpenShift3xCompatibilityMode        = "Deployment files are generated in OpenShift 3.x compatibility mode. Set the --openshift-version flag to 3 to suppress this note, or to 4 to take advantage of OpenShift 4.x features."
)

type sensorGenerateOpenShiftCommand struct {
	*sensorGenerateCommand

	openshiftVersion          int
	admissionControllerEvents *bool
	disableAuditLogCollection *bool
}

func (s *sensorGenerateOpenShiftCommand) ConstructOpenShift() error {
	s.cluster.Type = storage.ClusterType_OPENSHIFT_CLUSTER
	switch s.openshiftVersion {
	case 0:
		s.env.Logger().InfofLn(noteOpenShift3xCompatibilityMode)
	case 3:
	case 4:
		s.cluster.Type = storage.ClusterType_OPENSHIFT4_CLUSTER
	default:
		return errox.InvalidArgs.Newf("invalid OpenShift version %d, supported values are '3' and '4'", s.openshiftVersion)
	}

	if s.admissionControllerEvents == nil {
		s.admissionControllerEvents = pointers.Bool(s.cluster.Type == storage.ClusterType_OPENSHIFT4_CLUSTER) // enable for OpenShift 4 only
	} else if *s.admissionControllerEvents && s.cluster.Type == storage.ClusterType_OPENSHIFT_CLUSTER {
		// The below `Validate` call would also catch this, but catching it here allows us to print more
		// CLI-relevant error messages that reference flag names.
		return errox.InvalidArgs.New(errorAdmCntrlNotSupportedOnOpenShift3x)
	}
	s.cluster.AdmissionControllerEvents = *s.admissionControllerEvents

	// This is intentionally NOT feature-flagged, because we always want to set the correct (auto) value,
	// even if we turn off the flag before shipping.
	if s.disableAuditLogCollection == nil {
		s.disableAuditLogCollection = pointers.Bool(s.cluster.Type != storage.ClusterType_OPENSHIFT4_CLUSTER)
	} else if !*s.disableAuditLogCollection && s.cluster.Type != storage.ClusterType_OPENSHIFT4_CLUSTER {
		return errox.InvalidArgs.New(errorAuditLogsNotSupportedOnOpenShift3x)
	}

	s.cluster.DynamicConfig.DisableAuditLogs = *s.disableAuditLogCollection

	return nil
}

func openshift(generateCmd *sensorGenerateCommand) *cobra.Command {
	openshiftCommand := sensorGenerateOpenShiftCommand{sensorGenerateCommand: generateCmd}
	c := &cobra.Command{
		Use:   "openshift",
		Short: "Generate the required files to deploy StackRox services into an OpenShift cluster.",
		Long:  "Generate the required YAML configuration files to deploy StackRox Sensor and Collector into an OpenShift cluster.",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			if err := openshiftCommand.ConstructOpenShift(); err != nil {
				return err
			}

			if err := clusterValidation.ValidatePartial(openshiftCommand.cluster).ToError(); err != nil {
				return err
			}
			return openshiftCommand.fullClusterCreation()
		}),
	}
	c.PersistentFlags().IntVar(&openshiftCommand.openshiftVersion, "openshift-version", 0, "OpenShift major version to generate deployment files for")
	flags.OptBoolFlagVarPF(c.PersistentFlags(), &openshiftCommand.admissionControllerEvents, "admission-controller-listen-on-events", "", "enable admission controller webhook to listen on Kubernetes events", "auto")
	flags.OptBoolFlagVarPF(c.PersistentFlags(), &openshiftCommand.disableAuditLogCollection, "disable-audit-logs", "", "disable audit log collection for runtime detection", "auto")

	return c
}
