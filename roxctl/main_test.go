package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestSetDescription(t *testing.T) {
	cases := []struct {
		name  string
		cmd   *cobra.Command
		long  string
		short string
	}{
		{
			name: "short and long should not be overridden by property when non-empty",
			cmd: &cobra.Command{
				Use:   "central",
				Short: "testing",
				Long:  "testing",
			},
			long:  "testing",
			short: "testing",
		},
		{
			name: "short should not be overridden by property when non-empty, long should be set by property",
			cmd: &cobra.Command{
				Use:   "version",
				Short: "testing",
			},
			short: "testing",
			long:  "Display the current roxctl version.",
		},
		{
			name: "long should not be overridden by property when non-empty, short should be set by property",
			cmd: &cobra.Command{
				Use:  "version",
				Long: "testing",
			},
			short: "Display the current roxctl version.",
			long:  "testing",
		},
		{
			name: "short and long should be set by property",
			cmd: &cobra.Command{
				Use: "version",
			},
			short: "Display the current roxctl version.",
			long:  "Display the current roxctl version.",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rootCmd := &cobra.Command{Use: "root"}
			rootCmd.AddCommand(c.cmd)
			setDescription(c.cmd)
			assert.Equal(t, c.short, c.cmd.Short, "Short description: %q != %q", c.short, c.cmd.Short)
			assert.Equal(t, c.long, c.cmd.Long, "Long description: %q != %q", c.long, c.cmd.Long)
		})
	}
}
