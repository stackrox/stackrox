package resolvers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/graph-gophers/graphql-go"
	imageCVEV2Mocks "github.com/stackrox/rox/central/cve/image/v2/datastore/mocks"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	imageComponentV2Mocks "github.com/stackrox/rox/central/imagecomponent/v2/datastore/mocks"
	"github.com/stackrox/rox/central/views/imagecomponentflat"
	imageComponentFlatViewMocks "github.com/stackrox/rox/central/views/imagecomponentflat/mocks"
	"github.com/stackrox/rox/central/views/imagecveflat"
	imageCVEFlatViewMocks "github.com/stackrox/rox/central/views/imagecveflat/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// These tests were created to investigate the failures that occurred
// during the upgrade of graphql-go from 1.5.0 to the next version
// (eventually carried to 1.10.2).
//
// A mocked unit-test was later extracted to ease the trace of the issue
// (see image_component_stripped_graphql_test.go).

const (
	// GraphQL query matching the user's request
	getFixableCVEsForEntityQuery = `
		query getFixableCvesForEntity($id: ID!, $scopeQuery: String, $vulnQuery: String) {
			result: imageComponent(id: $id) {
				vulnerabilities: imageVulnerabilities(
					query: $vulnQuery
					scopeQuery: $scopeQuery
				) {
					cve
					cvss
					severity
					fixedByVersion
				}
			}
		}
	`
)

var (
	systemdStorageComponent = &storage.ImageComponentV2{
		Id:        "systemd#0#4cd5259a-d1fc-5c81-ab1a-92484311441e",
		Name:      "systemd",
		Version:   "249.11-0ubuntu3.11",
		Priority:  1,
		Source:    storage.SourceType_OS,
		RiskScore: 0,
		SetTopCvss: &storage.ImageComponentV2_TopCvss{
			TopCvss: 5.5,
		},
		OperatingSystem: "ubuntu:22.04",
		ImageIdV2:       "4cd5259a-d1fc-5c81-ab1a-92484311441e",
		FromBaseImage:   false,
		LayerType:       storage.LayerType_APPLICATION,
	}

	systemdFlatComponent = &flatComponentV2{
		component:       "systemd",
		componentIDs:    []string{"systemd#0#4cd5259a-d1fc-5c81-ab1a-92484311441e"},
		version:         "249.11-0ubuntu3.11",
		operatingSystem: "ubuntu:22.04",
		riskScore:       0,
		topCVSS:         5.5,
	}

	cve2023x7008 = &storage.ImageCVEV2{
		Id: "CVE-2023-7008#0#systemd#0#4cd5259a-d1fc-5c81-ab1a-92484311441e",
		CveBaseInfo: &storage.CVEInfo{
			Cve:          "CVE-2023-7008",
			CreatedAt:    protocompat.TimestampNow(),
			ScoreVersion: storage.CVEInfo_V2,
		},
		Cvss:                 5.5,
		Severity:             storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		NvdScoreVersion:      storage.CvssScoreVersion_UNKNOWN_VERSION,
		FirstImageOccurrence: protocompat.TimestampNow(),
		State:                storage.VulnerabilityState_OBSERVED,
		IsFixable:            true,
		HasFixedBy: &storage.ImageCVEV2_FixedBy{
			FixedBy: "249.11-0ubuntu3.12",
		},
		ComponentId: "systemd#0#4cd5259a-d1fc-5c81-ab1a-92484311441e",
		ImageIdV2:   "4cd5259a-d1fc-5c81-ab1a-92484311441e",
	}

	now           = time.Now()
	moderateVuln  = storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
	observedState = storage.VulnerabilityState_OBSERVED

	flatCVE2023x7008 = &flatCVEV2{
		cve:                     "CVE-2023-7008",
		cveIDs:                  []string{"CVE-2023-7008#0#systemd#0#4cd5259a-d1fc-5c81-ab1a-92484311441e"},
		severity:                &moderateVuln,
		topCVSS:                 5.5,
		affectedImageCount:      1,
		firstDiscoveredInSystem: &now,
		firstImageOccurrence:    &now,
		state:                   &observedState,
	}
)

