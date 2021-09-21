package properties

import (
	"fmt"
	"testing"

	"github.com/magiconair/properties"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/help"
	"github.com/stackrox/rox/roxctl/maincommand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHelpKeysExist tests that the short and long help key values exist for each command in the properties file
func TestHelpKeysExist(t *testing.T) {
	c := maincommand.Command()

	props, err := help.ReadProperties()
	require.NoError(t, err)

	checkHelp(t, c.Commands(), props)
}

// TestNoDanglingHelpKeys tests that there are no unused key value pairs in the help properties file
func TestNoDanglingHelpKeys(t *testing.T) {
	c := maincommand.Command()

	props, err := help.ReadProperties()
	require.NoError(t, err)

	findDanglingHelpKeys(t, c.Commands(), props)

	if len(props.Keys()) != 0 {
		fmt.Println("Unused help keys: ")
		for _, k := range props.Keys() {
			fmt.Println(k)
		}
		assert.Empty(t, props.Keys(), "found dangling property keys")
	}
}

func checkHelp(t *testing.T, commands []*cobra.Command, props *properties.Properties) {
	for _, c := range commands {
		_, shortKeyOk := props.Get(GetShortCommandKey(c.CommandPath()))
		_, longKeyOk := props.Get(GetLongCommandKey(c.CommandPath()))

		if c.Short == "" {
			assert.True(t, shortKeyOk, "unable to get short command help key for %s", c.Name())
		}
		if c.Long == "" {
			assert.True(t, longKeyOk, "unable to get long command help key for %s", c.Name())
		}
		checkHelp(t, c.Commands(), props)
	}
}

func findDanglingHelpKeys(t *testing.T, commands []*cobra.Command, props *properties.Properties) {
	for _, c := range commands {

		shortCommandKey := GetShortCommandKey(c.CommandPath())
		longCommandKey := GetLongCommandKey(c.CommandPath())

		props.Delete(shortCommandKey)
		props.Delete(longCommandKey)

		findDanglingHelpKeys(t, c.Commands(), props)
	}
}
