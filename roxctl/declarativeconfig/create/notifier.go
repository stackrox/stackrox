package create

import (
	"bytes"
	"context"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/url"
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
		Use:   notifierCommand.notifier.Type(),
		Args:  cobra.NoArgs,
		RunE:  notifierCommand.runE(),
		Short: "Create a declarative configuration for a notifier",
	}

	cmd.Flags().StringVar(&notifierCommand.notifier.Name, "name", "", "Name of the notifier")

	genericFlags := pflag.NewFlagSet("generic", pflag.PanicOnError)
	notifierCommand.gc = &declarativeconfig.GenericConfig{}
	genericFlags.BoolVar(&notifierCommand.gc.AuditLoggingEnabled, "audit-logging", false, "Notifier audit logging enabled")
	genericFlags.StringVar(&notifierCommand.gc.Endpoint, "endpoint", "", "Endpoint URL")
	genericFlags.StringVar(&notifierCommand.gc.Username, "username", "", "Username for the endpoint basic authentication")
	genericFlags.StringVar(&notifierCommand.gc.Password, "password", "", "Password for the endpoint basic authentication")
	genericFlags.StringVar(&notifierCommand.gc.CACertPEM, "cacert", "", "Endpoint CA certificate file name (PEM format)")
	genericFlags.StringToStringVar(&notifierCommand.gcHeaders, "headers", nil, "Headers (comma separated key=value pairs)")
	genericFlags.StringToStringVar(&notifierCommand.gcExtraFields, "extra-fields", nil, "Extra fields (comma separated key=value pairs)")
	genericFlags.BoolVar(&notifierCommand.gc.SkipTLSVerify, "skip-tls-verify", false, "Skip TLS verification")
	notifierCommand.genericFlagSet = genericFlags
	cmd.Flags().AddFlagSet(genericFlags)

	splunkFlags := pflag.NewFlagSet("splunk", pflag.PanicOnError)
	notifierCommand.sc = &declarativeconfig.SplunkConfig{}
	splunkFlags.BoolVar(&notifierCommand.sc.AuditLoggingEnabled, "splunk-audit-logging", false, "Splunk audit logging enabled")
	splunkFlags.StringVar(&notifierCommand.sc.HTTPToken, "splunk-token", "", "Splunk HTTP token")
	splunkFlags.StringVar(&notifierCommand.sc.HTTPEndpoint, "splunk-endpoint", "", "Splunk HTTP endpoint")
	splunkFlags.BoolVar(&notifierCommand.sc.Insecure, "splunk-insecure", false, "Insecure connection to Splunk")
	splunkFlags.Int64Var(&notifierCommand.sc.Truncate, "splunk-truncate", 0, "Splunk truncate limit")
	splunkFlags.StringToStringVar(&notifierCommand.scSourceTypes, "splunk-source-types", nil, "Splunk source types")
	notifierCommand.splunkFlagSet = splunkFlags

	cmd.Flags().AddFlagSet(splunkFlags)

	utils.Must(cmd.MarkFlagFilename("cacert"))
	utils.Must(cmd.MarkFlagRequired("name"))
	utils.Must(cmd.MarkFlagRequired("endpoint"))
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
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
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

	return "", errox.InvalidArgs.Newf("no certificate found in \"%s\"", path)
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
			n.gc.Headers = append(n.gc.Headers, declarativeconfig.KeyValuePair{k, n.gcHeaders[k]})
		}
		keys = maputil.Keys(n.gcExtraFields)
		sort.Strings(keys)
		for _, k := range keys {
			n.gc.ExtraFields = append(n.gc.ExtraFields, declarativeconfig.KeyValuePair{k, n.gcExtraFields[k]})
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
			n.sc.SourceTypes = append(n.sc.SourceTypes, declarativeconfig.SourceTypePair{k, n.scSourceTypes[k]})
		}
		n.notifier.SplunkConfig = n.sc
	}
	return nil
}

func (n *notifierCmd) validate() error {
	if _, err := url.Parse(n.gc.Endpoint); err != nil {
		return errox.InvalidArgs.New("parsing notifier endpoint URL").CausedBy(err)
	}
	if n.sc.HTTPEndpoint == "" && n.sc.HTTPToken != "" {
		return errox.InvalidArgs.New("missing splunk-endpoint")
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
			fmt.Sprintf("%s-%s", n.notifier.Type(), n.notifier.Name),
			yamlOutput.Bytes()), "writing the YAML output to config map")
	}
	if _, err := n.env.InputOutput().Out().Write(yamlOutput.Bytes()); err != nil {
		return errors.Wrap(err, "writing the YAML output")
	}
	return nil
}
