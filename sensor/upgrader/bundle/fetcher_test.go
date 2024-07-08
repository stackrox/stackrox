package bundle

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/upgrader/config"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
	"github.com/stretchr/testify/assert"
)

const (
	expectedRequestBody = `{
	"id": "` + fixtureconsts.Cluster1 + `"
}`
)

func TestFetcherRequestEncoding(t *testing.T) {
	handler := getFakeZipHandler(t, expectedRequestBody)
	server := httptest.NewServer(handler)
	defer server.Close()

	fetcherCtx, err := upgradectx.CreateForTest(
		context.Background(),
		t,
		&config.UpgraderConfig{
			ClusterID:       fixtureconsts.Cluster1,
			CentralEndpoint: strings.TrimPrefix(server.URL, "http://"),
		},
	)
	assert.NoError(t, err)

	clusterFetcher := &fetcher{ctx: fetcherCtx}
	bundle, fetchErr := clusterFetcher.FetchBundle()
	assert.Nil(t, bundle)
	assert.Error(t, fetchErr)
	expectedErrorText := "making HTTP request to central for cluster bundle download: " +
		"received response code 404 Not Found, but expected 2xx; error message: " +
		"No data for requested cluster ID"
	assert.Equal(t, expectedErrorText, fetchErr.Error())
}

func getFakeZipHandler(t *testing.T, expectedRequestBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hasError := false
		hasError = !assert.Equal(t, http.MethodPost, r.Method) && hasError
		hasError = !assert.Equal(t, "/api/extensions/clusters/zip", r.URL.Path) && hasError
		reqBody := r.Body
		defer utils.IgnoreError(reqBody.Close)
		reqBodyData, err := io.ReadAll(reqBody)
		hasError = !assert.NoError(t, err) && hasError
		hasError = !assert.JSONEq(t, expectedRequestBody, string(reqBodyData)) && hasError

		if hasError {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("failed request verification"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("No data for requested cluster ID"))
		}
	}
}
