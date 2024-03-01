package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/roxctl/maincommand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CommandUsage(t *testing.T) {
	t.Setenv(env.DeclarativeConfiguration.EnvVar(), "true")

	c := maincommand.Command()
	AddMissingDefaultsToFlagUsage(c)
	const plainTextUsage = "Use a plaintext (unencrypted) connection; only works in conjunction with --insecure. Alternatively can be enabled by setting the ROX_PLAINTEXT environment variable to true (default false)"

	assert.Equal(t, plainTextUsage, c.Flag("plaintext").Usage)

	var cmd *cobra.Command
	for _, cmd = range c.Commands() {
		if cmd.Name() == "central" {
			break
		}
	}
	require.NotNil(t, cmd)
	assert.Equal(t, plainTextUsage, cmd.Flag("plaintext").Usage)

leg:
	for _, cmd = range c.Commands() {
		if cmd.Name() == "declarative-config" {
			/*c = cmd
			for _, cmd = range c.Commands() {
				if cmd.Name() == "create" {
					break leg
				}
			}*/
			break leg
		}
	}
	require.NotNil(t, cmd)
	_ = cmd.UsageString()
	assert.True(t, cmd.InheritedFlags().Lookup("endpoint").Hidden)
	//assert.True(t, cmd.Flag("plaintext").Hidden)
}
