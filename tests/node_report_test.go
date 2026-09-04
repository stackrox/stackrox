//go:build test_e2e

package tests

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestNodeReport(t *testing.T) {
	suite.Run(t, new(NodeReportSuite))
}

type NodeReportSuite struct {
	suite.Suite

	ctx       context.Context
	cancel    context.CancelFunc
	service   apiV2.NodeReportServiceClient
	configIDs []string
}

func (s *NodeReportSuite) SetupSuite() {
	if !features.NodeVulnerabilityReports.Enabled() {
		s.T().Skipf("Skipping because %s is not enabled", features.NodeVulnerabilityReports.EnvVar())
	}
	s.ctx, s.cancel = context.WithTimeout(context.Background(), 30*time.Minute)
	conn := centralgrpc.GRPCConnectionToCentral(s.T())
	s.service = apiV2.NewNodeReportServiceClient(conn)
	s.waitForCentralReady()
}

func (s *NodeReportSuite) waitForCentralReady() {
	s.T().Log("Waiting for Central to be ready...")
	s.Require().Eventually(func() bool {
		ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
		defer cancel()
		_, err := s.service.CountNodeReportConfigurations(ctx, &apiV2.RawQuery{})
		if status.Code(err) == codes.Unimplemented {
			s.T().Skip("Node report service is not registered (feature flag disabled on Central)")
		}
		return err == nil
	}, 2*time.Minute, 2*time.Second, "Central did not become ready")
}

func (s *NodeReportSuite) TearDownSuite() {
	defer s.cancel()
	for _, id := range s.configIDs {
		ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
		_, err := s.service.DeleteNodeReportConfiguration(ctx, &apiV2.ResourceByID{Id: id})
		cancel()
		if err != nil {
			s.T().Logf("Failed to delete report config %s: %v", id, err)
		}
	}
}

func (s *NodeReportSuite) newNodeReportConfig(name, query string) *apiV2.ReportConfiguration {
	return &apiV2.ReportConfiguration{
		Name:        name,
		Description: "E2E test node vulnerability report",
		Type:        apiV2.ReportConfiguration_NODE_VULNERABILITY,
		Filter: &apiV2.ReportConfiguration_NodeVulnReportFilters{
			NodeVulnReportFilters: &apiV2.NodeVulnerabilityReportFilters{
				CvesSince: &apiV2.NodeVulnerabilityReportFilters_AllVuln{AllVuln: true},
				Query:     query,
			},
		},
		ResourceScope: &apiV2.ResourceScope{
			ScopeReference: &apiV2.ResourceScope_EntityScope{
				EntityScope: &apiV2.EntityScope{
					Rules: []*apiV2.EntityScopeRule{
						{
							Entity: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
							Field:  apiV2.ScopeField_FIELD_NAME,
							Values: []*apiV2.RuleValue{
								{Value: "remote", MatchType: apiV2.MatchType_EXACT},
							},
						},
					},
				},
			},
		},
	}
}

func (s *NodeReportSuite) TestCreateNodeReportConfig() {
	config := s.newNodeReportConfig("e2e-node-report-create", "CVSS:>=7")

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	created, err := s.service.PostNodeReportConfiguration(ctx, config)
	s.Require().NoError(err, "creating node report config")
	s.Require().NotEmpty(created.GetId())
	s.configIDs = append(s.configIDs, created.GetId())

	assert.Equal(s.T(), config.GetName(), created.GetName())
	assert.Equal(s.T(), apiV2.ReportConfiguration_NODE_VULNERABILITY, created.GetType())

	scope := created.GetResourceScope().GetEntityScope()
	s.Require().NotNil(scope, "entity scope should be set on the created config")
	s.Require().Len(scope.GetRules(), 1)

	rule := scope.GetRules()[0]
	assert.Equal(s.T(), apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER, rule.GetEntity())
	assert.Equal(s.T(), apiV2.ScopeField_FIELD_NAME, rule.GetField())
	s.Require().Len(rule.GetValues(), 1)
	assert.Equal(s.T(), "remote", rule.GetValues()[0].GetValue())
	assert.Equal(s.T(), apiV2.MatchType_EXACT, rule.GetValues()[0].GetMatchType())

	filters := created.GetNodeVulnReportFilters()
	s.Require().NotNil(filters)
	assert.Equal(s.T(), "CVSS:>=7", filters.GetQuery())
	assert.True(s.T(), filters.GetAllVuln())

	getCtx, getCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer getCancel()

	fetched, err := s.service.GetNodeReportConfiguration(getCtx, &apiV2.ResourceByID{Id: created.GetId()})
	s.Require().NoError(err, "fetching created report config")
	assert.Equal(s.T(), created.GetId(), fetched.GetId())
	assert.Equal(s.T(), "CVSS:>=7", fetched.GetNodeVulnReportFilters().GetQuery())
}

