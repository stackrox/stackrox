package maincommand

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func isCapitalized(s string) bool {
	if len(s) == 0 {
		return true
	}
	first := string([]byte{s[0]})
	return first == strings.ToUpper(first)
}

func getCommandPath(command *cobra.Command) string {
	// Assume the binary has no spaces in the filepath.
	if path := strings.SplitN(command.CommandPath(), " ", 2); len(path) > 1 {
		return path[1]
	}
	return "roxctl"
}

func checkUsageFirstCharacter(t *testing.T, command *cobra.Command) {
	command.LocalFlags().VisitAll(func(flag *pflag.Flag) {
		if !isCapitalized(flag.Usage) {
			t.Errorf(`"%s --%s" flag usage: %q`, getCommandPath(command), flag.Name, flag.Usage)
		}
	})
	if !isCapitalized(command.Short) {
		t.Errorf("%q, short usage: %q", getCommandPath(command), command.Short)
	}
	if !isCapitalized(command.Long) {
		t.Errorf("%q, long usage: %q", getCommandPath(command), command.Long)
	}
	for _, subcommand := range command.Commands() {
		t.Run(getCommandPath(subcommand), func(t *testing.T) {
			checkUsageFirstCharacter(t, subcommand)
		})
	}
}

func Test_Commands(t *testing.T) {
	checkUsageFirstCharacter(t, Command())
}
