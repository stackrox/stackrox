package create

import (
	"bytes"
	"context"
	"encoding/pem"
	"fmt"
	"net/url"
	"os"
	"sort"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/declarativeconfig/k8sobject"
	"github.com/stackrox/rox/roxctl/declarativeconfig/lint"
	"gopkg.in/yaml.v3"
)

func notifierCommand(cliEnvironment environment.Environment) *cobra.Command {
	notifierCommand := &notifierCmd{notifier: &declarativeconfig.Notifier{}, env: cliEnvironment}

	cmd := &cobra.Command{
		Use:   notifierCommand.notifier.ConfigurationType(),
		Args:  cobra.NoArgs,
		Short: "Commands to create a declarative configuration for a notifier",
	}

	cmd.PersistentFlags().StringVar(&notifierCommand.notifier.Name, "name", "", "Name of the notifier")
	cmd.AddCommand(
		notifierCommand.genericCommand(),
		notifierCommand.splunkCommand())

	utils.Must(cmd.MarkPersistentFlagRequired("name"))
	return cmd
}

func (n *notifierCmd) genericCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "generic",
		Args:  cobra.NoArgs,
		RunE:  n.runE(),
		Short: "Create a declarative configuration for a generic notifier",
	}
	genericFlags := cmd.Flags()
	n.gc = &declarativeconfig.GenericConfig{}
	genericFlags.BoolVar(&n.gc.AuditLoggingEnabled, "audit-logging", false,
		"Audit logging enabled")
	genericFlags.StringVar(&n.gc.Endpoint, "webhook-endpoint", "",
		"Webhook endpoint URL")
	genericFlags.StringVar(&n.gc.Username, "webhook-username", "",
		"Username for the webhook endpoint basic authentication. "+
			"No authentication if not provided. Requires --webhook-password")
	genericFlags.StringVar(&n.gc.Password, "webhook-password", "",
		"Password for the webhook endpoint basic authentication. "+
			"No authentication if not provided. Requires --webhook-username")
	genericFlags.StringVar(&n.gc.CACertPEM, "webhook-cacert-file", "",
		"Endpoint CA certificate file name (PEM format)")
	genericFlags.StringToStringVar(&n.gcHeaders, "headers", nil,
		"Headers (comma separated key=value pairs)")
	genericFlags.StringToStringVar(&n.gcExtraFields, "extra-fields", nil,
		"Extra fields (comma separated key=value pairs)")
	genericFlags.BoolVar(&n.gc.SkipTLSVerify, "webhook-skip-tls-verify", false,
		"Skip webhook TLS verification")
	n.genericFlagSet = genericFlags

	cmd.Flags().AddFlagSet(genericFlags)

	cmd.MarkFlagsRequiredTogether("webhook-username", "webhook-password")
	utils.Must(cmd.MarkFlagFilename("webhook-cacert-file"))
	utils.Must(cmd.MarkFlagRequired("webhook-endpoint"))
	return cmd
}

func (n *notifierCmd) splunkCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "splunk",
		Args:  cobra.NoArgs,
		RunE:  n.runE(),
		Short: "Create a declarative configuration for a Splunk notifier",
	}

	splunkFlags := cmd.Flags()
	n.sc = &declarativeconfig.SplunkConfig{}
	splunkFlags.BoolVar(&n.sc.AuditLoggingEnabled, "audit-logging", false,
		"Audit logging enabled")
	splunkFlags.StringVar(&n.sc.HTTPToken, "splunk-token", "",
		"Splunk HTTP token (required)")
	splunkFlags.StringVar(&n.sc.HTTPEndpoint, "splunk-endpoint", "",
		"Splunk HTTP endpoint (required)")
	splunkFlags.BoolVar(&n.sc.Insecure, "splunk-skip-tls-verify", false,
		"Insecure connection to Splunk")
	splunkFlags.Int64Var(&n.sc.Truncate, "truncate", 0,
		"Splunk truncate limit (default 10000)")
	splunkFlags.StringToStringVar(&n.scSourceTypes, "source-types", nil,
		"Splunk source types (comma separated key=value pairs)")
	n.splunkFlagSet = splunkFlags

	cmd.Flags().AddFlagSet(splunkFlags)

	utils.Must(cmd.MarkFlagRequired("splunk-endpoint"))
	utils.Must(cmd.MarkFlagRequired("splunk-token"))
	return cmd
}

