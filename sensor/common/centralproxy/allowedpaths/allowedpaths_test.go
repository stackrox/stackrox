package allowedpaths

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAllowed_NoPaths(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	assert.True(t, IsAllowed("/v1/alerts"), "all paths should be allowed when no paths are configured")
	assert.True(t, IsAllowed("/anything"), "all paths should be allowed when no paths are configured")
	assert.True(t, IsAllowed("/"), "all paths should be allowed when no paths are configured")
}

func TestIsAllowed_EmptySlice(t *testing.T) {
	Set([]string{})
	t.Cleanup(Reset)

	assert.True(t, IsAllowed("/v1/alerts"), "all paths should be allowed when empty slice is set")
	assert.True(t, IsAllowed("/anything"), "all paths should be allowed when empty slice is set")
}

func TestIsAllowed_NilSlice(t *testing.T) {
	Set(nil)
	t.Cleanup(Reset)

	assert.True(t, IsAllowed("/v1/alerts"), "all paths should be allowed when nil slice is set")
	assert.True(t, IsAllowed("/anything"), "all paths should be allowed when nil slice is set")
}

func TestIsAllowed_PrefixMatch(t *testing.T) {
	Set([]string{"/v1/", "/v2/", "/api/v1/"})
	t.Cleanup(Reset)

	assert.True(t, IsAllowed("/v1/alerts"), "trailing-slash entry should prefix-match")
	assert.True(t, IsAllowed("/v1/deployments"))
	assert.True(t, IsAllowed("/v2/something"))
	assert.True(t, IsAllowed("/api/v1/foo"))
}

func TestIsAllowed_ExactMatch(t *testing.T) {
	Set([]string{"/api/graphql"})
	t.Cleanup(Reset)

	assert.True(t, IsAllowed("/api/graphql"), "exact path should match")
	assert.False(t, IsAllowed("/api/graphql?query=foo"), "no-slash entry must not prefix-match")
	assert.False(t, IsAllowed("/api/graphql/sub"), "no-slash entry must not prefix-match subpath")
}

func TestIsAllowed_NonMatchingPaths(t *testing.T) {
	Set([]string{"/v1/", "/v2/", "/api/graphql", "/api/v1/"})
	t.Cleanup(Reset)

	assert.False(t, IsAllowed("/admin/secret"))
	assert.False(t, IsAllowed("/internal/debug"))
	assert.False(t, IsAllowed("/"))
	assert.False(t, IsAllowed("/v3/something"))
	assert.False(t, IsAllowed("/api/v2/foo"))
	assert.False(t, IsAllowed("/api/graphql/extra"), "no-slash entry must not prefix-match")
}

func TestReset(t *testing.T) {
	Set([]string{"/v1/"})
	assert.True(t, IsAllowed("/v1/alerts"))
	assert.False(t, IsAllowed("/admin/secret"))

	Reset()
	assert.True(t, IsAllowed("/admin/secret"), "after reset all paths should be allowed")
}
