package upload

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

type collectorSPUploadCommand struct {
	// Properties that are bound to cobra flags.
	overwrite    bool
	packageFile  string
	timeout      time.Duration
	retryTimeout time.Duration

	// Properties that are injected or constructed.
	env environment.Environment
}

// Command defines the command. See usage strings for details.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	collectorSPUploadCmd := &collectorSPUploadCommand{env: cliEnvironment}
	c := &cobra.Command{
		Use:   "upload <package-file>",
		Short: "Upload files from a collector support package to Central.",
		RunE: func(c *cobra.Command, args []string) error {
			if err := validate(args); err != nil {
				return err
			}
			if err := collectorSPUploadCmd.construct(c, args); err != nil {
				return err
			}
			return collectorSPUploadCmd.uploadFilesFromPackage()
		},
	}

	c.Flags().BoolVarP(&collectorSPUploadCmd.overwrite, "overwrite", "", false, "whether to overwrite present but different files")
	flags.AddTimeout(c)
	flags.AddRetryTimeout(c)
	return c
}

func validate(args []string) error {
	if len(args) > 1 {
		return errox.InvalidArgs.Newf("too many positional arguments (expected 1, got %d)", len(args))
	}
	if len(args) == 0 {
		return errox.InvalidArgs.New("missing <package-file> argument")
	}
	return nil
}

func (cmd *collectorSPUploadCommand) construct(c *cobra.Command, args []string) error {
	cmd.packageFile = args[0]
	cmd.timeout = flags.Timeout(c)
	cmd.retryTimeout = flags.RetryTimeout(c)
	return nil
}
