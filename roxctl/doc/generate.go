package doc

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Command provides the doc generation command.
func Command(_ environment.Environment) *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "doc [man|md|yaml|rest]",
		Short: "Generate docs for roxctl",
		Long: `Generate docs for roxctl in the given format.

The following formats are supported:
- man generates man page like docs
- md generates markdown pages
- rest generates reStructured text docs
`,
		Args: common.ExactArgsWithCustomErrMessage(1,
			"Missing argument. Use one of the following: [man|md|yaml|rest]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Current caveat: in case the command is not within PATH, cmd.CommandPath will be appended to the
			// given directory. This may result in errors during writing the docs to the specified output.
			switch args[0] {
			case "man":
				return errors.Wrap(doc.GenManTree(cmd.Root(), &doc.GenManHeader{
					Title:  "roxctl",
					Source: "roxctl",
				}, dir), "generating man page docs")
			case "md":
				return errors.Wrap(doc.GenMarkdownTree(cmd.Root(), dir), "generating markdown docs")
			case "yaml":
				return errors.Wrap(doc.GenYamlTree(cmd.Root(), dir), "generating YAML docs")
			case "rest":
				return errors.Wrap(doc.GenReSTTree(cmd.Root(), dir), "generating reStructured docs")
			default:
				return common.ErrInvalidCommandOption.CausedByf("invalid option %q used; use one of the following: [man|md|yaml|rest]", args[0])
			}
		},
	}

	cmd.Flags().StringVarP(&dir, "output", "o", "",
		"directory where the docs should be written to")
	utils.Must(cmd.MarkFlagRequired("output"))
	flags.HideInheritedFlags(cmd)

	return cmd
}
