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
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
)

func TestReportEntityScope(t *testing.T) {
	suite.Run(t, new(ReportEntityScopeSuite))
}

type ReportEntityScopeSuite struct {
	suite.Suite

	ctx       context.Context
	cancel    context.CancelFunc
	service   apiV2.ReportServiceClient
	configIDs []string
}

func (s *ReportEntityScopeSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithTimeout(context.Background(), 10*time.Minute)
	conn := centralgrpc.GRPCConnectionToCentral(s.T())
	s.service = apiV2.NewReportServiceClient(conn)
}

func (s *ReportEntityScopeSuite) TearDownSuite() {
	defer s.cancel()
	for _, id := range s.configIDs {
		ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
		_, err := s.service.DeleteReportConfiguration(ctx, &apiV2.ResourceByID{Id: id})
		cancel()
		if err != nil {
			s.T().Logf("Failed to delete report config %s: %v", id, err)
		}
	}
}

func (s *ReportEntityScopeSuite) newEntityScopeConfig(name string) *apiV2.ReportConfiguration {
	return &apiV2.ReportConfiguration{
		Name:        name,
		Description: "E2E test report with entity scope",
		Type:        apiV2.ReportConfiguration_VULNERABILITY,
		Filter: &apiV2.ReportConfiguration_VulnReportFilters{
			VulnReportFilters: &apiV2.VulnerabilityReportFilters{
				ImageTypes: []apiV2.VulnerabilityReportFilters_ImageType{
					apiV2.VulnerabilityReportFilters_DEPLOYED,
				},
				CvesSince: &apiV2.VulnerabilityReportFilters_AllVuln{AllVuln: true},
				Query:     "CVSS:>=7+Fixable:true",
			},
		},
		ResourceScope: &apiV2.ResourceScope{
			ScopeReference: &apiV2.ResourceScope_EntityScope{
				EntityScope: &apiV2.EntityScope{
					Rules: []*apiV2.EntityScopeRule{
						{
							Entity: apiV2.ScopeEntity_SCOPE_ENTITY_NAMESPACE,
							Field:  apiV2.ScopeField_FIELD_NAME,
							Values: []*apiV2.RuleValue{
								{Value: "stackrox", MatchType: apiV2.MatchType_EXACT},
							},
						},
						{
							Entity: apiV2.ScopeEntity_SCOPE_ENTITY_DEPLOYMENT,
							Field:  apiV2.ScopeField_FIELD_NAME,
							Values: []*apiV2.RuleValue{
								{Value: "scanner.*", MatchType: apiV2.MatchType_REGEX},
							},
						},
					},
				},
			},
		},
	}
}

func (s *ReportEntityScopeSuite) TestCreateReportConfigWithEntityScope() {
	config := s.newEntityScopeConfig("e2e-entity-scope-create")

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	created, err := s.service.PostReportConfiguration(ctx, config)
	s.Require().NoError(err, "creating report config with entity scope")
	s.Require().NotEmpty(created.GetId())
	s.configIDs = append(s.configIDs, created.GetId())

	assert.Equal(s.T(), config.GetName(), created.GetName())
	assert.Equal(s.T(), apiV2.ReportConfiguration_VULNERABILITY, created.GetType())

	scope := created.GetResourceScope().GetEntityScope()
	s.Require().NotNil(scope, "entity scope should be set on the created config")
	s.Require().Len(scope.GetRules(), 2)
	// verify entity scope rules
	firstRule := scope.GetRules()[0]
	assert.Equal(s.T(), apiV2.ScopeEntity_SCOPE_ENTITY_NAMESPACE, firstRule.GetEntity())
	assert.Equal(s.T(), apiV2.ScopeField_FIELD_NAME, firstRule.GetField())
	s.Require().Len(firstRule.GetValues(), 1)
	assert.Equal(s.T(), "stackrox", firstRule.GetValues()[0].GetValue())
	assert.Equal(s.T(), apiV2.MatchType_EXACT, firstRule.GetValues()[0].GetMatchType())

	secondRule := scope.GetRules()[1]
	assert.Equal(s.T(), apiV2.ScopeEntity_SCOPE_ENTITY_DEPLOYMENT, secondRule.GetEntity())
	assert.Equal(s.T(), apiV2.ScopeField_FIELD_NAME, secondRule.GetField())
	s.Require().Len(secondRule.GetValues(), 1)
	assert.Equal(s.T(), "scanner.*", secondRule.GetValues()[0].GetValue())
	assert.Equal(s.T(), apiV2.MatchType_REGEX, secondRule.GetValues()[0].GetMatchType())

	filters := created.GetVulnReportFilters()
	s.Require().NotNil(filters)
	assert.Equal(s.T(), "CVSS:>=7+Fixable:true", filters.GetQuery())

	getCtx, getCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer getCancel()

	fetched, err := s.service.GetReportConfiguration(getCtx, &apiV2.ResourceByID{Id: created.GetId()})
	s.Require().NoError(err, "fetching created report config")
	assert.Equal(s.T(), created.GetId(), fetched.GetId())
	assert.Equal(s.T(), "CVSS:>=7+Fixable:true", fetched.GetVulnReportFilters().GetQuery())

	fetchedScope := fetched.GetResourceScope().GetEntityScope()
	s.Require().NotNil(fetchedScope)
	s.Require().Len(fetchedScope.GetRules(), 2)
	// verify entity scope rules
	fetchedFirstRule := fetchedScope.GetRules()[0]
	assert.Equal(s.T(), apiV2.ScopeEntity_SCOPE_ENTITY_NAMESPACE, fetchedFirstRule.GetEntity())
	assert.Equal(s.T(), apiV2.ScopeField_FIELD_NAME, fetchedFirstRule.GetField())
	s.Require().Len(fetchedFirstRule.GetValues(), 1)
	assert.Equal(s.T(), "stackrox", fetchedFirstRule.GetValues()[0].GetValue())
	assert.Equal(s.T(), apiV2.MatchType_EXACT, fetchedFirstRule.GetValues()[0].GetMatchType())

	fetchedSecondRule := fetchedScope.GetRules()[1]
	assert.Equal(s.T(), apiV2.ScopeEntity_SCOPE_ENTITY_DEPLOYMENT, fetchedSecondRule.GetEntity())
	assert.Equal(s.T(), apiV2.ScopeField_FIELD_NAME, fetchedSecondRule.GetField())
	s.Require().Len(fetchedSecondRule.GetValues(), 1)
	assert.Equal(s.T(), "scanner.*", fetchedSecondRule.GetValues()[0].GetValue())
	assert.Equal(s.T(), apiV2.MatchType_REGEX, fetchedSecondRule.GetValues()[0].GetMatchType())
}

