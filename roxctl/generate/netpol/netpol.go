package netpol

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/printer"
)

type generateNetpolCommand struct {
	// Properties that are bound to cobra flags.
	offline          bool
	folderPath       string
	outputFolderPath string
	outputFilePath   string
	removeOutputPath bool
	mergeMode        bool
	splitMode        bool
	stdoutMode       bool

	// injected or constructed values
	env     environment.Environment
	printer printer.ObjectPrinter
}

// Command defines the netpol command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	generateNetpolCmd := &generateNetpolCommand{env: cliEnvironment}
	c := &cobra.Command{
		Use:  "netpol <folder-path>",
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if err := generateNetpolCmd.construct(args, c); err != nil {
				return err
			}
			if err := generateNetpolCmd.validate(); err != nil {
				return err
			}
			return generateNetpolCmd.generateNetpol()
		},
	}
	c.Flags().BoolVar(&generateNetpolCmd.removeOutputPath, "remove", false, "remove the output path if it already exists")
	c.Flags().StringVarP(&generateNetpolCmd.outputFolderPath, "output-dir", "d", "", "save generated policies into target folder - one file per policy")
	c.Flags().StringVarP(&generateNetpolCmd.outputFilePath, "output-file", "f", "", "save and merge generated policies into a single yaml file")
	return c
}

func (cmd *generateNetpolCommand) construct(args []string, c *cobra.Command) error {
	cmd.folderPath = args[0]
	cmd.splitMode = c.Flags().Changed("output-dir")
	cmd.mergeMode = c.Flags().Changed("output-file")
	if !cmd.splitMode && !cmd.mergeMode {
		cmd.stdoutMode = true
	}
	return nil
}

func (cmd *generateNetpolCommand) validate() error {
	if cmd.outputFolderPath != "" && cmd.outputFilePath != "" {
		return errors.New("Flags [-d|--output-dir, -f|--output-file] cannot be used together")
	}
	if cmd.splitMode {
		if err := cmd.setupPath(cmd.outputFolderPath); err != nil {
			return errors.Wrap(err, "failed to set up folder path")
		}
	} else if cmd.mergeMode {
		if err := cmd.setupPath(cmd.outputFilePath); err != nil {
			return errors.Wrap(err, "failed to set up file path")
		}
	}

	return nil
}

func (cmd *generateNetpolCommand) setupPath(path string) error {
	if _, err := os.Stat(path); err == nil {
		if !cmd.removeOutputPath {
			return errox.AlreadyExists.Newf("path %s already exists. Use --remove to overwrite or select a different path.", path)
		}
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to check if path %s exists", path)
	}
	cmd.env.Logger().InfofLn("Writing generated Network Policies to %s", path)
	return nil
}
