package tests

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
)

func TestCancelReport(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	conn := centralgrpc.GRPCConnectionToCentral(t)
	reportServiceClient := v2.NewReportServiceClient(conn)
	debugClient := v2.NewDebugActionServiceClient(conn)

	testConfig := getTestReportConfig()
	testConfig, err := reportServiceClient.PostReportConfiguration(ctx, testConfig)
	assert.NoError(t, err)

	breakpoint := "VMReports:SchedulerV2:RunReports"
	debugAction := &v2.DebugAction{
		Identifier: breakpoint,
		NumTimes:   1,
		Action: &v2.DebugAction_WaitAction{
			WaitAction: &v2.WaitAction{},
		},
	}
	_, err = debugClient.RegisterAction(ctx, debugAction)
	assert.NoError(t, err)

	resp, err := reportServiceClient.RunReport(ctx, &v2.RunReportRequest{
		ReportConfigId:           testConfig.Id,
		ReportNotificationMethod: v2.NotificationMethod_DOWNLOAD,
	})
	assert.NoError(t, err)

	err = waitUntilBreakpointReached(ctx, breakpoint, debugClient, 1, 0)
	assert.NoError(t, err)

	_, err = reportServiceClient.CancelReport(ctx, &v2.ResourceByID{Id: resp.ReportId})
	assert.NoError(t, err)

	_, err = debugClient.ProceedAll(ctx, &v2.ResourceByID{Id: breakpoint})
	assert.NoError(t, err)

	err = waitUntilBreakpointReached(ctx, breakpoint, debugClient, 1, 1)
	assert.NoError(t, err)

	_, err = debugClient.DeleteAction(ctx, &v2.ResourceByID{Id: breakpoint})
	assert.NoError(t, err)
}

func getTestReportConfig() *v2.ReportConfiguration {
	return &v2.ReportConfiguration{
		Name: "test-report-config",
		Type: v2.ReportConfiguration_VULNERABILITY,
		Filter: &v2.ReportConfiguration_VulnReportFilters{
			VulnReportFilters: &v2.VulnerabilityReportFilters{
				Fixability: v2.VulnerabilityReportFilters_BOTH,
				Severities: []v2.VulnerabilityReportFilters_VulnerabilitySeverity{
					v2.VulnerabilityReportFilters_CRITICAL_VULNERABILITY_SEVERITY,
					v2.VulnerabilityReportFilters_IMPORTANT_VULNERABILITY_SEVERITY,
					v2.VulnerabilityReportFilters_MODERATE_VULNERABILITY_SEVERITY,
					v2.VulnerabilityReportFilters_LOW_VULNERABILITY_SEVERITY,
				},
				ImageTypes: []v2.VulnerabilityReportFilters_ImageType{
					v2.VulnerabilityReportFilters_DEPLOYED,
					v2.VulnerabilityReportFilters_WATCHED,
				},
				CvesSince: &v2.VulnerabilityReportFilters_AllVuln{
					AllVuln: true,
				},
				IncludeNvdCvss:         false,
				IncludeEpssProbability: false,
			},
		},
	}
}

func waitUntilBreakpointReached(ctx context.Context, identifier string, debugClient v2.DebugActionServiceClient, expectedExecs int64, expectedSignals int64) error {
	retries := 0
	for {
		if retries > 100 {
			return errors.New("Timed out waiting for breakpoint")
		}
		status, err := debugClient.GetActionStatus(ctx, &v2.ResourceByID{Id: identifier})
		if err != nil {
			return err
		}
		if status.TimesExecuted == expectedExecs && status.TimesSignaled == expectedSignals {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
		retries += 1
	}
}
