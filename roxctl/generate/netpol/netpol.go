package netpol

import (
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
	mergePolicies    bool
	splitPolicies    bool

	// injected or constructed values
	env     environment.Environment
	printer printer.ObjectPrinter
}

// Command defines the netpol command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	generateNetpolCmd := &generateNetpolCommand{env: cliEnvironment}
	c := &cobra.Command{
		Use: "netpol <folder-path>",
		RunE: func(c *cobra.Command, args []string) error {
			if err := validate(args); err != nil {
				return err
			}
			if err := generateNetpolCmd.construct(args); err != nil {
				return err
			}
			return generateNetpolCmd.generateNetpol()
		},
	}

	c.Flags().StringVarP(&generateNetpolCmd.outputFolderPath, "output-dir", "d", "./policies", "path to the output directory for generated policies")
	c.Flags().StringVarP(&generateNetpolCmd.outputFilePath, "output-file", "f", "./policies.yaml", "path to the output file for merged policies")
	c.Flags().BoolVar(&generateNetpolCmd.mergePolicies, "merge-policies", false, "Merge all generated Network Policies into a single file. Combine with -f for target file")
	c.Flags().BoolVar(&generateNetpolCmd.splitPolicies, "split-policies", false, "Create one file per Network Policy. Combine with -d for target folder")

	c.MarkFlagsRequiredTogether("split-policies", "output-dir")
	c.MarkFlagsRequiredTogether("merge-policies", "output-file")
	return c
}

func validate(args []string) error {
	if len(args) > 1 {
		return errox.InvalidArgs.Newf("too many positional arguments (expected 1, got %d)", len(args))
	}
	if len(args) == 0 {
		return errox.InvalidArgs.New("missing <folder-path> argument")
	}
	return nil
}

func (cmd *generateNetpolCommand) construct(args []string) error {
	cmd.folderPath = args[0]
	return nil
}
