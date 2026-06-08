package cmd

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerFlagDefaultsToReactive(t *testing.T) {
	cmd := RootCmd(context.Background())
	err := cmd.ParseFlags([]string{})
	require.NoError(t, err)

	flag := cmd.Flags().Lookup("trigger")
	require.NotNil(t, flag, "expected --trigger flag to exist")
	assert.Equal(t, "", flag.Value.String(), "default should be empty string (resolves to reactive)")
}

func TestParseTriggerScheduled(t *testing.T) {
	trigger := parseTrigger("scheduled")
	assert.Equal(t, v1.ReportTrigger_REPORT_TRIGGER_SCHEDULED, trigger)
}

func TestParseTriggerDefaultsToReactive(t *testing.T) {
	cases := map[string]string{
		"empty string":  "",
		"unknown value": "something",
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			trigger := parseTrigger(input)
			assert.Equal(t, v1.ReportTrigger_REPORT_TRIGGER_REACTIVE, trigger)
		})
	}
}
