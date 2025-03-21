//go:build sql_integration

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	deploymentsQuery = `query listDeployments($query:String){
	deployments(query: $query) {
		id
		name
		type
		namespace
		namespaceId
		stateTimestamp
	}
}`

	imagesQuery = `query listImages($query:String){
	images(query: $query) {
		id
		name {
			registry
			remote
			tag
			fullName
		}
		# scan is required to ensure the full image is pulled
		# and not just the associated metadata.
		scan {
			operatingSystem
			scanTime
		}
	}
}`

	deploymentsWithImagesQuery = `query listDeploymentsWithImages($query: String) {
	deployments(query: $query) {
		id
		name
		type
		namespace
		namespaceId
		stateTimestamp
	}
	# For the purpose of the test, the plan is to pull the deployments
	# and images filtered by namespace. The query resolution for images
	# relies on the relation to the deployment table, so the behaviour
	# should be similar to the output of "/v1/export/vuln-mgmt/workloads/"
	images(query: $query) {
		id
		name {
			registry
			remote
			tag
			fullName
		}
		# scan is required to ensure the full image is pulled
		# and not just the associated metadata.
		scan {
			operatingSystem
			scanTime
		}
	}
}`
)

func BenchmarkDeploymentExport(b *testing.B) {
	testHelper := &testutils.ExportServicePostgresTestHelper{}
	err := testHelper.SetupTest(b)
	require.NoError(b, err)

	testHelper.InjectDataAndRunBenchmark(b, false, getDeploymentBenchmark(testHelper))
}

func getDeploymentBenchmark(testHelper *testutils.ExportServicePostgresTestHelper) func(b *testing.B) {
	return getGraphQLBenchmark(testHelper, deploymentsQuery)
}

func BenchmarkImageExport(b *testing.B) {
	testHelper := &testutils.ExportServicePostgresTestHelper{}
	err := testHelper.SetupTest(b)
	require.NoError(b, err)

	testHelper.InjectDataAndRunBenchmark(b, true, getImageBenchmark(testHelper))
}

func getImageBenchmark(testHelper *testutils.ExportServicePostgresTestHelper) func(b *testing.B) {
	return getGraphQLBenchmark(testHelper, imagesQuery)
}

func BenchmarkDeploymentWithImageExport(b *testing.B) {
	testHelper := &testutils.ExportServicePostgresTestHelper{}
	err := testHelper.SetupTest(b)
	require.NoError(b, err)

	testHelper.InjectDataAndRunBenchmark(b, true, getDeploymentWithImageBenchmark(testHelper))
}

func getDeploymentWithImageBenchmark(testHelper *testutils.ExportServicePostgresTestHelper) func(b *testing.B) {
	return getGraphQLBenchmark(testHelper, deploymentsWithImagesQuery)
}

type wrappedHandler struct {
	ctx     context.Context
	handler http.Handler
}

func (h *wrappedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wrappedRequest := r.WithContext(h.ctx)
	h.handler.ServeHTTP(w, wrappedRequest)
}

func getRequestJSON(
	query string,
	targetNamespace string,
) ([]byte, error) {
	vals := map[string]interface{}{"query": query}
	if targetNamespace != "" {
		query := "Namespace:" + targetNamespace
		vals["variables"] = map[string]interface{}{
			"query": query,
		}
	}
	jsonBytes, err := json.Marshal(vals)
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}

func getGraphQLServer(testHelper *testutils.ExportServicePostgresTestHelper) (*httptest.Server, error) {
	schema := resolvers.Schema()
	resolver := resolvers.NewMock()
	resolver.DeploymentDataStore = testHelper.Deployments
	resolver.ImageDataStore = testHelper.Images
	// Override Deployment and Image loader to avoid panics
	deploymentFactory := func() interface{} {
		return loaders.NewDeploymentLoader(resolver.DeploymentDataStore)
	}
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.Deployment{}), deploymentFactory)
	imageFactory := func() interface{} {
		return loaders.NewImageLoader(resolver.ImageDataStore, testHelper.ImageView)
	}
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.Image{}), imageFactory)
	ourSchema, err := graphql.ParseSchema(schema, resolver)
	if err != nil {
		return nil, err
	}
	handler := &relayHandler{Schema: ourSchema}
	ourHandler := &wrappedHandler{
		ctx: sac.WithAllAccess(
			loaders.WithLoaderContext(
				resolvers.SetAuthorizerOverride(
					context.Background(),
					allow.Anonymous(),
				),
			),
		),
		handler: handler,
	}
	server := httptest.NewServer(ourHandler)
	return server, nil
}

func prepareGraphQLCall(
	testHelper *testutils.ExportServicePostgresTestHelper,
	jsonBytes []byte,
) (*http.Client, *http.Request, error) {
	server, err := getGraphQLServer(testHelper)
	if err != nil {
		return nil, nil, err
	}
	client := server.Client()
	requestData := bytes.NewBuffer(jsonBytes)
	req, reqErr := http.NewRequest(http.MethodPost, server.URL, requestData)
	if reqErr != nil {
		return nil, nil, err
	}
	return client, req, nil
}

func consumeResponse(b *testing.B, resp *http.Response) {
	defer func() { assert.NoError(b, resp.Body.Close()) }()
	assert.Equal(b, http.StatusOK, resp.StatusCode)
	_, err := io.ReadAll(resp.Body)
	assert.NoError(b, err)
}

func getGraphQLBenchmark(
	testHelper *testutils.ExportServicePostgresTestHelper,
	query string,
) func(b *testing.B) {
	return func(b *testing.B) {
		testCases := testutils.GetExportTestCases()
		for _, testCase := range testCases {
			b.Run(testCase.Name, func(ib *testing.B) {
				jsonBytes, err := getRequestJSON(query, testCase.TargetNamespace)
				require.NoError(ib, err)
				ib.ResetTimer()
				for i := 0; i < ib.N; i++ {
					ib.StopTimer()
					// A new server is needed for each loop run, otherwise
					// the GraphQL loader will cache the objects loaded
					// during the first loop run, which is not the behaviour
					// we're trying to test.
					client, req, err := prepareGraphQLCall(testHelper, jsonBytes)
					require.NoError(ib, err)
					ib.StartTimer()
					resp, callErr := client.Do(req)
					assert.NoError(ib, callErr)
					consumeResponse(ib, resp)
				}
			})
		}
	}
}
