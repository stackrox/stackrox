package get

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	{{- range $index, $subCmd := .subCmds }}
    {{$subCmd.Name}} "github.com/stackrox/rox/{{$subCmd.Dir}}"
    {{- end }}
)

// Command defines the get command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "get",
		Short: "Display resources",
	}

    {{- range $index, $subCmd := .subCmds }}
    c.AddCommand({{$subCmd.Name}}.Command(cliEnvironment))
    {{- end }}

	flags.AddTimeoutWithDefault(c, time.Minute)
	return c
}