type flatComponentV2 struct {
	component       string
	componentIDs    []string
	version         string
	operatingSystem string
	riskScore       float32
	topCVSS         float32
}

func (c *flatComponentV2) GetComponent() string       { return c.component }
func (c *flatComponentV2) GetComponentIDs() []string  { return c.componentIDs }
func (c *flatComponentV2) GetVersion() string         { return c.version }
func (c *flatComponentV2) GetTopCVSS() float32        { return c.topCVSS }
func (c *flatComponentV2) GetRiskScore() float32      { return c.riskScore }
func (c *flatComponentV2) GetOperatingSystem() string { return c.operatingSystem }

type flatCVEV2 struct {
	cve                     string
	cveIDs                  []string
	severity                *storage.VulnerabilitySeverity
	topCVSS                 float32
	topNVDCVSS              float32
	epssProbability         float32
	affectedImageCount      int
	firstDiscoveredInSystem *time.Time
	publishedDate           *time.Time
	firstImageOccurrence    *time.Time
	state                   *storage.VulnerabilityState
}

func (f *flatCVEV2) GetCVE() string                              { return f.cve }
func (f *flatCVEV2) GetCVEIDs() []string                         { return f.cveIDs }
func (f *flatCVEV2) GetSeverity() *storage.VulnerabilitySeverity { return f.severity }
func (f *flatCVEV2) GetTopCVSS() float32                         { return f.topCVSS }
func (f *flatCVEV2) GetTopNVDCVSS() float32                      { return f.topNVDCVSS }
func (f *flatCVEV2) GetEPSSProbability() float32                 { return f.epssProbability }
func (f *flatCVEV2) GetAffectedImageCount() int                  { return f.affectedImageCount }
func (f *flatCVEV2) GetFirstDiscoveredInSystem() *time.Time      { return f.firstDiscoveredInSystem }
func (f *flatCVEV2) GetPublishDate() *time.Time                  { return f.publishedDate }
func (f *flatCVEV2) GetFirstImageOccurrence() *time.Time         { return f.firstImageOccurrence }
func (f *flatCVEV2) GetState() *storage.VulnerabilityState       { return f.state }

func TestGetFixableCVEsForEntityGraphQL(t *testing.T) {
	suite.Run(t, new(ImageComponentCVEGraphQLTestSuite))
}

type ImageComponentCVEGraphQLTestSuite struct {
	suite.Suite

	ctx      context.Context
	resolver *Resolver
	schema   *graphql.Schema

	mockCtrl *gomock.Controller

	imageComponentDS       *imageComponentV2Mocks.MockDataStore
	imageComponentFlatView *imageComponentFlatViewMocks.MockComponentFlatView
	imageCVEDS             *imageCVEV2Mocks.MockDataStore
	imageCVEFlatView       *imageCVEFlatViewMocks.MockCveFlatView
}

func (s *ImageComponentCVEGraphQLTestSuite) SetupSuite() {
	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	s.mockCtrl = gomock.NewController(s.T())

	// TODO(ROX-30117): Remove conditional when FlattenImageData feature flag is removed.
	var resolver *Resolver
	s.imageComponentDS = imageComponentV2Mocks.NewMockDataStore(s.mockCtrl)
	s.imageComponentFlatView = imageComponentFlatViewMocks.NewMockComponentFlatView(s.mockCtrl)
	s.imageCVEDS = imageCVEV2Mocks.NewMockDataStore(s.mockCtrl)
	s.imageCVEFlatView = imageCVEFlatViewMocks.NewMockCveFlatView(s.mockCtrl)
	resolver, _ = SetupTestResolver(s.T(),
		s.imageComponentDS,
		s.imageCVEDS,
		s.imageCVEFlatView,
		s.imageComponentFlatView,
	)
	s.resolver = resolver

	// Parse the GraphQL schema
	var err error
	s.schema, err = graphql.ParseSchema(Schema(), s.resolver)
	s.Require().NoError(err)
}

