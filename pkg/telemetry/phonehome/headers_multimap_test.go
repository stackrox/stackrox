package phonehome

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestHeaders(t *testing.T) {
	h := make(http.Header)
	h.Add("key", "value 1")
	h.Add("key", "value 2")
	assert.Equal(t, []string{"value 1", "value 2"}, Headers(h).Get("key"))

	h = make(http.Header)
	Headers(h).Set("key", "value1", "value2")
	assert.Equal(t, "value1", h.Get("key"))
	assert.Equal(t, []string{"value1", "value2"}, Headers(h).Get("key"))
}

func TestKeyCase(t *testing.T) {
	const keyCase1 = "TEST-key"
	const keyCase2 = "test-KEY"
	const goodValue = "good"

	testKeys := func(t *testing.T, getter interface{ Get(string) []string }) {
		assert.Equal(t, []string{goodValue}, getter.Get(keyCase1))
		assert.Equal(t, []string{goodValue}, getter.Get(keyCase2))
	}

	t.Run("test metadata.MD", func(t *testing.T) {
		// keys are lowercased in metadata.MD.
		md := metadata.New(nil)
		md.Append(keyCase1, goodValue)
		testKeys(t, md)
	})

	t.Run("test http.Header", func(t *testing.T) {
		// keys are canonicalized in http.Header.
		h := make(http.Header)
		h.Add(keyCase1, goodValue)
		testKeys(t, Headers(h))
	})
}

func TestGetFirst(t *testing.T) {
	h := make(http.Header)
	h.Add("key", "value1")
	h.Add("key", "value2")
	assert.Equal(t, []string{"value1", "value2"}, h.Values("key"))

	assert.Equal(t, "value1", GetFirst(Headers(h).Get, "key"))
	assert.Equal(t, "", GetFirst(Headers(h).Get, "nokey"))
	assert.Equal(t, "", GetFirst(nil, "nokey"))
}
