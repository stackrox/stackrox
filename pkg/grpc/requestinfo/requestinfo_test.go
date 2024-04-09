package requestinfo

import (
	"net/http"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestWithGet(t *testing.T) {
	h := make(http.Header)
	h.Add("key", "value")

	assert.Equal(t, h.Get("key"), WithGet(h).Get("key")[0])
}

func TestGetFirst(t *testing.T) {
	h := make(http.Header)
	h.Add("key", "value1")
	h.Add("key", "value2")
	assert.Equal(t, []string{"value1", "value2"}, h.Values("key"))

	assert.Equal(t, "value1", GetFirst(WithGet(h), "key"))
	assert.Equal(t, "", GetFirst(WithGet(h), "nokey"))
	assert.Equal(t, "", GetFirst(nil, "nokey"))
}

func TestIgnoreMetadataPrefix(t *testing.T) {
	md := metadata.New(nil)
	md.Append("Accept", "value1")
	md.Append("custom-key", "value1")
	md.Append(runtime.MetadataPrefix+"Accept", "value2")
	md.Append(runtime.MetadataPrefix+"custom-key", "value2")

	noPrefix := WithHeaderMatcher(md)
	assert.Equal(t, "value2", noPrefix.Get("Accept")[0])
	assert.Equal(t, "value1", noPrefix.Get("custom-key")[0])
}

func TestKeyCase(t *testing.T) {

	// Use "permanent" headers to enable header matcher.
	const keyCase1 = "content-Type"
	const keyCase2 = "Content-type"
	const goodValue = "good"

	testKeys := func(t *testing.T, getter HeaderGetter) {
		assert.Len(t, getter.Get(keyCase1), 1)
		assert.Len(t, getter.Get(keyCase2), 1)
		assert.Equal(t, goodValue, GetFirst(getter, keyCase1))
		assert.Equal(t, goodValue, GetFirst(getter, keyCase2))
	}

	t.Run("test metadata.MD without prefix", func(t *testing.T) {
		// keys are lowercased in metadata.MD.
		md := metadata.New(nil)
		md.Append(keyCase1, goodValue)
		testKeys(t, md)
		assert.Empty(t, WithHeaderMatcher(md).Get(keyCase1))
		assert.Empty(t, WithHeaderMatcher(md).Get(keyCase2))
	})

	t.Run("test metadata.MD with prefix", func(t *testing.T) {
		// keys are lowercased in metadata.MD.
		md := metadata.New(nil)
		md.Append(runtime.MetadataPrefix+keyCase1, goodValue)
		testKeys(t, WithHeaderMatcher(md))
		assert.Empty(t, md.Get(keyCase1))
		assert.Empty(t, md.Get(keyCase2))
	})

	t.Run("test http.Header", func(t *testing.T) {
		// keys are canonicalized in http.Header.
		h := make(http.Header)
		h.Add(keyCase1, goodValue)
		testKeys(t, WithGet(h))
		assert.Empty(t, WithHeaderMatcher(WithGet(h)).Get(keyCase1))
		assert.Empty(t, WithHeaderMatcher(WithGet(h)).Get(keyCase2))
	})
}
