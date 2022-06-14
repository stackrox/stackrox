package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stackrox/stackrox/tools/mitre/command"
)

func main() {
	c := &cobra.Command{
		Use:          fmt.Sprintf("%s <command> ...", os.Args[0]),
		Short:        "StackRox MITRE ATT&CK utility tool",
		SilenceUsage: true,
	}

	c.AddCommand(
		command.FetchC(),
	)

	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}
