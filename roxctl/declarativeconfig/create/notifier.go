package create

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/declarativeconfig/k8sobject"
	"gopkg.in/yaml.v3"
)

var (
	errInvalidPair = errox.InvalidArgs.New("invalid key-value pair")
)

func notifierCommand(cliEnvironment environment.Environment) *cobra.Command {
	notifierCommand := &notifierCmd{notifier: &declarativeconfig.Notifier{}, env: cliEnvironment}

	cmd := &cobra.Command{
		Use:   notifierCommand.notifier.Type(),
		Args:  cobra.NoArgs,
		RunE:  notifierCommand.RunE(),
		Short: "Create a declarative configuration for a notifier",
	}

	cmd.Flags().StringVar(&notifierCommand.notifier.Name, "name", "", "name of the notifier")

	genericFlags := pflag.NewFlagSet("generic", pflag.PanicOnError)
	notifierCommand.gc = &declarativeconfig.GenericConfig{}
	genericFlags.BoolVar(&notifierCommand.gc.AuditLoggingEnabled, "audit-logging", false, "Audit logging enabled")
	genericFlags.StringVar(&notifierCommand.gc.Endpoint, "endpoint", "", "Endpoint")
	genericFlags.StringVar(&notifierCommand.gc.Username, "username", "", "Username")
	genericFlags.StringVar(&notifierCommand.gc.Password, "password", "", "Password")
	genericFlags.StringVar(&notifierCommand.gc.CACertPEM, "cacert", "", "CA certificate file name")
	genericFlags.StringToStringVar(&notifierCommand.gcHeaders, "headers", nil, "Headers (comma separated key=value pairs)")
	genericFlags.StringToStringVar(&notifierCommand.gcExtraFields, "extra-fields", nil, "Extra fields (comma separated key=value pairs)")
	genericFlags.BoolVar(&notifierCommand.gc.SkipTLSVerify, "skip-tls-verify", false, "Skip TLS verification")
	notifierCommand.genericFlagSet = genericFlags

	splunkFlags := pflag.NewFlagSet("splunk", pflag.PanicOnError)
	notifierCommand.sc = &declarativeconfig.SplunkConfig{}
	splunkFlags.StringVar(&notifierCommand.sc.HTTPToken, "splunk-token", "", "Splunk HTTP token")
	splunkFlags.StringVar(&notifierCommand.sc.HTTPEndpoint, "splunk-endpoint", "", "Splunk HTTP endpoint")
	splunkFlags.BoolVar(&notifierCommand.sc.Insecure, "splunk-insecure", false, "Insecure connection to Splunk")
	splunkFlags.BoolVar(&notifierCommand.sc.AuditLoggingEnabled, "splunk-audit-logging", false, "Audit logging enabled")
	splunkFlags.Int64Var(&notifierCommand.sc.Truncate, "splunk-truncate", 0, "Splunk truncate limit")
	splunkFlags.StringToStringVar(&notifierCommand.scSourceTypes, "splunk-source-types", nil, "Splunk source types")
	notifierCommand.splunkFlagSet = splunkFlags

	cmd.Flags().AddFlagSet(genericFlags)
	cmd.Flags().AddFlagSet(splunkFlags)
	// No additional validation is required for notifiers, since a notifier
	// is valid when name is set, which is covered by requiring the flag.
	utils.Must(cmd.MarkFlagRequired("name"))
	return cmd
}

type notifierCmd struct {
	notifier *declarativeconfig.Notifier

	genericFlagSet *pflag.FlagSet
	splunkFlagSet  *pflag.FlagSet

	gc *declarativeconfig.GenericConfig
	sc *declarativeconfig.SplunkConfig

	gcHeaders     map[string]string
	gcExtraFields map[string]string
	scSourceTypes map[string]string

	env       environment.Environment
	configMap string
	secret    string
	namespace string
}

func (n *notifierCmd) RunE() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := n.Construct(cmd); err != nil {
			return err
		}
		return n.PrintYAML()
	}
}

func anyFlagChanged(fs *pflag.FlagSet) bool {
	var changed bool
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			changed = true
		}
	})
	return changed
}

func (n *notifierCmd) Construct(cmd *cobra.Command) error {
	configMap, secret, namespace, err := k8sobject.ReadK8sObjectFlags(cmd)
	if err != nil {
		return errors.Wrap(err, "reading config map flag values")
	}
	n.configMap = configMap
	n.secret = secret
	n.namespace = namespace

	if anyFlagChanged(n.genericFlagSet) {
		for k, v := range n.gcHeaders {
			n.gc.Headers = append(n.gc.Headers, declarativeconfig.KeyValuePair{k, v})
		}
		for k, v := range n.gcExtraFields {
			n.gc.ExtraFields = append(n.gc.ExtraFields, declarativeconfig.KeyValuePair{k, v})
		}
		n.notifier.GenericConfig = n.gc
	}

	if anyFlagChanged(n.splunkFlagSet) {
		for k, v := range n.scSourceTypes {
			n.sc.SourceTypes = append(n.sc.SourceTypes, declarativeconfig.SourceTypePair{k, v})
		}
		n.notifier.SplunkConfig = n.sc
	}
	return nil
}

func (n *notifierCmd) PrintYAML() error {
	yamlOutput := &bytes.Buffer{}
	enc := yaml.NewEncoder(yamlOutput)
	if err := enc.Encode(n.notifier); err != nil {
		return errors.Wrap(err, "creating the YAML output")
	}
	if n.configMap != "" || n.secret != "" {
		return errors.Wrap(k8sobject.WriteToK8sObject(context.Background(), n.configMap, n.secret, n.namespace,
			fmt.Sprintf("%s-%s", n.notifier.Type(), n.notifier.Name),
			yamlOutput.Bytes()), "writing the YAML output to config map")
	}
	if _, err := n.env.InputOutput().Out().Write(yamlOutput.Bytes()); err != nil {
		return errors.Wrap(err, "writing the YAML output")
	}
	return nil
}