// Response structure matching the GraphQL query
type cveResponse struct {
	CVE            string  `json:"cve"`
	CVSS           float64 `json:"cvss"`
	Severity       string  `json:"severity"`
	FixedByVersion string  `json:"fixedByVersion"`
}

type imageComponentResponse struct {
	Vulnerabilities []cveResponse `json:"vulnerabilities"`
}

type queryResponse struct {
	Result imageComponentResponse `json:"result"`
}

// TestGetFixableCVEsForEntityWithGraphQLEngine validates the GraphQL query by executing it
// through the GraphQL engine (graph-gophers/graphql-go) rather than calling resolver functions directly.
// This tests the full query execution path including parsing, validation, and execution.
func (s *ImageComponentCVEGraphQLTestSuite) TestGetFixableCVEsForEntityWithGraphQLEngine() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	// Step 1: Find the systemd component ID using a separate query
	findComponentQuery := `
		query findComponent($query: String) {
			components: imageComponents(query: $query) {
				id
			}
		}
	`

	s.imageComponentFlatView.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return([]imagecomponentflat.ComponentFlat{systemdFlatComponent}, nil)
	s.imageComponentDS.EXPECT().
		SearchRawImageComponents(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return([]*storage.ImageComponentV2{systemdStorageComponent}, nil)
	findResponse := s.schema.Exec(ctx, findComponentQuery, "findComponent",
		map[string]interface{}{
			"query": "Component:systemd+Component Version:249.11-0ubuntu3.11",
		})

	s.Require().Empty(findResponse.Errors, "Finding component should not produce errors")

	var findResult struct {
		Components []struct {
			ID string `json:"id"`
		} `json:"components"`
	}
	s.Require().NoError(json.Unmarshal(findResponse.Data, &findResult))
	s.Require().NotEmpty(findResult.Components, "Should find systemd component")

	componentID := findResult.Components[0].ID
	s.T().Logf("Found systemd component with ID: %s", componentID)

	s.T().Run("query without filters", func(t *testing.T) {
		// Execute the GraphQL query without vulnerability filters
		s.imageComponentDS.EXPECT().GetBatch(gomock.Any(), gomock.Any()).AnyTimes().Return([]*storage.ImageComponentV2{systemdStorageComponent}, nil)
		s.imageCVEDS.EXPECT().Search(gomock.Any(), gomock.Any()).AnyTimes().Return([]searchPkg.Result{{ID: cve2023x7008.GetId()}}, nil)
		s.imageCVEDS.EXPECT().GetBatch(gomock.Any(), gomock.Any()).AnyTimes().Return([]*storage.ImageCVEV2{cve2023x7008}, nil)
		s.imageCVEDS.EXPECT().SearchRawImageCVEs(gomock.Any(), gomock.Any()).AnyTimes().Return([]*storage.ImageCVEV2{cve2023x7008}, nil)
		s.imageCVEFlatView.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]imagecveflat.CveFlat{flatCVE2023x7008}, nil)
		response := s.schema.Exec(ctx, getFixableCVEsForEntityQuery, "getFixableCvesForEntity",
			map[string]interface{}{
				"id":         componentID,
				"vulnQuery":  nil,
				"scopeQuery": nil,
			})

		// Check for GraphQL errors
		if len(response.Errors) > 0 {
			for _, err := range response.Errors {
				t.Logf("GraphQL Error: %s", err.Error())
			}
		}
		assert.Empty(t, response.Errors, "Query should not produce errors")

		// Parse the response
		var result queryResponse
		require.NoError(t, json.Unmarshal(response.Data, &result))

		// Validate we got results
		assert.NotEmpty(t, result.Result.Vulnerabilities, "Should have at least one vulnerability")

		// Validate CVE-2023-7008 is present
		foundCVE := false
		for _, vuln := range result.Result.Vulnerabilities {
			t.Logf("Found CVE: %s, CVSS: %f, Severity: %s, Fixed: %s",
				vuln.CVE, vuln.CVSS, vuln.Severity, vuln.FixedByVersion)

			if vuln.CVE == "CVE-2023-7008" {
				foundCVE = true
				assert.Equal(t, 5.5, vuln.CVSS, "CVSS should match")
				assert.Equal(t, "MODERATE_VULNERABILITY_SEVERITY", vuln.Severity, "Severity should match")
				assert.Equal(t, "249.11-0ubuntu3.12", vuln.FixedByVersion, "Fix version should match")
			}
		}
		assert.True(t, foundCVE, "CVE-2023-7008 should be found in results")
	})

	s.T().Run("query with vulnQuery filter", func(t *testing.T) {
		// Execute the GraphQL query with CVE filter
		s.imageCVEDS.EXPECT().Search(gomock.Any(), gomock.Any()).AnyTimes().Return([]searchPkg.Result{{ID: cve2023x7008.GetId()}}, nil)
		s.imageCVEDS.EXPECT().SearchRawImageCVEs(gomock.Any(), gomock.Any()).AnyTimes().Return([]*storage.ImageCVEV2{cve2023x7008}, nil)
		s.imageCVEFlatView.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]imagecveflat.CveFlat{flatCVE2023x7008}, nil)
		response := s.schema.Exec(ctx, getFixableCVEsForEntityQuery, "getFixableCvesForEntity",
			map[string]interface{}{
				"id":         componentID,
				"vulnQuery":  "CVE:CVE-2023-7008",
				"scopeQuery": nil,
			})

		assert.Empty(t, response.Errors, "Query should not produce errors")

		// Parse the response
		var result queryResponse
		require.NoError(t, json.Unmarshal(response.Data, &result))

		// Should find exactly one CVE when filtered
		require.Len(t, result.Result.Vulnerabilities, 1, "Should find exactly one CVE")

		vuln := result.Result.Vulnerabilities[0]
		assert.Equal(t, "CVE-2023-7008", vuln.CVE)
		assert.Equal(t, 5.5, vuln.CVSS)
		assert.Equal(t, "MODERATE_VULNERABILITY_SEVERITY", vuln.Severity)
		assert.Equal(t, "249.11-0ubuntu3.12", vuln.FixedByVersion)

		t.Logf("Successfully validated CVE-2023-7008: CVSS=%f, Severity=%s, Fixed=%s",
			vuln.CVSS, vuln.Severity, vuln.FixedByVersion)
	})

	s.T().Run("query with fixable filter", func(t *testing.T) {
		// Execute the GraphQL query with fixable filter
		s.imageCVEDS.EXPECT().Search(gomock.Any(), gomock.Any()).AnyTimes().Return([]searchPkg.Result{{ID: cve2023x7008.GetId()}}, nil)
		s.imageCVEDS.EXPECT().SearchRawImageCVEs(gomock.Any(), gomock.Any()).AnyTimes().Return([]*storage.ImageCVEV2{cve2023x7008}, nil)
		s.imageCVEFlatView.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]imagecveflat.CveFlat{flatCVE2023x7008}, nil)
		response := s.schema.Exec(ctx, getFixableCVEsForEntityQuery, "getFixableCvesForEntity",
			map[string]interface{}{
				"id":         componentID,
				"vulnQuery":  "Fixable:true",
				"scopeQuery": nil,
			})

		assert.Empty(t, response.Errors, "Query should not produce errors")

		// Parse the response
		var result queryResponse
		require.NoError(t, json.Unmarshal(response.Data, &result))

		assert.NotEmpty(t, result.Result.Vulnerabilities, "Should find fixable vulnerabilities")

		// CVE-2023-7008 has a fixedBy version, so it should be fixable
		foundFixable := false
		for _, vuln := range result.Result.Vulnerabilities {
			if vuln.CVE == "CVE-2023-7008" {
				foundFixable = true
				assert.Equal(t, "249.11-0ubuntu3.12", vuln.FixedByVersion)
				t.Logf("CVE-2023-7008 is fixable by version: %s", vuln.FixedByVersion)
			}
		}
		assert.True(t, foundFixable, "CVE-2023-7008 should be in fixable results")
	})

	s.T().Run("query with combined filters", func(t *testing.T) {
		// Execute the GraphQL query with combined CVE and fixable filters
		s.imageCVEDS.EXPECT().Search(gomock.Any(), gomock.Any()).AnyTimes().Return([]searchPkg.Result{{ID: cve2023x7008.GetId()}}, nil)
		s.imageCVEDS.EXPECT().SearchRawImageCVEs(gomock.Any(), gomock.Any()).AnyTimes().Return([]*storage.ImageCVEV2{cve2023x7008}, nil)
		s.imageCVEFlatView.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]imagecveflat.CveFlat{flatCVE2023x7008}, nil)
		response := s.schema.Exec(ctx, getFixableCVEsForEntityQuery, "getFixableCvesForEntity",
			map[string]interface{}{
				"id":         componentID,
				"vulnQuery":  "CVE:CVE-2023-7008+Fixable:true",
				"scopeQuery": nil,
			})

		assert.Empty(t, response.Errors, "Query should not produce errors")

		// Parse the response
		var result queryResponse
		require.NoError(t, json.Unmarshal(response.Data, &result))

		require.Len(t, result.Result.Vulnerabilities, 1, "Should find exactly one fixable CVE-2023-7008")

		vuln := result.Result.Vulnerabilities[0]
		assert.Equal(t, "CVE-2023-7008", vuln.CVE)
		assert.Equal(t, 5.5, vuln.CVSS)
		assert.Equal(t, "MODERATE_VULNERABILITY_SEVERITY", vuln.Severity)
		assert.Equal(t, "249.11-0ubuntu3.12", vuln.FixedByVersion)
	})

	s.T().Run("query with invalid component ID", func(t *testing.T) {
		// Test error handling with invalid component ID
		response := s.schema.Exec(ctx, getFixableCVEsForEntityQuery, "getFixableCvesForEntity",
			map[string]interface{}{
				"id":         "invalid-component-id",
				"vulnQuery":  nil,
				"scopeQuery": nil,
			})

		// Should have errors for invalid component
		assert.NotEmpty(t, response.Errors, "Query with invalid ID should produce errors")

		if len(response.Errors) > 0 {
			t.Logf("Expected error: %s", response.Errors[0].Error())
		}
	})
}