func (s *NodeReportSuite) TestUpdateNodeReportConfig() {
	config := s.newNodeReportConfig("e2e-node-report-update", "CVSS:>=7")

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	created, err := s.service.PostNodeReportConfiguration(ctx, config)
	s.Require().NoError(err)
	s.configIDs = append(s.configIDs, created.GetId())

	updated, ok := proto.Clone(created).(*apiV2.ReportConfiguration)
	s.Require().True(ok)
	updated.Id = created.GetId()
	updated.Description = "Updated description"
	updated.GetNodeVulnReportFilters().Query = "Severity:CRITICAL"
	updated.ResourceScope = &apiV2.ResourceScope{
		ScopeReference: &apiV2.ResourceScope_EntityScope{
			EntityScope: &apiV2.EntityScope{
				Rules: []*apiV2.EntityScopeRule{
					{
						Entity: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
						Field:  apiV2.ScopeField_FIELD_NAME,
						Values: []*apiV2.RuleValue{
							{Value: "prod-.*", MatchType: apiV2.MatchType_REGEX},
						},
					},
				},
			},
		},
	}

	updateCtx, updateCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer updateCancel()

	_, err = s.service.UpdateNodeReportConfiguration(updateCtx, updated)
	s.Require().NoError(err, "updating node report config")

	getCtx, getCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer getCancel()

	fetched, err := s.service.GetNodeReportConfiguration(getCtx, &apiV2.ResourceByID{Id: created.GetId()})
	s.Require().NoError(err)

	assert.Equal(s.T(), "Updated description", fetched.GetDescription())
	assert.Equal(s.T(), "Severity:CRITICAL", fetched.GetNodeVulnReportFilters().GetQuery())

	fetchedScope := fetched.GetResourceScope().GetEntityScope()
	s.Require().NotNil(fetchedScope)
	s.Require().Len(fetchedScope.GetRules(), 1)

	fetchedRule := fetchedScope.GetRules()[0]
	assert.Equal(s.T(), apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER, fetchedRule.GetEntity())
	assert.Equal(s.T(), apiV2.ScopeField_FIELD_NAME, fetchedRule.GetField())
	s.Require().Len(fetchedRule.GetValues(), 1)
	assert.Equal(s.T(), "prod-.*", fetchedRule.GetValues()[0].GetValue())
	assert.Equal(s.T(), apiV2.MatchType_REGEX, fetchedRule.GetValues()[0].GetMatchType())
}

func (s *NodeReportSuite) TestListAndCountNodeReportConfigs() {
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	config1, err := s.service.PostNodeReportConfiguration(ctx, s.newNodeReportConfig("e2e-node-list-1", "CVSS:>=7"))
	s.Require().NoError(err)
	s.configIDs = append(s.configIDs, config1.GetId())

	config2, err := s.service.PostNodeReportConfiguration(ctx, s.newNodeReportConfig("e2e-node-list-2", "CVSS:>=8"))
	s.Require().NoError(err)
	s.configIDs = append(s.configIDs, config2.GetId())

	config3, err := s.service.PostNodeReportConfiguration(ctx, s.newNodeReportConfig("e2e-node-list-3", "CVSS:>=9"))
	s.Require().NoError(err)
	s.configIDs = append(s.configIDs, config3.GetId())

	listCtx, listCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer listCancel()

	listResp, err := s.service.ListNodeReportConfigurations(listCtx, &apiV2.RawQuery{})
	s.Require().NoError(err)
	s.Require().GreaterOrEqual(len(listResp.GetReportConfigs()), 3, "should have at least 3 configs")

	countCtx, countCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer countCancel()

	countResp, err := s.service.CountNodeReportConfigurations(countCtx, &apiV2.RawQuery{})
	s.Require().NoError(err)
	s.Require().GreaterOrEqual(countResp.GetCount(), int32(3), "count should be at least 3")

	for _, config := range listResp.GetReportConfigs() {
		assert.Equal(s.T(), apiV2.ReportConfiguration_NODE_VULNERABILITY, config.GetType(),
			"list should only return NODE_VULNERABILITY type configs")
	}
}

