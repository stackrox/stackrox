package resolvers

import (
	"context"
	"encoding/json"
	"testing"

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
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// This test was created to investigate the failures that occurred
// during the upgrade of graphql-go from 1.5.0 to the next version
// (eventually carried to 1.10.2).

func TestGraphQLQuery(t *testing.T) {
	testCtx := loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	var resolver *Resolver

	imageComponentDS := imageComponentV2Mocks.NewMockDataStore(mockCtrl)
	imageComponentFlatView := imageComponentFlatViewMocks.NewMockComponentFlatView(mockCtrl)
	imageCVEDS := imageCVEV2Mocks.NewMockDataStore(mockCtrl)
	imageCVEFlatView := imageCVEFlatViewMocks.NewMockCveFlatView(mockCtrl)

	resolver, _ = SetupTestResolver(t,
		imageComponentDS,
		imageCVEDS,
		imageCVEFlatView,
		imageComponentFlatView,
	)

	schema, err := graphql.ParseSchema(Schema(), resolver)
	require.NoError(t, err)

	ctx := SetAuthorizerOverride(testCtx, allow.Anonymous())

	componentID := "systemd#0#4cd5259a-d1fc-5c81-ab1a-92484311441e"

	imageComponentFlatView.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return([]imagecomponentflat.ComponentFlat{systemdFlatComponent}, nil)
	imageComponentDS.EXPECT().
		SearchRawImageComponents(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return([]*storage.ImageComponentV2{systemdStorageComponent}, nil)

	imageComponentDS.EXPECT().GetBatch(gomock.Any(), gomock.Any()).AnyTimes().Return([]*storage.ImageComponentV2{systemdStorageComponent}, nil)
	imageCVEDS.EXPECT().Search(gomock.Any(), gomock.Any()).AnyTimes().Return([]searchPkg.Result{{ID: cve2023x7008.GetId()}}, nil)
	imageCVEDS.EXPECT().GetBatch(gomock.Any(), gomock.Any()).AnyTimes().Return([]*storage.ImageCVEV2{cve2023x7008}, nil)
	imageCVEDS.EXPECT().SearchRawImageCVEs(gomock.Any(), gomock.Any()).AnyTimes().Return([]*storage.ImageCVEV2{cve2023x7008}, nil)
	imageCVEFlatView.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]imagecveflat.CveFlat{flatCVE2023x7008}, nil)

	gqlID := graphql.ID(componentID)
	imageComponentResolver, err := resolver.ImageComponent(ctx, struct{ ID *graphql.ID }{ID: &gqlID})
	assert.NoError(t, err)
	assert.NotNil(t, imageComponentResolver)

	dumpResolverMethods(t, imageComponentResolver, "imageComponentV2Resolver")

	response := schema.Exec(ctx, getFixableCVEsForEntityQuery, "getFixableCvesForEntity",
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
}
