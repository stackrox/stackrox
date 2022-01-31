package upload

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
)

type collectorSPUploadCommand struct {
	// Properties that are bound to cobra flags.
	overwrite   bool
	packageFile string

	// Properties that are injected or constructed.
	env environment.Environment
}

// Command defines the command. See usage strings for details.
func Command(cliEnvironment environment.Environment) *cobra.Command {

	collectorSPUploadCmd := &collectorSPUploadCommand{env: cliEnvironment}
	c := &cobra.Command{
		Use: "upload <package-file>",
		RunE: func(c *cobra.Command, args []string) error {
			if err := validate(args); err != nil {
				return err
			}
			if err := collectorSPUploadCmd.construct(args); err != nil {
				return err
			}
			return collectorSPUploadCmd.uploadFilesFromPackage()
		},
	}

	c.Flags().BoolVarP(&collectorSPUploadCmd.overwrite, "overwrite", "", false, "whether to overwrite present but different files")
	return c
}

func validate(args []string) error {
	if len(args) > 1 {
		return errors.Errorf("too many positional arguments (expected 1, got %d)", len(args))
	}
	if len(args) == 0 {
		return errors.New("missing <package-file> argument")
	}
	return nil
}

func (cmd *collectorSPUploadCommand) construct(args []string) error {
	cmd.packageFile = args[0]
	return nil
}