func (s *NodeReportSuite) TestRunAndDownloadNodeReport() {
	config := s.newNodeReportConfig("e2e-node-report-run", "CVSS:>=7")

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	created, err := s.service.PostNodeReportConfiguration(ctx, config)
	s.Require().NoError(err)
	s.configIDs = append(s.configIDs, created.GetId())

	runCtx, runCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer runCancel()

	runResp, err := s.service.RunNodeReport(runCtx, &apiV2.RunReportRequest{
		ReportConfigId:           created.GetId(),
		ReportNotificationMethod: apiV2.NotificationMethod_DOWNLOAD,
	})
	s.Require().NoError(err, "submitting node report run request")
	s.Require().NotEmpty(runResp.GetReportId())

	reportID := runResp.GetReportId()
	s.T().Logf("Node report job submitted: config=%s report=%s", created.GetId(), reportID)

	s.waitForReportCompletion(reportID)
	s.downloadReport(reportID)
}

func (s *NodeReportSuite) TestNodeReportHistory() {
	config := s.newNodeReportConfig("e2e-node-report-history", "CVSS:>=7")

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	created, err := s.service.PostNodeReportConfiguration(ctx, config)
	s.Require().NoError(err)
	s.configIDs = append(s.configIDs, created.GetId())

	runCtx, runCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer runCancel()

	runResp, err := s.service.RunNodeReport(runCtx, &apiV2.RunReportRequest{
		ReportConfigId:           created.GetId(),
		ReportNotificationMethod: apiV2.NotificationMethod_DOWNLOAD,
	})
	s.Require().NoError(err)

	s.waitForReportCompletion(runResp.GetReportId())

	histCtx, histCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer histCancel()

	histResp, err := s.service.GetNodeReportHistory(histCtx, &apiV2.GetReportHistoryRequest{
		Id: created.GetId(),
	})
	s.Require().NoError(err)
	s.Require().NotEmpty(histResp.GetReportSnapshots(), "should have at least one snapshot in history")

	snapshot := histResp.GetReportSnapshots()[0]
	assert.Equal(s.T(), created.GetId(), snapshot.GetReportConfigId())

	snapshotScope := snapshot.GetResourceScope()
	s.Require().NotNil(snapshotScope, "snapshot should have resource scope")
	entityScope := snapshotScope.GetEntityScope()
	s.Require().NotNil(entityScope, "snapshot should have entity scope")
	assert.NotEmpty(s.T(), entityScope.GetRules())

	snapshotFilters := snapshot.GetNodeVulnReportFilters()
	s.Require().NotNil(snapshotFilters, "snapshot should have node vuln report filters")
	assert.Equal(s.T(), "CVSS:>=7", snapshotFilters.GetQuery())
}

func (s *NodeReportSuite) TestMyNodeReportHistory() {
	config := s.newNodeReportConfig("e2e-node-report-my-history", "CVSS:>=7")

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	created, err := s.service.PostNodeReportConfiguration(ctx, config)
	s.Require().NoError(err)
	s.configIDs = append(s.configIDs, created.GetId())

	runCtx, runCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer runCancel()

	runResp, err := s.service.RunNodeReport(runCtx, &apiV2.RunReportRequest{
		ReportConfigId:           created.GetId(),
		ReportNotificationMethod: apiV2.NotificationMethod_DOWNLOAD,
	})
	s.Require().NoError(err)

	s.waitForReportCompletion(runResp.GetReportId())

	histCtx, histCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer histCancel()

	histResp, err := s.service.GetMyNodeReportHistory(histCtx, &apiV2.GetReportHistoryRequest{
		Id: created.GetId(),
	})
	s.Require().NoError(err)
	s.Require().NotEmpty(histResp.GetReportSnapshots(), "should have at least one snapshot in user history")

	snapshot := histResp.GetReportSnapshots()[0]
	assert.Equal(s.T(), created.GetId(), snapshot.GetReportConfigId())
	s.Require().NotNil(snapshot.GetUser(), "snapshot should have user information")
}

