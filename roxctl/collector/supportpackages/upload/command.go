package upload

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	description = "Upload the files from a collector support package to Central"
)

// Command defines the command. See usage strings for details.
func Command() *cobra.Command {
	overwrite := false

	c := &cobra.Command{
		Use:   "upload <package-file>",
		Short: description,
		Long:  description,
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.Errorf("too many positional arguments (expected 1, got %d)", len(args))
			}
			if len(args) == 0 {
				return errors.New("missing <package-file> argument")
			}

			packageFile := args[0]
			return uploadFilesFromPackage(packageFile, overwrite)
		},
	}

	c.Flags().BoolVarP(&overwrite, "overwrite", "", false, "whether to overwrite present but different files")
	return c
}
