package lint

import (
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/declarativeconfig/transform"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
)

// Command provides the lint command for declartive configuration.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	lintCmd := &lintCmd{env: cliEnvironment}

	cmd := &cobra.Command{
		Use:  "lint",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := lintCmd.Validate(); err != nil {
				return err
			}
			return lintCmd.Lint()
		},
	}

	cmd.Flags().StringVarP(&lintCmd.file, "file", "f", "", "file containing the declarative configuration in YAML format")

	utils.Must(cmd.MarkFlagRequired("file"))
	return cmd
}

type lintCmd struct {
	env environment.Environment

	file         string
	fileContents []byte
}

func (l *lintCmd) Validate() error {
	contents, err := os.ReadFile(l.file)
	if err != nil {
		if os.IsNotExist(err) {
			return errox.NotFound.Newf("file %s could not be found", l.file).CausedBy(err)
		}
		return errox.InvalidArgs.CausedBy(err)
	}
	l.fileContents = contents
	return nil
}

func (l *lintCmd) Lint() error {
	configurations, err := declarativeconfig.ConfigurationFromRawBytes(l.fileContents)
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

	l.env.Logger().InfofLn("Successfully validated declarative configuration within file %s", l.file)
	return nil
}