type notifierCmd struct {
	notifier *declarativeconfig.Notifier

	env       environment.Environment
	configMap string
	secret    string
	namespace string

	genericFlagSet *pflag.FlagSet
	gc             *declarativeconfig.GenericConfig
	gcHeaders      map[string]string
	gcExtraFields  map[string]string

	splunkFlagSet *pflag.FlagSet
	sc            *declarativeconfig.SplunkConfig
	scSourceTypes map[string]string
}

func (n *notifierCmd) runE() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := n.construct(cmd); err != nil {
			return err
		}
		if err := n.validate(); err != nil {
			return err
		}
		return n.printYAML()
	}
}

func anyFlagChanged(fs *pflag.FlagSet) bool {
	if fs == nil {
		return false
	}
	var changed bool
	fs.VisitAll(func(f *pflag.Flag) { changed = changed || f.Changed })
	return changed
}

func loadCertficate(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", errors.Wrap(err, "reading certificate file")
	}

	for {
		block, rest := pem.Decode(raw)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			return (string)(block.Bytes), nil
		}
		raw = rest
	}

	return "", errox.InvalidArgs.Newf("no certificate found in %q", path)
}

func (n *notifierCmd) construct(cmd *cobra.Command) error {
	configMap, secret, namespace, err := k8sobject.ReadK8sObjectFlags(cmd)
	if err != nil {
		return errors.Wrap(err, "reading config map flag values")
	}
	n.configMap = configMap
	n.secret = secret
	n.namespace = namespace

	if anyFlagChanged(n.genericFlagSet) {
		keys := maputil.Keys(n.gcHeaders)
		sort.Strings(keys)
		for _, k := range keys {
			n.gc.Headers = append(n.gc.Headers, declarativeconfig.KeyValuePair{Key: k, Value: n.gcHeaders[k]})
		}
		keys = maputil.Keys(n.gcExtraFields)
		sort.Strings(keys)
		for _, k := range keys {
			n.gc.ExtraFields = append(n.gc.ExtraFields, declarativeconfig.KeyValuePair{Key: k, Value: n.gcExtraFields[k]})
		}

		if n.gc.CACertPEM != "" {
			if n.gc.CACertPEM, err = loadCertficate(n.gc.CACertPEM); err != nil {
				return errors.Wrap(err, "reading CA certificate file")
			}
		}
		n.notifier.GenericConfig = n.gc
	}

	if anyFlagChanged(n.splunkFlagSet) {
		keys := maputil.Keys(n.scSourceTypes)
		sort.Strings(keys)
		for _, k := range keys {
			n.sc.SourceTypes = append(n.sc.SourceTypes, declarativeconfig.SourceTypePair{Key: k, Value: n.scSourceTypes[k]})
		}
		n.notifier.SplunkConfig = n.sc
	}
	return nil
}

func (n *notifierCmd) validate() error {
	if _, err := url.Parse(n.gc.Endpoint); err != nil {
		return errox.InvalidArgs.New("parsing notifier webhook endpoint URL").CausedBy(err)
	}
	if _, err := url.Parse(n.sc.HTTPEndpoint); err != nil {
		return errox.InvalidArgs.New("parsing notifier Splunk endpoint URL").CausedBy(err)
	}
	return nil
}

func (n *notifierCmd) printYAML() error {
	yamlOutput := &bytes.Buffer{}
	enc := yaml.NewEncoder(yamlOutput)
	if err := enc.Encode(n.notifier); err != nil {
		return errors.Wrap(err, "creating the YAML output")
	}
	if err := lint.Lint(yamlOutput.Bytes()); err != nil {
		return errors.Wrap(err, "linting the YAML output")
	}
	if n.configMap != "" || n.secret != "" {
		return errors.Wrap(k8sobject.WriteToK8sObject(context.Background(), n.configMap, n.secret, n.namespace,
			fmt.Sprintf("%s-%s", n.notifier.ConfigurationType(), n.notifier.Name),
			yamlOutput.Bytes()), "writing the YAML output to config map")
	}
	if _, err := n.env.InputOutput().Out().Write(yamlOutput.Bytes()); err != nil {
		return errors.Wrap(err, "writing the YAML output")
	}
	return nil
}
