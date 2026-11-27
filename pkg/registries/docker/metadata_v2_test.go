package docker

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/opencontainers/go-digest"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	fakeDigestStr0 = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	fakeDigestStr1 = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
)

func TestImageReference(t *testing.T) {
	dig0, err := digest.Parse(fakeDigestStr0)
	require.NoError(t, err)

	tcs := []struct {
		ref  string
		dig  digest.Digest
		want string
	}{
		// expect ref if digest is empty
		{"latest", "", "latest"},
		{"", "", ""},

		// expect digest if ref is not a digest
		{"latest", dig0, fakeDigestStr0},

		// expect ref if it is a digest
		{fakeDigestStr1, dig0, fakeDigestStr1},
	}
	for i, tc := range tcs {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got := digestOrRef(tc.ref, tc.dig)
			assert.Equal(t, tc.want, got)
		})
	}
}

// dgst returns a syntactically valid sha256 digest from a single repeated char.
func dgst(c string) string { return "sha256:" + strings.Repeat(c, 64) }

func schema2ManifestJSON(configDigest string, layerDigests ...string) string {
	var layers strings.Builder
	for i, l := range layerDigests {
		if i > 0 {
			layers.WriteString(",")
		}
		fmt.Fprintf(&layers, `{"mediaType":%q,"size":10,"digest":%q}`,
			"application/vnd.docker.image.rootfs.diff.tar.gzip", l)
	}
	return fmt.Sprintf(`{"schemaVersion":2,"mediaType":%q,"config":{"mediaType":%q,"size":50,"digest":%q},"layers":[%s]}`,
		schema2.MediaTypeManifest,
		"application/vnd.docker.container.image.v1+json",
		configDigest,
		layers.String())
}

func configBlobJSON(os, arch string) string {
	return fmt.Sprintf(`{"architecture":%q,"os":%q,"created":"2024-01-01T00:00:00Z","config":{},"rootfs":{"type":"layers","diff_ids":[]},"history":[{"created":"2024-01-01T00:00:00Z","created_by":"/bin/sh -c #(nop) ADD file:abc in /"},{"created":"2024-01-01T00:00:00Z","created_by":"RUN echo hi"}]}`,
		arch, os)
}

