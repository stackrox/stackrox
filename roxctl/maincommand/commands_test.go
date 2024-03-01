package maincommand

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func checkUsageFirstCharacter(command *cobra.Command, t *testing.T) {
	command.Flags().VisitAll(func(flag *pflag.Flag) {
		s := string([]byte{flag.Usage[0]})
		assert.Equal(t, s, strings.ToUpper(s),
			"Command %q, flag %q, usage doesn't start with capital letter: %q",
			command.Name(), flag.Name, flag.Usage)
	})
	for _, subcommand := range command.Commands() {
		checkUsageFirstCharacter(subcommand, t)
	}
}

func Test_Commands(t *testing.T) {
	checkUsageFirstCharacter(Command(), t)
}
