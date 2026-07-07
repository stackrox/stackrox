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

	findComponentID = `
		query findComponent($query: String) {
			components: imageComponents(query: $query) {
				id
			}
		}
	`
)

// This test was extracted from TestGetImageComponentVulnerabilitiesGraphQL /
// TestGetFixableCVEsForEntityWithGraphQLEngine / query_without_filters
// for detailed tracing purpose.
func TestImageComponentGraphQLQuery(t *testing.T) {
	testCtx := loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockContainer := getImageComponentCVEMocks(mockCtrl)
	resolver := getImageComponentCVEResolver(t, mockContainer)

	schema, err := graphql.ParseSchema(Schema(), resolver)
	require.NoError(t, err)

	ctx := SetAuthorizerOverride(testCtx, allow.Anonymous())

	componentID := "systemd#0#4cd5259a-d1fc-5c81-ab1a-92484311441e"

	setupImageComponentMocks(mockCtrl, mockContainer)
	setupImageComponentCVEMocks(mockCtrl, mockContainer)

	response := schema.Exec(ctx, getFixableCVEsForEntityQuery, "getFixableCvesForEntity",
		map[string]interface{}{
			"id":         componentID,
			"vulnQuery":  nil,
			"scopeQuery": nil,
		})

	// Check for GraphQL errors
	validateResponseHasNoError(t, response)
	// Validate the response
	validateResponseContainsCVE2023x7008(t, response)
}

func TestGetImageComponentVulnerabilitiesGraphQL(t *testing.T) {
	suite.Run(t, new(ImageComponentVulnerabilitiesGraphQLTestSuite))
}

type ImageComponentVulnerabilitiesGraphQLTestSuite struct {
	suite.Suite

	ctx      context.Context
	resolver *Resolver
	schema   *graphql.Schema

	mockCtrl      *gomock.Controller
	mockContainer *imageComponentCVEMocks
}

func (s *ImageComponentVulnerabilitiesGraphQLTestSuite) SetupTest() {
	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	s.mockCtrl = gomock.NewController(s.T())
	s.mockContainer = getImageComponentCVEMocks(s.mockCtrl)
	s.resolver = getImageComponentCVEResolver(s.T(), s.mockContainer)

	// Parse the GraphQL schema
	var err error
	s.schema, err = graphql.ParseSchema(Schema(), s.resolver)
	s.Require().NoError(err)
}

func (s *ImageComponentVulnerabilitiesGraphQLTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

// TestGetFixableCVEsForEntityWithGraphQLEngine validates the GraphQL query by executing it
// through the GraphQL engine (graph-gophers/graphql-go) rather than calling resolver functions directly.
// This tests the full query execution path including parsing, validation, and execution.
func (s *ImageComponentVulnerabilitiesGraphQLTestSuite) TestGetFixableCVEsForEntityWithGraphQLEngine() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	setupImageComponentMocks(s.mockCtrl, s.mockContainer)

	// Step 1: Find the systemd component ID using a separate query
	findResponse := s.schema.Exec(ctx, findComponentID, "findComponent",
		map[string]interface{}{
			"query": "Component:systemd+Component Version:249.11-0ubuntu3.11",
		})

	validateResponseHasNoError(s.T(), findResponse)
	componentIDs := validateComponentIDResponse(s.T(), findResponse)
	require.NotEmpty(s.T(), componentIDs)
	componentID := componentIDs[0]
	s.T().Logf("Found systemd component with ID: %s", componentID)

	s.T().Run("query without filters", func(t *testing.T) {
		// Execute the GraphQL query without vulnerability filters
		setupImageComponentCVEMocks(s.mockCtrl, s.mockContainer)

		response := s.schema.Exec(ctx, getFixableCVEsForEntityQuery, "getFixableCvesForEntity",
			map[string]interface{}{
				"id":         componentID,
				"vulnQuery":  nil,
				"scopeQuery": nil,
			})

		// Check for GraphQL errors
		validateResponseHasNoError(t, response)
		// Validate the response
		validateResponseContainsCVE2023x7008(t, response)
	})

	s.T().Run("query with vulnQuery filter", func(t *testing.T) {
		// Execute the GraphQL query with CVE filter
		setupImageComponentCVEMocks(s.mockCtrl, s.mockContainer)

		response := s.schema.Exec(ctx, getFixableCVEsForEntityQuery, "getFixableCvesForEntity",
			map[string]interface{}{
				"id":         componentID,
				"vulnQuery":  "CVE:CVE-2023-7008",
				"scopeQuery": nil,
			})

		validateResponseHasNoError(t, response)
		// Should find exactly one CVE when filtered
		validateSingleVulnerabilityResponse(t, response)
		validateResponseContainsCVE2023x7008(t, response)
	})

	s.T().Run("query with fixable filter", func(t *testing.T) {
		// Execute the GraphQL query with fixable filter
		setupImageComponentCVEMocks(s.mockCtrl, s.mockContainer)

		response := s.schema.Exec(ctx, getFixableCVEsForEntityQuery, "getFixableCvesForEntity",
			map[string]interface{}{
				"id":         componentID,
				"vulnQuery":  "Fixable:true",
				"scopeQuery": nil,
			})

		validateResponseHasNoError(t, response)
		// Validate the response
		validateResponseContainsCVE2023x7008(t, response)
	})

	s.T().Run("query with combined filters", func(t *testing.T) {
		// Execute the GraphQL query with combined CVE and fixable filters
		setupImageComponentCVEMocks(s.mockCtrl, s.mockContainer)

		response := s.schema.Exec(ctx, getFixableCVEsForEntityQuery, "getFixableCvesForEntity",
			map[string]interface{}{
				"id":         componentID,
				"vulnQuery":  "CVE:CVE-2023-7008+Fixable:true",
				"scopeQuery": nil,
			})

		// Validate the response
		validateResponseHasNoError(t, response)
		validateSingleVulnerabilityResponse(t, response)
		validateResponseContainsCVE2023x7008(t, response)
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
			assert.ErrorContains(t, response.Errors[0].Unwrap(), "not all components could be found: invalid-component-id")
		}
	})
}

