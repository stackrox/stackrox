package maincommand

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
)

//go:embed commands_tree.yaml
var commandsTree string

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

type cmdNode struct {
	Commands        map[string]*cmdNode `yaml:"cmd,inline,omitempty"`
	LocalFlags      []string            `yaml:"LOCAL_FLAGS,omitempty"`
	PersistentFlags []string            `yaml:"PERSISTENT_FLAGS,omitempty"`
	InheritedFlags  []string            `yaml:"INHERITED_FLAGS,omitempty"`
}

func buildCmdTree(c *cobra.Command) *cmdNode {
	command := &cmdNode{
		Commands: make(map[string]*cmdNode),
	}
	c.LocalNonPersistentFlags().VisitAll(func(f *pflag.Flag) {
		command.LocalFlags = append(command.LocalFlags, f.Name)
	})
	c.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		command.PersistentFlags = append(command.PersistentFlags, f.Name)
	})
	c.InheritedFlags().VisitAll(func(f *pflag.Flag) {
		command.InheritedFlags = append(command.InheritedFlags, f.Name)
	})
	c.InheritedFlags().VisitAll(func(f *pflag.Flag) {
		command.InheritedFlags = append(command.InheritedFlags, f.Name)
	})

	for _, cmd := range c.Commands() {
		command.Commands[cmd.Name()] = buildCmdTree(cmd)
	}
	return command
}

func Test_commandTree(t *testing.T) {
	sb := &strings.Builder{}
	e := yaml.NewEncoder(sb)
	e.SetIndent(2)
	require.NoError(t, e.Encode(buildCmdTree(Command())))
	_ = e.Close()
	result := sb.String()
	// Regenerate the tree:
	// os.WriteFile(commandTreeFilename, []byte(result), 0666)
	assert.NotEmpty(t, commandTreeFilename, result)
	assert.Equal(t, commandTree, result)
}