// TestGraphQLVariableTypes validates that the GraphQL engine properly handles different variable types
func (s *ImageComponentCVEGraphQLTestSuite) TestGraphQLVariableTypes() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	s.T().Run("null variables", func(t *testing.T) {
		// Find component first
		findComponentQuery := `
			query findComponent {
				components: imageComponents(query: "Component:systemd") {
					id
				}
			}
		`
		s.imageCVEDS.EXPECT().Search(gomock.Any(), gomock.Any()).AnyTimes().Return([]searchPkg.Result{{ID: cve2023x7008.GetId()}}, nil)
		s.imageCVEDS.EXPECT().SearchRawImageCVEs(gomock.Any(), gomock.Any()).AnyTimes().Return([]*storage.ImageCVEV2{cve2023x7008}, nil)
		s.imageCVEFlatView.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]imagecveflat.CveFlat{flatCVE2023x7008}, nil)
		findResponse := s.schema.Exec(ctx, findComponentQuery, "findComponent", nil)
		require.Empty(t, findResponse.Errors)

		var findResult struct {
			Components []struct {
				ID string `json:"id"`
			} `json:"components"`
		}
		require.NoError(t, json.Unmarshal(findResponse.Data, &findResult))
		require.NotEmpty(t, findResult.Components)

		componentID := findResult.Components[0].ID

		// Test with nil/null variables - GraphQL should handle this gracefully
		response := s.schema.Exec(ctx, getFixableCVEsForEntityQuery, "getFixableCvesForEntity",
			map[string]interface{}{
				"id": componentID,
				// vulnQuery and scopeQuery are optional, so nil is valid
			})

		assert.Empty(t, response.Errors, "Query with nil optional variables should work")

		var result queryResponse
		require.NoError(t, json.Unmarshal(response.Data, &result))
		assert.NotEmpty(t, result.Result.Vulnerabilities)
	})
}