func (s *ReportEntityScopeSuite) TestUpdateReportConfigEntityScope() {
	config := s.newEntityScopeConfig("e2e-entity-scope-update")

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	created, err := s.service.PostReportConfiguration(ctx, config)
	s.Require().NoError(err)
	s.configIDs = append(s.configIDs, created.GetId())

	updated, ok := proto.Clone(created).(*apiV2.ReportConfiguration)
	s.Require().True(ok)
	updated.Id = created.GetId()
	updated.Description = "Updated description"
	updated.GetVulnReportFilters().Query = "CVSS:>=9+Severity:CRITICAL"
	updated.ResourceScope = &apiV2.ResourceScope{
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
	}

	updateCtx, updateCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer updateCancel()

	_, err = s.service.UpdateReportConfiguration(updateCtx, updated)
	s.Require().NoError(err, "updating report config with new entity scope")

	getCtx, getCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer getCancel()

	fetched, err := s.service.GetReportConfiguration(getCtx, &apiV2.ResourceByID{Id: created.GetId()})
	s.Require().NoError(err)

	assert.Equal(s.T(), "Updated description", fetched.GetDescription())
	assert.Equal(s.T(), "CVSS:>=9+Severity:CRITICAL", fetched.GetVulnReportFilters().GetQuery())

	fetchedScope := fetched.GetResourceScope().GetEntityScope()
	s.Require().NotNil(fetchedScope)
	s.Require().Len(fetchedScope.GetRules(), 1)
	// verify entity scope rules
	fetchedRule := fetchedScope.GetRules()[0]
	assert.Equal(s.T(), apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER, fetchedRule.GetEntity())
	assert.Equal(s.T(), apiV2.ScopeField_FIELD_NAME, fetchedRule.GetField())
	s.Require().Len(fetchedRule.GetValues(), 1)
	assert.Equal(s.T(), "remote", fetchedRule.GetValues()[0].GetValue())
	assert.Equal(s.T(), apiV2.MatchType_EXACT, fetchedRule.GetValues()[0].GetMatchType())
}

func (s *ReportEntityScopeSuite) TestRunAndDownloadReport() {
	config := s.newEntityScopeConfig("e2e-entity-scope-download")

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	created, err := s.service.PostReportConfiguration(ctx, config)
	s.Require().NoError(err)
	s.configIDs = append(s.configIDs, created.GetId())

	runCtx, runCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer runCancel()

	runResp, err := s.service.RunReport(runCtx, &apiV2.RunReportRequest{
		ReportConfigId:           created.GetId(),
		ReportNotificationMethod: apiV2.NotificationMethod_DOWNLOAD,
	})
	s.Require().NoError(err, "submitting report run request")
	s.Require().NotEmpty(runResp.GetReportId())

	reportID := runResp.GetReportId()
	s.T().Logf("Report job submitted: config=%s report=%s", created.GetId(), reportID)

	s.waitForReportCompletion(reportID)
	s.downloadReport(reportID)
}

