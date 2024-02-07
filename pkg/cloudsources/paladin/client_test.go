package paladin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetAssets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "testing", request.Header.Get("Authorization"))
		assert.NotEmpty(t, request.Header.Get("User-Agent"))

		data, err := os.ReadFile("testdata/response.json")
		require.NoError(t, err)

		_, err = writer.Write(data)
		require.NoError(t, err)
	}))
	defer server.Close()

	client := NewClient(&storage.CloudSource{
		Credentials: &storage.CloudSource_Credentials{Secret: "testing"},
		Config: &storage.CloudSource_PaladinCloud{PaladinCloud: &storage.PaladinCloudConfig{
			Endpoint: server.URL,
		}},
	})

	resp, err := client.GetAssets(context.Background())
	require.NoError(t, err)
	assert.Len(t, resp.Assets, 3)
}
