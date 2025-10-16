package docker

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

type fakeRegistry struct {
	NumFailedHeadCalls int
	NumAllowedCalls    int
}

func (f *fakeRegistry) HandlerFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodHead {
		http.Error(w, "", http.StatusMethodNotAllowed)
		f.NumFailedHeadCalls++
		return
	}

	f.NumAllowedCalls++
}

func TestMetadataFallback(t *testing.T) {
	tcs := []struct {
		desc           string
		featureEnabled bool
	}{
		{"attempt manifest digest, fallback on error", true},
		{"do not attempt manifest digest", false},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			testutils.MustUpdateFeature(t, features.AttemptManifestDigest, tc.featureEnabled)

			fr := &fakeRegistry{}

			s := httptest.NewServer(http.HandlerFunc(fr.HandlerFunc))
			defer s.Close()

			dc := &storage.DockerConfig{}
			dc.SetEndpoint(s.URL)
			ii := &storage.ImageIntegration{}
			ii.SetDocker(proto.ValueOrDefault(dc))
			r, err := NewRegistryWithoutManifestCall(ii, true, nil)
			require.NoError(t, err)

			imgStr := fmt.Sprintf("%s/%s:%s", urlfmt.TrimHTTPPrefixes(s.URL), "repo/path", "tag")
			cimg, err := utils.GenerateImageFromString(imgStr)
			require.NoError(t, err)
			img := types.ToImage(cimg)

			_, err = r.Metadata(img)

			// An overall error is expected because the 'fake registry' is returning garbage data.
			assert.Error(t, err)

			// Expect each of the handler functions to attempt a single call and fail.
			assert.Equal(t, len(manifestFuncs), fr.NumAllowedCalls)

			if tc.featureEnabled {
				assert.Equal(t, 1, fr.NumFailedHeadCalls)
			} else {
				assert.Equal(t, 0, fr.NumFailedHeadCalls)
			}

		})
	}
}