func (s *ReportEntityScopeSuite) TestCreateConfigWithInvalidEntityScope() {
	config := s.newEntityScopeConfig("e2e-entity-scope-invalid")
	config.ResourceScope = &apiV2.ResourceScope{
		ScopeReference: &apiV2.ResourceScope_EntityScope{
			EntityScope: &apiV2.EntityScope{
				Rules: []*apiV2.EntityScopeRule{
					{
						Entity: apiV2.ScopeEntity_SCOPE_ENTITY_UNSET,
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

	_, err := s.service.PostReportConfiguration(ctx, config)
	s.Require().Error(err, "should reject config with unset entity")
}

func (s *ReportEntityScopeSuite) TestCreateConfigWithDuplicateEntityFieldPair() {
	config := s.newEntityScopeConfig("e2e-entity-scope-duplicate")
	config.ResourceScope = &apiV2.ResourceScope{
		ScopeReference: &apiV2.ResourceScope_EntityScope{
			EntityScope: &apiV2.EntityScope{
				Rules: []*apiV2.EntityScopeRule{
					{
						Entity: apiV2.ScopeEntity_SCOPE_ENTITY_DEPLOYMENT,
						Field:  apiV2.ScopeField_FIELD_NAME,
						Values: []*apiV2.RuleValue{
							{Value: "a", MatchType: apiV2.MatchType_EXACT},
						},
					},
					{
						Entity: apiV2.ScopeEntity_SCOPE_ENTITY_DEPLOYMENT,
						Field:  apiV2.ScopeField_FIELD_NAME,
						Values: []*apiV2.RuleValue{
							{Value: "b", MatchType: apiV2.MatchType_EXACT},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	_, err := s.service.PostReportConfiguration(ctx, config)
	s.Require().Error(err, "should reject config with duplicate entity+field pair")
}

func (s *ReportEntityScopeSuite) TestReportHistoryWithEntityScope() {
	config := s.newEntityScopeConfig("e2e-entity-scope-history")
	s.T().Logf("report config is %v", config)

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	created, err := s.service.PostReportConfiguration(ctx, config)
	s.Require().NoError(err)
	s.configIDs = append(s.configIDs, created.GetId())

	runCtx, runCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer runCancel()

	runResp, err := s.service.RunReport(runCtx, &apiV2.RunReportRequest{
		ReportConfigId:           created.GetId(),
		ReportNotificationMethod: apiV2.NotificationMethod_DOWNLOAD,
	})
	s.Require().NoError(err)

	s.waitForReportCompletion(runResp.GetReportId())

	histCtx, histCancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer histCancel()

	histResp, err := s.service.GetReportHistory(histCtx, &apiV2.GetReportHistoryRequest{
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

	snapshotFilters := snapshot.GetVulnReportFilters()
	s.Require().NotNil(snapshotFilters, "snapshot should have vuln report filters")
	assert.Equal(s.T(), "CVSS:>=7+Fixable:true", snapshotFilters.GetQuery())
}

func (s *ReportEntityScopeSuite) waitForReportCompletion(reportID string) {
	s.T().Logf("Waiting for report %s to complete...", reportID)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(5 * time.Minute)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
			statusResp, err := s.service.GetReportStatus(ctx, &apiV2.ResourceByID{Id: reportID})
			cancel()
			if err != nil {
				s.T().Logf("Error checking report status: %v", err)
				continue
			}

			state := statusResp.GetStatus().GetRunState()
			s.T().Logf("Report %s state: %s", reportID, state)

			switch state {
			case apiV2.ReportStatus_GENERATED, apiV2.ReportStatus_DELIVERED:
				s.T().Logf("Report %s completed successfully", reportID)
				return
			case apiV2.ReportStatus_FAILURE:
				s.Require().Failf("Report generation failed",
					"Report %s failed: %s", reportID, statusResp.GetStatus().GetErrorMsg())
				return
			}
		case <-timer.C:
			s.Require().Failf("Timed out", "Report %s did not complete within 5 minutes", reportID)
			return
		}
	}
}

func (s *ReportEntityScopeSuite) downloadReport(reportID string) {
	endpoint := centralgrpc.RoxAPIEndpoint(s.T())
	password := centralgrpc.RoxPassword(s.T())
	username := centralgrpc.RoxUsername(s.T())

	url := fmt.Sprintf("https://%s/api/reports/jobs/download?id=%s", endpoint, reportID)
	s.T().Logf("Downloading report from %s", url)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 60 * time.Second,
	}

	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, url, nil)
	s.Require().NoError(err)
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	require.Equal(s.T(), http.StatusOK, resp.StatusCode,
		"expected 200 OK downloading report, got %d: %s", resp.StatusCode, string(body))
	assert.Equal(s.T(), "application/zip", resp.Header.Get("Content-Type"))
	assert.NotEmpty(s.T(), body, "downloaded report should not be empty")

	s.T().Logf("Downloaded report: %d bytes, Content-Type: %s", len(body), resp.Header.Get("Content-Type"))
}
