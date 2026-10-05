package olm

import (
	"context"
	"io"
	"testing"

	"github.com/stackrox/rox/pkg/operatorbundle"
	api "github.com/stackrox/rox/tools/operatorbundle/olm/registryapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

// --- helpers ------------------------------------------------------------------

func csvObject(namespace, name, properties string, labels map[string]interface{}, relatedImages []interface{}) *unstructured.Unstructured {
	meta := map[string]interface{}{
		"name":      name,
		"namespace": namespace,
	}
	if properties != "" {
		meta["annotations"] = map[string]interface{}{propertiesAnnotation: properties}
	}
	if labels != nil {
		meta["labels"] = labels
	}
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "operators.coreos.com/v1alpha1",
		"kind":       "ClusterServiceVersion",
		"metadata":   meta,
		"spec": map[string]interface{}{
			"displayName":   "My Package",
			"version":       "1.2.3",
			"relatedImages": relatedImages,
		},
	}}
}

func subscriptionObject(namespace, name, pkg, channel string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "operators.coreos.com/v1alpha1",
		"kind":       "Subscription",
		"metadata":   map[string]interface{}{"name": name, "namespace": namespace},
		"spec":       map[string]interface{}{"name": pkg, "channel": channel},
	}}
}

func relatedImage(name, image string) interface{} {
	return map[string]interface{}{"name": name, "image": image}
}

const olmPackageProps = `{"properties":[{"type":"olm.package","value":{"packageName":"mypkg","version":"1.2.3"}}]}`

func newFakeDynamic(objs ...runtime.Object) *dynamicfake.FakeDynamicClient {
	listKinds := map[schema.GroupVersionResource]string{
		csvGVR: "ClusterServiceVersionList",
		subGVR: "SubscriptionList",
	}
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds, objs...)
}

// --- FindInstalledBundle ------------------------------------------------------

func TestFindInstalledBundle(t *testing.T) {
	ctx := context.Background()
	realCSV := csvObject("operators", "mypkg.v1.2.3", olmPackageProps, nil, []interface{}{
		relatedImage("comp-a", "registry.example.com/a@sha256:aaa"),
		relatedImage("comp-b", "registry.example.com/b@sha256:bbb"),
	})
	// An OLM copy of the same CSV in another namespace must be ignored.
	copiedCSV := csvObject("other", "mypkg.v1.2.3", olmPackageProps,
		map[string]interface{}{"olm.copiedFrom": "operators"}, []interface{}{
			relatedImage("comp-a", "registry.example.com/a@sha256:aaa"),
		})
	sub := subscriptionObject("operators", "mypkg-sub", "mypkg", "stable")

	t.Run("resolves digest to installed bundle", func(t *testing.T) {
		c := NewClient(newFakeDynamic(realCSV, copiedCSV, sub), nil)
		b, err := c.FindInstalledBundle(ctx, "sha256:aaa")
		require.NoError(t, err)
		require.NotNil(t, b)
		assert.Equal(t, "mypkg", b.Package)
		assert.Equal(t, "1.2.3", b.Version)
		assert.Equal(t, "stable", b.ChannelName)
		assert.Equal(t, "mypkg.v1.2.3", b.CSVName)
		assert.Equal(t, "My Package", b.CSVDisplayName)
		require.Len(t, b.RelatedImages, 2)
		assert.Equal(t, "comp-a", b.RelatedImages[0].Name)
		assert.Equal(t, "sha256:aaa", b.RelatedImages[0].Digest)
	})

	t.Run("unknown digest returns nil", func(t *testing.T) {
		c := NewClient(newFakeDynamic(realCSV, sub), nil)
		b, err := c.FindInstalledBundle(ctx, "sha256:zzz")
		require.NoError(t, err)
		assert.Nil(t, b)
	})

	t.Run("package falls back to name when property missing", func(t *testing.T) {
		noProps := csvObject("operators", "fallbackpkg.v4.5.6", "", nil, []interface{}{
			relatedImage("x", "registry.example.com/x@sha256:xxx"),
		})
		c := NewClient(newFakeDynamic(noProps), nil)
		b, err := c.FindInstalledBundle(ctx, "sha256:xxx")
		require.NoError(t, err)
		require.NotNil(t, b)
		assert.Equal(t, "fallbackpkg", b.Package)
		assert.Equal(t, "1.2.3", b.Version) // from spec.version
	})
}