// region test helper functions

func getImageComponentCVEMocks(mockCtrl *gomock.Controller) *imageComponentCVEMocks {
	return &imageComponentCVEMocks{
		imageComponentDS:       imageComponentV2Mocks.NewMockDataStore(mockCtrl),
		imageComponentFlatView: imageComponentFlatViewMocks.NewMockComponentFlatView(mockCtrl),
		imageCVEDS:             imageCVEV2Mocks.NewMockDataStore(mockCtrl),
		imageCVEFlatView:       imageCVEFlatViewMocks.NewMockCveFlatView(mockCtrl),
	}
}

func getImageComponentCVEResolver(t *testing.T, mockContainer *imageComponentCVEMocks) *Resolver {
	resolver, _ := SetupTestResolver(
		t,
		mockContainer.imageComponentDS,
		mockContainer.imageCVEDS,
		mockContainer.imageCVEFlatView,
		mockContainer.imageComponentFlatView,
	)
	return resolver
}

func setupImageComponentMocks(mockCtrl *gomock.Controller, mockContainer *imageComponentCVEMocks) {
	systemdFlatComponent := imageComponentFlatViewMocks.NewMockComponentFlat(mockCtrl)
	systemdFlatComponent.EXPECT().GetComponent().AnyTimes().Return("systemd")
	systemdFlatComponent.EXPECT().GetComponentIDs().AnyTimes().Return([]string{"systemd#0#4cd5259a-d1fc-5c81-ab1a-92484311441e"})
	systemdFlatComponent.EXPECT().GetVersion().AnyTimes().Return("249.11-0ubuntu3.11")
	systemdFlatComponent.EXPECT().GetTopCVSS().AnyTimes().Return(float32(5.5))
	systemdFlatComponent.EXPECT().GetRiskScore().AnyTimes().Return(float32(0))
	systemdFlatComponent.EXPECT().GetOperatingSystem().AnyTimes().Return("ubuntu:22.04")
	systemdStorageComponent := &storage.ImageComponentV2{
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
	mockContainer.imageComponentFlatView.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		AnyTimes().Return([]imagecomponentflat.ComponentFlat{systemdFlatComponent}, nil)
	mockContainer.imageComponentDS.EXPECT().
		SearchRawImageComponents(gomock.Any(), gomock.Any()).
		AnyTimes().Return([]*storage.ImageComponentV2{systemdStorageComponent}, nil)
	mockContainer.imageComponentDS.EXPECT().
		GetBatch(gomock.Any(), gomock.Any()).
		AnyTimes().Return([]*storage.ImageComponentV2{systemdStorageComponent}, nil)
}

func setupImageComponentCVEMocks(mockCtrl *gomock.Controller, mockContainer *imageComponentCVEMocks) {
	now := new(time.Now())
	moderateVuln := new(storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY)
	observedState := new(storage.VulnerabilityState_OBSERVED)

	flatCVE2023x7008 := imageCVEFlatViewMocks.NewMockCveFlat(mockCtrl)
	flatCVE2023x7008.EXPECT().GetCVE().AnyTimes().Return("CVE-2023-7008")
	flatCVE2023x7008.EXPECT().GetCVEIDs().AnyTimes().Return([]string{"CVE-2023-7008#0#systemd#0#4cd5259a-d1fc-5c81-ab1a-92484311441e"})
	flatCVE2023x7008.EXPECT().GetSeverity().AnyTimes().Return(moderateVuln)
	flatCVE2023x7008.EXPECT().GetTopCVSS().AnyTimes().Return(float32(5.5))
	flatCVE2023x7008.EXPECT().GetTopNVDCVSS().AnyTimes().Return(float32(5.4))
	flatCVE2023x7008.EXPECT().GetEPSSProbability().AnyTimes().Return(float32(0.45))
	flatCVE2023x7008.EXPECT().GetAffectedImageCount().AnyTimes().Return(1)
	flatCVE2023x7008.EXPECT().GetFirstDiscoveredInSystem().AnyTimes().Return(now)
	flatCVE2023x7008.EXPECT().GetPublishDate().AnyTimes().Return(now)
	flatCVE2023x7008.EXPECT().GetFirstImageOccurrence().AnyTimes().Return(now)
	flatCVE2023x7008.EXPECT().GetState().AnyTimes().Return(observedState)
	cve2023x7008 := &storage.ImageCVEV2{
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
	mockContainer.imageCVEDS.EXPECT().
		Search(gomock.Any(), gomock.Any()).
		AnyTimes().Return([]searchPkg.Result{{ID: cve2023x7008.GetId()}}, nil)
	mockContainer.imageCVEDS.EXPECT().
		SearchRawImageCVEs(gomock.Any(), gomock.Any()).
		AnyTimes().Return([]*storage.ImageCVEV2{cve2023x7008}, nil)
	mockContainer.imageCVEFlatView.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().Return([]imagecveflat.CveFlat{flatCVE2023x7008}, nil)
	mockContainer.imageCVEDS.EXPECT().
		GetBatch(gomock.Any(), gomock.Any()).
		AnyTimes().Return([]*storage.ImageCVEV2{cve2023x7008}, nil)

}

func validateResponseHasNoError(t testing.TB, response *graphql.Response) {
	t.Helper()

	if len(response.Errors) > 0 {
		for _, err := range response.Errors {
			t.Logf("GraphQL Error: %s", err.Error())
		}
	}
	assert.Empty(t, response.Errors, "Query should not produce errors")
}

func validateComponentIDResponse(t testing.TB, response *graphql.Response) []string {
	t.Helper()

	var findResult struct {
		Components []struct {
			ID string `json:"id"`
		} `json:"components"`
	}
	require.NoError(t, json.Unmarshal(response.Data, &findResult))
	require.NotEmpty(t, findResult.Components, "Should find systemd component")

	res := make([]string, 0, len(findResult.Components))
	for _, component := range findResult.Components {
		res = append(res, component.ID)
	}
	return res
}

func validateSingleVulnerabilityResponse(t testing.TB, response *graphql.Response) {
	t.Helper()

	var result testQueryResponse
	require.NoError(t, json.Unmarshal(response.Data, &result))

	require.Len(t, result.Result.Vulnerabilities, 1, "Should find exactly one fixable CVE-2023-7008")
}

func validateResponseContainsCVE2023x7008(t testing.TB, response *graphql.Response) {
	t.Helper()

	// Parse the response
	var result testQueryResponse
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
}

// endregion test helper functions

// region helper test types

type imageComponentCVEMocks struct {
	imageComponentDS       *imageComponentV2Mocks.MockDataStore
	imageComponentFlatView *imageComponentFlatViewMocks.MockComponentFlatView
	imageCVEDS             *imageCVEV2Mocks.MockDataStore
	imageCVEFlatView       *imageCVEFlatViewMocks.MockCveFlatView
}

// Response structure matching the GraphQL query
type testCVEResponse struct {
	CVE            string  `json:"cve"`
	CVSS           float64 `json:"cvss"`
	Severity       string  `json:"severity"`
	FixedByVersion string  `json:"fixedByVersion"`
}

type testImageComponentResponse struct {
	Vulnerabilities []testCVEResponse `json:"vulnerabilities"`
}

type testQueryResponse struct {
	Result testImageComponentResponse `json:"result"`
}

// endregion helper test types
