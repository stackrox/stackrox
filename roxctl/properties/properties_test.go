package properties

import (
	"fmt"
	"log"
	"testing"

	"github.com/magiconair/properties"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/maincommand"
	"github.com/stackrox/rox/roxctl/packer"
	"github.com/stretchr/testify/assert"
)

// TestHelpKeysExist tests that the short and long help key values exist for each command in the properties file
func TestHelpKeysExist(t *testing.T) {
	c := maincommand.Command()
	props := properties.NewProperties()

	buf, err := packer.RoxctlBox.Find(packer.PropertiesFile)
	if err != nil {
		log.Panicf("error reading help properties file %s: %v", packer.PropertiesFile, err)
	}
	err = props.Load(buf, properties.UTF8)
	if err != nil {
		log.Panicf("error loading help properties file %s: %v", packer.PropertiesFile, err)
	}
	checkHelp(t, c.Commands(), props)
}

// TestNoDanglingHelpKeys tests that there are no unused key value pairs in the help properties file
func TestNoDanglingHelpKeys(t *testing.T) {
	c := maincommand.Command()
	props := properties.NewProperties()

	buf, err := packer.RoxctlBox.Find(packer.PropertiesFile)
	if err != nil {
		log.Panicf("error reading help properties file %s: %v", packer.PropertiesFile, err)
	}
	err = props.Load(buf, properties.UTF8)
	if err != nil {
		log.Panicf("error loading help properties file %s: %v", packer.PropertiesFile, err)
	}

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

		assert.True(t, shortKeyOk, "unable to get short command help key for %s", c.Name())
		assert.True(t, longKeyOk, "unable to get long command help key for %s", c.Name())

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
