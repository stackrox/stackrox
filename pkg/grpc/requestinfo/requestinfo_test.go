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

	hWithGet := WithGet(h)
	assert.Equal(t, h.Get("key"), hWithGet.Get("key")[0])
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

func TestHasGrpcPrefix(t *testing.T) {
	h := make(http.Header)
	h.Add("key", "value")
	assert.False(t, HasGrpcPrefix(WithGet(h)))
	h.Add(runtime.MetadataPrefix+"Accept", "value")
	assert.True(t, HasGrpcPrefix(WithGet(h)))
	assert.False(t, HasGrpcPrefix(nil))
}

func TestIgnoreGrcpPrefix(t *testing.T) {
	md := metadata.New(nil)
	md.Append(runtime.MetadataPrefix+"Accept", "value")
	md.Append("key", "value1")
	md.Append(runtime.MetadataPrefix+"key", "value2")
	assert.True(t, HasGrpcPrefix(md))

	noPrefix := IgnoreGrpcPrefix(md)
	assert.False(t, HasGrpcPrefix(noPrefix))
	assert.Equal(t, "value2", noPrefix.Get("key")[0])
}

func TestKeyCase(t *testing.T) {

	keyCase1 := "mixed-Case-key"
	keyCase2 := "Mixed-case-Key"
	goodValue := "good"

	testKeys := func(t *testing.T, getter HeaderGetter) {
		assert.Len(t, getter.Get(keyCase1), 1)
		assert.Equal(t, goodValue, GetFirst(getter, keyCase1))
		assert.Equal(t, goodValue, GetFirst(getter, keyCase2))
	}

	t.Run("test metadata.MD without prefix", func(t *testing.T) {
		// keys are lowercased in metadata.MD.
		md := metadata.New(nil)
		md.Append(keyCase1, goodValue)
		testKeys(t, md)
	})

	t.Run("test metadata.MD with prefix", func(t *testing.T) {
		// keys are lowercased in metadata.MD.
		md := metadata.New(nil)
		md.Append(runtime.MetadataPrefix+keyCase1, goodValue)
		testKeys(t, IgnoreGrpcPrefix(md))
	})

	t.Run("test http.Header", func(t *testing.T) {
		// keys are canonicalized in http.Header.
		h := make(http.Header)
		h.Add(keyCase1, goodValue)
		testKeys(t, WithGet(h))
	})
}