func (s *NodeReportSuite) TestInvalidEntityScope_NonClusterType() {
	config := s.newNodeReportConfig("e2e-node-invalid-entity", "CVSS:>=7")
	config.ResourceScope = &apiV2.ResourceScope{
		ScopeReference: &apiV2.ResourceScope_EntityScope{
			EntityScope: &apiV2.EntityScope{
				Rules: []*apiV2.EntityScopeRule{
					{
						Entity: apiV2.ScopeEntity_SCOPE_ENTITY_DEPLOYMENT,
						Field:  apiV2.ScopeField_FIELD_NAME,
						Values: []*apiV2.RuleValue{
							{Value: "test", MatchType: apiV2.MatchType_EXACT},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	_, err := s.service.PostNodeReportConfiguration(ctx, config)
	s.Require().Error(err, "should reject config with deployment entity type")
}

func (s *NodeReportSuite) TestInvalidEntityScope_UnsupportedField() {
	config := s.newNodeReportConfig("e2e-node-invalid-field", "CVSS:>=7")
	config.ResourceScope = &apiV2.ResourceScope{
		ScopeReference: &apiV2.ResourceScope_EntityScope{
			EntityScope: &apiV2.EntityScope{
				Rules: []*apiV2.EntityScopeRule{
					{
						Entity: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
						Field:  apiV2.ScopeField_FIELD_ANNOTATION,
						Values: []*apiV2.RuleValue{
							{Value: "key=value", MatchType: apiV2.MatchType_EXACT},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	_, err := s.service.PostNodeReportConfiguration(ctx, config)
	s.Require().Error(err, "should reject config with annotation field")
}

func (s *NodeReportSuite) TestDeleteNodeReportConfig() {
	config := s.newNodeReportConfig("e2e-node-delete", "CVSS:>=7")

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	created, err := s.service.PostNodeReportConfiguration(ctx, config)
	s.Require().NoError(err)

	deleteCtx, deleteCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer deleteCancel()

	_, err = s.service.DeleteNodeReportConfiguration(deleteCtx, &apiV2.ResourceByID{Id: created.GetId()})
	s.Require().NoError(err, "deleting node report config")

	getCtx, getCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer getCancel()

	_, err = s.service.GetNodeReportConfiguration(getCtx, &apiV2.ResourceByID{Id: created.GetId()})
	s.Require().Error(err, "should return error when getting deleted config")
}

func (s *NodeReportSuite) TestDeleteNodeReportConfig_WithRunningJob() {
	config := s.newNodeReportConfig("e2e-node-delete-running", "CVSS:>=7")

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	created, err := s.service.PostNodeReportConfiguration(ctx, config)
	s.Require().NoError(err)
	s.configIDs = append(s.configIDs, created.GetId())

	runCtx, runCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer runCancel()

	_, err = s.service.RunNodeReport(runCtx, &apiV2.RunReportRequest{
		ReportConfigId:           created.GetId(),
		ReportNotificationMethod: apiV2.NotificationMethod_DOWNLOAD,
	})
	s.Require().NoError(err)

	deleteCtx, deleteCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer deleteCancel()

	_, err = s.service.DeleteNodeReportConfiguration(deleteCtx, &apiV2.ResourceByID{Id: created.GetId()})
	s.Require().Error(err, "should not allow deleting config with running job")
}

func (s *NodeReportSuite) TestCancelNodeReport() {
	config := s.newNodeReportConfig("e2e-node-cancel", "CVSS:>=7")

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	created, err := s.service.PostNodeReportConfiguration(ctx, config)
	s.Require().NoError(err)
	s.configIDs = append(s.configIDs, created.GetId())

	runCtx, runCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer runCancel()

	runResp, err := s.service.RunNodeReport(runCtx, &apiV2.RunReportRequest{
		ReportConfigId:           created.GetId(),
		ReportNotificationMethod: apiV2.NotificationMethod_DOWNLOAD,
	})
	s.Require().NoError(err)

	cancelCtx, cancelCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancelCancel()

	_, err = s.service.CancelNodeReport(cancelCtx, &apiV2.ResourceByID{Id: runResp.GetReportId()})
	if err != nil {
		s.T().Logf("Cancel returned error (job may have already finished): %v", err)
	}

	statusCtx, statusCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer statusCancel()

	statusResp, err := s.service.GetNodeReportStatus(statusCtx, &apiV2.ResourceByID{Id: runResp.GetReportId()})
	if err != nil {
		s.T().Logf("Report not found after cancel: %v", err)
		return
	}

	reportStatus := statusResp.GetStatus()
	s.T().Logf("Report state after cancel: %s", reportStatus.GetRunState())
	switch reportStatus.GetRunState() {
	case apiV2.ReportStatus_FAILURE:
		s.Equal("report cancelled by user", reportStatus.GetErrorMsg(), "cancelled report should have cancellation message")
	case apiV2.ReportStatus_GENERATED, apiV2.ReportStatus_DELIVERED:
		s.T().Log("report completed before cancel took effect")
	default:
		s.Failf("unexpected report state after cancel", "got %s: %s", reportStatus.GetRunState(), reportStatus.GetErrorMsg())
	}
}

func (s *NodeReportSuite) TestQueryFiltering() {
	config := s.newNodeReportConfig("e2e-node-query-filter", "CVE:CVE-2024-*+CVSS:>=9")

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	created, err := s.service.PostNodeReportConfiguration(ctx, config)
	s.Require().NoError(err)
	s.configIDs = append(s.configIDs, created.GetId())

	runCtx, runCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer runCancel()

	runResp, err := s.service.RunNodeReport(runCtx, &apiV2.RunReportRequest{
		ReportConfigId:           created.GetId(),
		ReportNotificationMethod: apiV2.NotificationMethod_DOWNLOAD,
	})
	s.Require().NoError(err)

	s.waitForReportCompletion(runResp.GetReportId())

	getCtx, getCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer getCancel()

	fetched, err := s.service.GetNodeReportConfiguration(getCtx, &apiV2.ResourceByID{Id: created.GetId()})
	s.Require().NoError(err)
	assert.Equal(s.T(), "CVE:CVE-2024-*+CVSS:>=9", fetched.GetNodeVulnReportFilters().GetQuery(),
		"query filter should be preserved")
}

func (s *NodeReportSuite) waitForReportCompletion(reportID string) {
	s.T().Logf("Waiting for node report %s to complete...", reportID)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(5 * time.Minute)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
			statusResp, err := s.service.GetNodeReportStatus(ctx, &apiV2.ResourceByID{Id: reportID})
			cancel()
			if err != nil {
				s.T().Logf("Error checking report status: %v", err)
				continue
			}

			state := statusResp.GetStatus().GetRunState()
			s.T().Logf("Node report %s state: %s", reportID, state)

			switch state {
			case apiV2.ReportStatus_GENERATED, apiV2.ReportStatus_DELIVERED:
				s.T().Logf("Node report %s completed successfully", reportID)
				return
			case apiV2.ReportStatus_FAILURE:
				s.Require().Failf("Node report generation failed",
					"Report %s failed: %s", reportID, statusResp.GetStatus().GetErrorMsg())
				return
			}
		case <-timer.C:
			s.Require().Failf("Timed out", "Node report %s did not complete within 5 minutes", reportID)
			return
		}
	}
}

func (s *NodeReportSuite) downloadReport(reportID string) {
	endpoint := centralgrpc.RoxAPIEndpoint(s.T())
	password := centralgrpc.RoxPassword(s.T())
	username := centralgrpc.RoxUsername(s.T())

	downloadURL := fmt.Sprintf("https://%s/api/reports/node/jobs/download?id=%s", endpoint, reportID)
	s.T().Logf("Downloading node report from %s", downloadURL)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 60 * time.Second,
	}

	var resp *http.Response
	s.Require().Eventually(func() bool {
		req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, downloadURL, nil)
		s.Require().NoError(err)
		req.SetBasicAuth(username, password)

		newResp, err := client.Do(req)
		if err != nil {
			s.T().Logf("Download attempt failed: %v", err)
			return false
		}
		if newResp.StatusCode != http.StatusOK {
			s.T().Logf("Download attempt returned %d", newResp.StatusCode)
			_ = newResp.Body.Close()
			return false
		}
		resp = newResp
		return true
	}, 2*time.Minute, 5*time.Second, "failed to download node report after retries")
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	require.Equal(s.T(), http.StatusOK, resp.StatusCode,
		"expected 200 OK downloading node report, got %d: %s", resp.StatusCode, string(body))
	assert.Equal(s.T(), "application/zip", resp.Header.Get("Content-Type"))
	assert.NotEmpty(s.T(), body, "downloaded node report should not be empty")

	s.T().Logf("Downloaded node report: %d bytes, Content-Type: %s", len(body), resp.Header.Get("Content-Type"))
}
