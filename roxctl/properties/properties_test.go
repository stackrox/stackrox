package properties

import (
	"fmt"
	"testing"

	"github.com/magiconair/properties"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/roxctl/help"
	"github.com/stackrox/rox/roxctl/maincommand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type helpKeysTestSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func TestHelpKeys(t *testing.T) {
	suite.Run(t, new(helpKeysTestSuite))
}

func (s *helpKeysTestSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.RoxctlNetpolGenerate.EnvVar(), "true")
	testbuildinfo.SetForTest(s.T())
}

func (s *helpKeysTestSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

// TestHelpKeysExist tests that the short and long help key values exist for each command in the properties file
func (s *helpKeysTestSuite) TestHelpKeysExist() {
	c := maincommand.Command()

	props, err := help.ReadProperties()
	require.NoError(s.T(), err)

	checkHelp(s.T(), c.Commands(), props)
}

// TestNoDanglingHelpKeys tests that there are no unused key value pairs in the help properties file
func (s *helpKeysTestSuite) TestNoDanglingHelpKeys() {
	c := maincommand.Command()

	props, err := help.ReadProperties()
	require.NoError(s.T(), err)

	findDanglingHelpKeys(s.T(), c.Commands(), props)

	// do not report dangling help-keys for disabled feature
	if !features.RoxctlNetpolGenerate.Enabled() {
		props.Delete("generate.short")
		props.Delete("generate.long")
		props.Delete("generate.netpol.short")
		props.Delete("generate.netpol.long")
	}

	if len(props.Keys()) != 0 {
		fmt.Println("Unused help keys: ")
		for _, k := range props.Keys() {
			fmt.Println(k)
		}
		assert.Empty(s.T(), props.Keys(), "found dangling property keys")
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