// --- FindCandidateBundles -----------------------------------------------------

type fakeListStream struct {
	grpc.ClientStream
	bundles []*api.Bundle
	i       int
}

func (f *fakeListStream) Recv() (*api.Bundle, error) {
	if f.i >= len(f.bundles) {
		return nil, io.EOF
	}
	b := f.bundles[f.i]
	f.i++
	return b, nil
}

type fakeRegistry struct {
	api.RegistryClient // embedded nil; only ListBundles is exercised
	bundles            []*api.Bundle
}

func (f *fakeRegistry) ListBundles(_ context.Context, _ *api.ListBundlesRequest, _ ...grpc.CallOption) (api.Registry_ListBundlesClient, error) {
	return &fakeListStream{bundles: f.bundles}, nil
}

func csvJSONWithImages(bundlePath string, named map[string]string) string {
	// Build a CSV doc including a self-reference (image == bundlePath, empty name).
	related := `{"name":"","image":"` + bundlePath + `"}`
	for name, image := range named {
		related += `,{"name":"` + name + `","image":"` + image + `"}`
	}
	return `{"spec":{"displayName":"My Package","version":"9.9.9","relatedImages":[` + related + `]}}`
}

func TestFindCandidateBundles(t *testing.T) {
	ctx := context.Background()
	const bundlePath = "registry.example.com/bundle@sha256:self"

	v123 := &api.Bundle{
		PackageName: "mypkg", ChannelName: "stable", CsvName: "mypkg.v1.2.3", Version: "1.2.3",
		BundlePath: bundlePath,
		CsvJson:    csvJSONWithImages(bundlePath, map[string]string{"comp-a": "registry.example.com/a@sha256:aaa"}),
	}
	// Same bundle re-listed under another channel: must be deduped by CsvName.
	v123dup := &api.Bundle{
		PackageName: "mypkg", ChannelName: "stable-1.2", CsvName: "mypkg.v1.2.3", Version: "1.2.3",
		BundlePath: bundlePath, CsvJson: v123.CsvJson,
	}
	// Version empty on the proto -> fall back to csvJson spec.version ("9.9.9").
	v999 := &api.Bundle{
		PackageName: "mypkg", ChannelName: "stable", CsvName: "mypkg.v9.9.9", Version: "",
		BundlePath: bundlePath,
		CsvJson:    csvJSONWithImages(bundlePath, map[string]string{"comp-a": "registry.example.com/a@sha256:ccc"}),
	}
	// Different package: must be filtered out.
	other := &api.Bundle{PackageName: "otherpkg", ChannelName: "stable", CsvName: "otherpkg.v1.0.0", Version: "1.0.0"}

	reg := &fakeRegistry{bundles: []*api.Bundle{v123, v123dup, v999, other}}
	c := NewClient(nil, reg)

	bundles, err := c.FindCandidateBundles(ctx, "mypkg", "", "")
	require.NoError(t, err)
	require.Len(t, bundles, 2, "dedupe by CsvName and filter other packages")

	byCSV := make(map[string]operatorbundle.Bundle, len(bundles))
	for _, b := range bundles {
		byCSV[b.CSVName] = b
	}

	got123 := byCSV["mypkg.v1.2.3"]
	assert.Equal(t, "1.2.3", got123.Version)
	require.Len(t, got123.RelatedImages, 1, "self-reference (bundle image) must be filtered out")
	assert.Equal(t, "comp-a", got123.RelatedImages[0].Name)
	assert.Equal(t, "sha256:aaa", got123.RelatedImages[0].Digest)

	got999 := byCSV["mypkg.v9.9.9"]
	assert.Equal(t, "9.9.9", got999.Version, "version falls back to csvJson spec.version")
}
