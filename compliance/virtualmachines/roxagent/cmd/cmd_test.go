package cmd

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduledFlagDefaultsToReactive(t *testing.T) {
	cmd := RootCmd(context.Background())
	err := cmd.ParseFlags([]string{})
	require.NoError(t, err)

	flag := cmd.Flags().Lookup("scheduled")
	require.NotNil(t, flag, "expected --scheduled flag to exist")
	assert.Equal(t, "false", flag.Value.String(), "default should be false (reactive)")
	assert.Nil(t, cmd.Flags().Lookup("trigger"), "old --trigger flag should be removed")
}

func TestTriggerFromScheduled(t *testing.T) {
	trigger := triggerFromScheduled(true)
	assert.Equal(t, v1.ReportTrigger_REPORT_TRIGGER_SCHEDULED, trigger)
}

func TestTriggerFromReactiveDefault(t *testing.T) {
	cases := map[string]bool{
		"default": false,
	}
	for name, scheduled := range cases {
		t.Run(name, func(t *testing.T) {
			trigger := triggerFromScheduled(scheduled)
			assert.Equal(t, v1.ReportTrigger_REPORT_TRIGGER_REACTIVE, trigger)
		})
	}
}
