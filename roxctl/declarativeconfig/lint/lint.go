package lint

import (
	"context"
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/declarativeconfig/transform"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/declarativeconfig/configmap"
)

// Command provides the lint command for declartive configuration.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	lintCmd := &lintCmd{env: cliEnvironment}

	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Lint an existing declarative configuration YAML file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := lintCmd.Construct(cmd); err != nil {
				return err
			}
			return lintCmd.Lint()
		},
	}

	cmd.Flags().StringVarP(&lintCmd.file, "file", "f", "", "file containing the declarative configuration in YAML format")

	cmd.Flags().String(configmap.ConfigMapFlag, "", `config map from which to read the declarative configuration from.
In case this is not set, the declarative configuration will be read from the YAML file provided via the --file flag.`)
	cmd.Flags().String(configmap.NamespaceFlag, "", `namespace of the config map from which to read the declarative configuration from.
In case this is not set, the namespace set within the current kube config context will be used`)

	cmd.MarkFlagsMutuallyExclusive("file", configmap.ConfigMapFlag)

	return cmd
}

type lintCmd struct {
	env environment.Environment

	file         string
	fileContents [][]byte

	configMap string
	namespace string
}

func (l *lintCmd) Construct(cmd *cobra.Command) error {
	configMap, namespace, err := configmap.ReadConfigMapFlags(cmd)
	if err != nil {
		return errors.Wrap(err, "reading config map flag values")
	}
	l.configMap = configMap
	l.namespace = namespace

	if l.configMap != "" {
		contents, err := configmap.ReadFromConfigMap(context.Background(), l.configMap, l.namespace)
		if err != nil {
			return errors.Wrapf(err, "reading from config map %s", l.configMap)
		}
		l.fileContents = contents
	}

	contents, err := os.ReadFile(l.file)
	if err != nil {
		if os.IsNotExist(err) {
			return errox.NotFound.Newf("file %s could not be found", l.file).CausedBy(err)
		}
		return errox.InvalidArgs.CausedBy(err)
	}
	l.fileContents = [][]byte{contents}
	return nil
}

func (l *lintCmd) Lint() error {
	if err := l.lint(); err != nil {
		return err
	}
	l.env.Logger().InfofLn("Successfully validated declarative configuration within file %s", l.file)
	return nil
}

func (l *lintCmd) lint() error {
	configurations, err := declarativeconfig.ConfigurationFromRawBytes(l.fileContents...)
	if err != nil {
		return errors.Wrap(err, "unmarshalling raw configuration")
	}
	t := transform.New()
	var transformErrs *multierror.Error
	for _, config := range configurations {
		if _, err := t.Transform(config); err != nil {
			transformErrs = multierror.Append(transformErrs, err)
		}
	}

	err = transformErrs.ErrorOrNil()
	if err != nil {
		return errors.Wrap(err, "validating configuration")
	}
	return nil
}

// Lint provides a helper utility to lint a YAML input containing declarative configuration.
func Lint(yaml []byte) error {
	l := lintCmd{fileContents: [][]byte{yaml}}
	return l.lint()
}
