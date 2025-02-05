package maincommand

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func isCapitalized(s string) bool {
	if len(s) == 0 {
		return true
	}
	first := string([]byte{s[0]})
	return first == strings.ToUpper(first)
}

func hasNoTrailingPeriod(s string) bool {
	return !strings.HasSuffix(s, ".") && !strings.HasSuffix(s, "!")
}

var shortChecks = map[string]func(string) bool{
	"must be capitalized":           isCapitalized,
	"must not have trailing period": hasNoTrailingPeriod,
}

var longChecks = map[string]func(string) bool{
	"must be capitalized":       isCapitalized,
	"must have trailing period": func(s string) bool { return s == "" || !hasNoTrailingPeriod(s) },
}

var isCapitalizedCheck = map[string]func(string) bool{
	"must be capitalized": isCapitalized,
}

func runChecks(message string, checks map[string]func(string) bool) error {
	errors := []string{}
	for test, check := range checks {
		if !check(message) {
			errors = append(errors, test)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}

func getCommandPath(command *cobra.Command) string {
	// Assume the binary has no spaces in the filepath.
	if path := strings.SplitN(command.CommandPath(), " ", 2); len(path) > 1 {
		return path[1]
	}
	return "roxctl"
}

func checkUsageStyle(t *testing.T, command *cobra.Command) {
	command.LocalFlags().VisitAll(func(flag *pflag.Flag) {
		assert.NoErrorf(t, runChecks(flag.Usage, longChecks),
			`"%s --%s" flag usage: %q`, getCommandPath(command), flag.Name, flag.Usage)

	})
	assert.NoErrorf(t, runChecks(command.Short, shortChecks),
		"%q, short usage: %q", getCommandPath(command), command.Short)

	if command.Use == "doc [man|md|yaml|rest]" {
		// This command long description ends with a list, hence exception.
		assert.NoErrorf(t, runChecks(command.Long, isCapitalizedCheck),
			"%q, long usage: %q", getCommandPath(command), command.Long)
	} else {
		assert.NoErrorf(t, runChecks(command.Long, longChecks),
			"%q, long usage: %q", getCommandPath(command), command.Long)
	}

	for _, subcommand := range command.Commands() {
		t.Run(getCommandPath(subcommand), func(t *testing.T) {
			checkUsageStyle(t, subcommand)
		})
	}
}

func Test_Commands(t *testing.T) {
	checkUsageStyle(t, Command())
}