// newTestRegistry wires a Registry to an httptest server. headDigest/headType are
// returned for all HEAD manifest requests; getBodies maps request paths to GET bodies.
func newTestRegistry(t *testing.T, headDigest, headType string, getBodies map[string]string) *Registry {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodHead {
			w.Header().Set("Docker-Content-Digest", headDigest)
			w.Header().Set("Content-Type", headType)
			w.WriteHeader(http.StatusOK)
			return
		}
		body, ok := getBodies[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(s.Close)

	r, err := NewDockerRegistryWithConfig(&Config{Endpoint: s.URL, DisableRepoList: true}, &storage.ImageIntegration{})
	require.NoError(t, err)
	return r
}

func imageForTest(t *testing.T, reg *Registry, tag string) *storage.Image {
	return &storage.Image{
		Name: &storage.ImageName{
			Registry: urlfmt.TrimHTTPPrefixes(reg.url),
			Remote:   "test/repo",
			Tag:      tag,
		},
	}
}

func TestMetadataForAllPlatforms(t *testing.T) {
	listDigest := dgst("a")
	amdManifest, armManifest, winManifest := dgst("b"), dgst("c"), dgst("f")
	armV6Manifest, armV7Manifest := dgst("6"), dgst("7")
	amdConfig, armConfig := dgst("d"), dgst("e")
	armV6Config, armV7Config := dgst("8"), dgst("9")
	amdLayers := []string{dgst("1"), dgst("2")}
	armLayers := []string{dgst("3"), dgst("4")}
	armV6Layers := []string{dgst("5")}
	armV7Layers := []string{dgst("0")}

	// platformKey mirrors the fact that os/architecture alone is not unique: arm
	// variants share os/arch and must be distinguished by variant.
	platformKey := func(pm *types.PlatformMetadata) string {
		return pm.OS + "/" + pm.Architecture + "/" + pm.Variant
	}

	t.Run("manifest list returns one entry per linux platform, distinguishes variants, skips non-linux", func(t *testing.T) {
		list := manifestListJSONWithDigests(
			platformEntry{amdManifest, "linux", "amd64", ""},
			platformEntry{armManifest, "linux", "arm64", ""},
			platformEntry{armV6Manifest, "linux", "arm", "v6"},
			platformEntry{armV7Manifest, "linux", "arm", "v7"},
			platformEntry{winManifest, "windows", "amd64", ""},
		)

		reg := newTestRegistry(t, listDigest, manifestlist.MediaTypeManifestList, map[string]string{
			"/v2/test/repo/manifests/" + listDigest:    list,
			"/v2/test/repo/manifests/" + amdManifest:   schema2ManifestJSON(amdConfig, amdLayers...),
			"/v2/test/repo/manifests/" + armManifest:   schema2ManifestJSON(armConfig, armLayers...),
			"/v2/test/repo/manifests/" + armV6Manifest: schema2ManifestJSON(armV6Config, armV6Layers...),
			"/v2/test/repo/manifests/" + armV7Manifest: schema2ManifestJSON(armV7Config, armV7Layers...),
			"/v2/test/repo/blobs/" + amdConfig:         configBlobJSON("linux", "amd64"),
			"/v2/test/repo/blobs/" + armConfig:         configBlobJSON("linux", "arm64"),
			"/v2/test/repo/blobs/" + armV6Config:       configBlobJSON("linux", "arm"),
			"/v2/test/repo/blobs/" + armV7Config:       configBlobJSON("linux", "arm"),
		})

		result, err := reg.MetadataForAllPlatforms(imageForTest(t, reg, "latest"))
		require.NoError(t, err)
		assert.Equal(t, listDigest, result.ManifestDigest)
		assert.True(t, result.IsManifestList)
		require.Len(t, result.Platforms, 4, "windows platform should be skipped, all linux platforms kept")

		byPlatform := map[string]*types.PlatformMetadata{}
		for _, pm := range result.Platforms {
			byPlatform[platformKey(pm)] = pm
		}
		require.Len(t, byPlatform, 4, "arm v6 and v7 must not collide")

		amd := byPlatform["linux/amd64/"]
		require.NotNil(t, amd)
		assert.Equal(t, amdLayers, amd.Metadata.GetLayerShas())
		assert.NotNil(t, amd.Metadata.GetV1().GetCreated())
		assert.Len(t, amd.Metadata.GetV1().GetLayers(), 2)

		armV6 := byPlatform["linux/arm/v6"]
		require.NotNil(t, armV6)
		assert.Equal(t, armV6Layers, armV6.Metadata.GetLayerShas())

		armV7 := byPlatform["linux/arm/v7"]
		require.NotNil(t, armV7)
		assert.Equal(t, armV7Layers, armV7.Metadata.GetLayerShas())

		assert.NotEqual(t, armV6.Metadata.GetLayerShas(), armV7.Metadata.GetLayerShas(),
			"same os/arch, different variant must yield distinct layers")
	})

	t.Run("single manifest returns one entry with os/arch from config", func(t *testing.T) {
		reg := newTestRegistry(t, amdManifest, schema2.MediaTypeManifest, map[string]string{
			"/v2/test/repo/manifests/" + amdManifest: schema2ManifestJSON(amdConfig, amdLayers...),
			"/v2/test/repo/blobs/" + amdConfig:       configBlobJSON("linux", "amd64"),
		})

		result, err := reg.MetadataForAllPlatforms(imageForTest(t, reg, "latest"))
		require.NoError(t, err)
		assert.Equal(t, amdManifest, result.ManifestDigest)
		assert.False(t, result.IsManifestList)
		require.Len(t, result.Platforms, 1)
		assert.Equal(t, "linux", result.Platforms[0].OS)
		assert.Equal(t, "amd64", result.Platforms[0].Architecture)
		assert.Equal(t, amdLayers, result.Platforms[0].Metadata.GetLayerShas())
	})

	t.Run("manifest list with no linux platforms errors", func(t *testing.T) {
		list := manifestListJSONWithDigests(platformEntry{winManifest, "windows", "amd64", ""})
		reg := newTestRegistry(t, listDigest, manifestlist.MediaTypeManifestList, map[string]string{
			"/v2/test/repo/manifests/" + listDigest: list,
		})

		result, err := reg.MetadataForAllPlatforms(imageForTest(t, reg, "latest"))
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("nil image returns nil", func(t *testing.T) {
		reg := newTestRegistry(t, listDigest, manifestlist.MediaTypeManifestList, nil)
		result, err := reg.MetadataForAllPlatforms(nil)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

type platformEntry struct {
	digest, os, arch, variant string
}

func manifestListJSONWithDigests(entries ...platformEntry) string {
	var mfs strings.Builder
	for i, e := range entries {
		if i > 0 {
			mfs.WriteString(",")
		}
		fmt.Fprintf(&mfs, `{"mediaType":%q,"size":100,"digest":%q,"platform":{"architecture":%q,"os":%q,"variant":%q}}`,
			schema2.MediaTypeManifest, e.digest, e.arch, e.os, e.variant)
	}
	return fmt.Sprintf(`{"schemaVersion":2,"mediaType":%q,"manifests":[%s]}`,
		manifestlist.MediaTypeManifestList, mfs.String())
}
