package views

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRiskViewSelectProtosMatchDests(t *testing.T) {
	var sc ProcessIndicatorRiskScanner
	dests := sc.Dests()
	protos := RiskViewSelectProtos()

	require.Equal(t, len(protos), len(dests),
		"RiskViewSelectProtos and Dests() must have the same length")

	expectedOrder := []search.FieldLabel{
		search.ProcessID,
		search.ContainerName,
		search.ProcessExecPath,
		search.ProcessContainerStartTime,
		search.ProcessCreationTime,
		search.ProcessName,
		search.ProcessArguments,
	}

	require.Equal(t, len(expectedOrder), len(protos),
		"expectedOrder must match RiskViewSelectProtos length")

	for i, sel := range protos {
		assert.Equal(t, expectedOrder[i].String(), sel.GetField().GetName(),
			"RiskViewSelectProtos[%d] field name mismatch", i)
	}
}

func TestRiskViewSelectProtosReturnsFreshCopies(t *testing.T) {
	a := RiskViewSelectProtos()
	b := RiskViewSelectProtos()

	require.Equal(t, len(a), len(b))
	for i := range a {
		assert.NotSame(t, a[i], b[i],
			"RiskViewSelectProtos()[%d] must not share pointers across calls", i)
	}
}

func TestProcessIndicatorRiskScannerBuild(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	cases := map[string]struct {
		scanner  ProcessIndicatorRiskScanner
		expected ProcessIndicatorRiskView
	}{
		"all fields populated": {
			scanner: ProcessIndicatorRiskScanner{
				ID:                 "id-1",
				ContainerName:      "nginx",
				ExecFilePath:       "/bin/sh",
				ContainerStartTime: pgtype.Timestamp{Time: now.Add(-time.Hour), Valid: true},
				SignalTime:         pgtype.Timestamp{Time: now, Valid: true},
				SignalName:         "sh",
				SignalArgs:         "-c echo",
			},
			expected: ProcessIndicatorRiskView{
				ID:                 "id-1",
				ContainerName:      "nginx",
				ExecFilePath:       "/bin/sh",
				ContainerStartTime: func() *time.Time { t := now.Add(-time.Hour); return &t }(),
				SignalTime:         func() *time.Time { t := now; return &t }(),
				SignalName:         "sh",
				SignalArgs:         "-c echo",
			},
		},
		"nil timestamps": {
			scanner: ProcessIndicatorRiskScanner{
				ID:           "id-2",
				ExecFilePath: "/usr/bin/curl",
				SignalName:   "curl",
			},
			expected: ProcessIndicatorRiskView{
				ID:           "id-2",
				ExecFilePath: "/usr/bin/curl",
				SignalName:   "curl",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result := tc.scanner.Build()
			assert.Equal(t, tc.expected, result)
		})
	}
}
