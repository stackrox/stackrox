//go:build sql_integration

package handler

import (
	"bytes"
	"context"
	"encoding/json"
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
	_ = b
	testHelper := &testutils.ExportServicePostgresTestHelper{}
	err := testHelper.SetupTest(b)
	if err != nil {
		b.Error(err)
	}
	defer testHelper.TearDownTest(b)

	testHelper.InjectDataAndRunBenchmark(b, false, getDeploymentBenchmark(testHelper))
}

func getDeploymentBenchmark(testHelper *testutils.ExportServicePostgresTestHelper) func(b *testing.B) {
	return getGraphQLBenchmark(testHelper, deploymentsQuery)
}

func BenchmarkImageExport(b *testing.B) {
	_ = b
	testHelper := &testutils.ExportServicePostgresTestHelper{}
	err := testHelper.SetupTest(b)
	if err != nil {
		b.Error(err)
	}
	defer testHelper.TearDownTest(b)

	testHelper.InjectDataAndRunBenchmark(b, true, getImageBenchmark(testHelper))
}

func getImageBenchmark(testHelper *testutils.ExportServicePostgresTestHelper) func(b *testing.B) {
	return getGraphQLBenchmark(testHelper, imagesQuery)
}

func BenchmarkDeploymentWithImageExport(b *testing.B) {
	_ = b
	testHelper := &testutils.ExportServicePostgresTestHelper{}
	err := testHelper.SetupTest(b)
	if err != nil {
		b.Error(err)
	}
	defer testHelper.TearDownTest(b)

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
		return loaders.NewImageLoader(resolver.ImageDataStore)
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

func getGraphQLBenchmark(
	testHelper *testutils.ExportServicePostgresTestHelper,
	query string,
) func(b *testing.B) {
	return func(b *testing.B) {
		testCases := testutils.GetExportTestCases()
		for _, testCase := range testCases {
			b.Run(testCase.Name, func(ib *testing.B) {
				vals := map[string]interface{}{"query": query}
				if testCase.TargetNamespace != "" {
					query := "Namespace:" + testCase.TargetNamespace
					vals["variables"] = map[string]interface{}{
						"query": query,
					}
				}
				jsonBytes, err := json.Marshal(vals)
				if err != nil {
					ib.Error(err)
				}
				ib.ResetTimer()
				for i := 0; i < ib.N; i++ {
					ib.StopTimer()
					server, err := getGraphQLServer(testHelper)
					if err != nil {
						ib.Error(err)
					}
					ib.StartTimer()
					client := server.Client()
					requestData := bytes.NewBuffer(jsonBytes)
					req, reqErr := http.NewRequest(http.MethodPost, server.URL, requestData)
					if reqErr != nil {
						ib.Error(reqErr)
					}
					resp, callErr := client.Do(req)
					if callErr != nil {
						ib.Error(callErr)
					}
					_ = resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						ib.Error(resp.Status)
					}
				}
			})
		}
	}
}
