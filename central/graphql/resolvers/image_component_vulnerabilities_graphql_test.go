//go:build sql_integration

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
	deploymentsView "github.com/stackrox/rox/central/views/deployments"
	"github.com/stackrox/rox/central/views/imagecomponentflat"
	imageComponentFlatViewMocks "github.com/stackrox/rox/central/views/imagecomponentflat/mocks"
	"github.com/stackrox/rox/central/views/imagecveflat"
	imageCVEFlatViewMocks "github.com/stackrox/rox/central/views/imagecveflat/mocks"
	imagesView "github.com/stackrox/rox/central/views/images"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

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

func TestGetFixableCVEsForEntityGraphQL(t *testing.T) {
	suite.Run(t, new(ImageComponentCVEGraphQLTestSuite))
}

type ImageComponentCVEGraphQLTestSuite struct {
	suite.Suite

	ctx      context.Context
	testDB   *pgtest.TestPostgres
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
	s.testDB = pgtest.ForT(s.T())

	// TODO(ROX-30117): Remove conditional when FlattenImageData feature flag is removed.
	var resolver *Resolver
	s.imageComponentDS = imageComponentV2Mocks.NewMockDataStore(s.mockCtrl)
	s.imageComponentFlatView = imageComponentFlatViewMocks.NewMockComponentFlatView(s.mockCtrl)
	s.imageCVEDS = imageCVEV2Mocks.NewMockDataStore(s.mockCtrl)
	s.imageCVEFlatView = imageCVEFlatViewMocks.NewMockCveFlatView(s.mockCtrl)
	if features.FlattenImageData.Enabled() {
		imgV2DataStore := CreateTestImageV2Datastore(s.T(), s.testDB, s.mockCtrl)
		resolver, _ = SetupTestResolver(s.T(),
			imagesView.NewImageView(s.testDB.DB),
			imgV2DataStore,
			// s.imageComponentDS,
			CreateTestImageComponentV2Datastore(s.T(), s.testDB, s.mockCtrl),
			// s.imageCVEDS,
			CreateTestImageCVEV2Datastore(s.T(), s.testDB),
			CreateTestDeploymentDatastoreWithImageV2(s.T(), s.testDB, s.mockCtrl, imgV2DataStore),
			deploymentsView.NewDeploymentView(s.testDB.DB),
			// s.imageCVEFlatView,
			imagecveflat.NewCVEFlatView(s.testDB.DB),
			// s.imageComponentFlatView,
			imagecomponentflat.NewComponentFlatView(s.testDB.DB),
		)
	} else {
		imageDataStore := CreateTestImageDatastore(s.T(), s.testDB, s.mockCtrl)
		resolver, _ = SetupTestResolver(s.T(),
			imagesView.NewImageView(s.testDB.DB),
			imageDataStore,
			// s.imageComponentDS,
			CreateTestImageComponentV2Datastore(s.T(), s.testDB, s.mockCtrl),
			// s.imageCVEDS,
			CreateTestImageCVEV2Datastore(s.T(), s.testDB),
			CreateTestDeploymentDatastore(s.T(), s.testDB, s.mockCtrl, imageDataStore),
			deploymentsView.NewDeploymentView(s.testDB.DB),
			// s.imageCVEFlatView,
			imagecveflat.NewCVEFlatView(s.testDB.DB),
			// s.imageComponentFlatView,
			imagecomponentflat.NewComponentFlatView(s.testDB.DB),
		)
	}
	s.resolver = resolver

	// Parse the GraphQL schema
	var err error
	s.schema, err = graphql.ParseSchema(Schema(), s.resolver)
	s.Require().NoError(err)

	// Create test image with systemd component and CVE-2023-7008
	testImage := s.createUbuntuImageWithSystemd()

	// TODO(ROX-30117): Remove conditional when FlattenImageData feature flag is removed.
	if features.FlattenImageData.Enabled() {
		err := s.resolver.ImageV2DataStore.UpsertImage(s.ctx, imageUtils.ConvertToV2(testImage))
		s.NoError(err)
	} else {
		err := s.resolver.ImageDataStore.UpsertImage(s.ctx, testImage)
		s.NoError(err)
	}
}

func (s *ImageComponentCVEGraphQLTestSuite) createUbuntuImageWithSystemd() *storage.Image {
	scanTime, err := protocompat.ConvertTimeToTimestampOrError(time.Now())
	utils.CrashOnError(err)

	return &storage.Image{
		Id: "sha256:ubuntu-22.04-amd64",
		Name: &storage.ImageName{
			Registry: "quay.io",
			Remote:   "rhacs-eng/qa",
			Tag:      "ubuntu-22.04-amd64",
			FullName: "quay.io/rhacs-eng/qa:ubuntu-22.04-amd64",
		},
		SetCves: &storage.Image_Cves{
			Cves: 1,
		},
		Scan: &storage.ImageScan{
			ScanTime: scanTime,
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "systemd",
					Version: "249.11-0ubuntu3.11",
					Source:  storage.SourceType_OS,
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve: "CVE-2023-7008",
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "249.11-0ubuntu3.12",
							},
							Cvss:     5.5,
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						},
					},
				},
			},
			OperatingSystem: "ubuntu:22.04",
		},
	}
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

	/*
		mockComponentView := imageComponentFlatViewMocks.NewMockComponentFlat(s.mockCtrl)
		mockComponentView.EXPECT().GetComponentIDs().AnyTimes().Return([]string{"4ed5259a-d1fc-5c81-ab1a-92484311441e"})
		mockComponentView.EXPECT().GetComponent().AnyTimes().Return("systemd")
		mockComponentView.EXPECT().GetVersion().AnyTimes().Return("249.11-0ubuntu3.11")
		mockComponentView.EXPECT().GetOperatingSystem().AnyTimes().Return("ubuntu:22.04")
		s.imageComponentFlatView.EXPECT().
			Get(gomock.Any(), gomock.Any()).
			Times(1).
			Return([]imageComponentFlatView.ComponentFlat{mockComponentView}, nil)
		testComponent := &storage.ImageComponentV2{
			Id:              "systemd#0#4ed5259a-d1fc-5c81-ab1a-92484311441e",
			Name:            "systemd",
			Version:         "249.11-0ubuntu3.11",
			OperatingSystem: "ubuntu:22.04",
		}
		s.imageComponentDS.EXPECT().
			SearchRawImageComponents(gomock.Any(), gomock.Any()).
			Times(1).
			Return([]*storage.ImageComponentV2{testComponent}, nil)
	*/
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
