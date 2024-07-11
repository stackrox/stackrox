package resources

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type fakeServerResources struct {
	returnErrorFromFullList bool
	gvksFromFullList        map[schema.GroupVersion][]string

	returnErrorFromSpecificCall bool
	gvksFromSpecificCall        map[schema.GroupVersion][]string
}

func gvAndKindToResourceList(gv schema.GroupVersion, kinds []string) *v1.APIResourceList {
	out := &v1.APIResourceList{
		GroupVersion: gv.String(),
	}
	for _, k := range kinds {
		out.APIResources = append(out.APIResources, v1.APIResource{
			Kind: k,
		})
	}
	return out
}

func (f *fakeServerResources) ServerResourcesForGroupVersion(groupVersion string) (*v1.APIResourceList, error) {
	if f.returnErrorFromSpecificCall {
		return nil, errors.New("NOO")
	}
	gv, err := schema.ParseGroupVersion(groupVersion)
	if err != nil {
		return nil, err
	}
	kinds := f.gvksFromSpecificCall[gv]
	if len(kinds) == 0 {
		return nil, k8sErrors.NewNotFound(schema.GroupResource{Group: gv.Group}, groupVersion)
	}
	return gvAndKindToResourceList(gv, f.gvksFromSpecificCall[gv]), nil
}

func (f *fakeServerResources) ServerResources() ([]*v1.APIResourceList, error) {
	panic("implement me")
}

func (f *fakeServerResources) ServerGroupsAndResources() ([]*v1.APIGroup, []*v1.APIResourceList, error) {
	var err error
	if f.returnErrorFromFullList {
		err = errors.New("NOO")
	}

	var out []*v1.APIResourceList
	for gv, kinds := range f.gvksFromFullList {
		out = append(out, gvAndKindToResourceList(gv, kinds))
	}
	return nil, out, err
}

func (f *fakeServerResources) ServerPreferredResources() ([]*v1.APIResourceList, error) {
	panic("implement me")
}

func (f *fakeServerResources) ServerPreferredNamespacedResources() ([]*v1.APIResourceList, error) {
	panic("implement me")
}

func gvksToMap(gvks []schema.GroupVersionKind) map[schema.GroupVersionKind]*Metadata {
	out := make(map[schema.GroupVersionKind]*Metadata)
	for _, gvk := range gvks {
		out[gvk] = &Metadata{APIResource: v1.APIResource{
			Group:   gvk.Group,
			Version: gvk.Version,
			Kind:    gvk.Kind,
		}}
	}
	return out
}

func gvToKindMap(gvks ...schema.GroupVersionKind) map[schema.GroupVersion][]string {
	out := make(map[schema.GroupVersion][]string)
	for _, gvk := range gvks {
		gv := gvk.GroupVersion()
		out[gv] = append(out[gv], gvk.Kind)
	}
	return out
}

func TestGetAvailableResources(t *testing.T) {
	secretGVK := schema.GroupVersionKind{Version: "v1", Kind: "Secret"}
	pspGVK := schema.GroupVersionKind{Group: "policy", Version: "v1beta1", Kind: "PodSecurityPolicy"}
	deploymentGVK := schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}

	expectedGVKs := map[schema.GroupVersionKind]struct{}{
		secretGVK:     {},
		pspGVK:        {},
		deploymentGVK: {},
	}

	for _, testCase := range []struct {
		desc         string
		client       *fakeServerResources
		expectedGVKs []schema.GroupVersionKind
		errExpected  bool
	}{
		{
			desc: "base case, we get everything in the first try",
			client: &fakeServerResources{
				gvksFromFullList:     gvToKindMap(secretGVK, pspGVK, deploymentGVK),
				gvksFromSpecificCall: gvToKindMap(secretGVK, pspGVK, deploymentGVK),
			},
			expectedGVKs: []schema.GroupVersionKind{secretGVK, pspGVK, deploymentGVK},
		},
		{
			desc: "we never get psps",
			client: &fakeServerResources{
				gvksFromFullList:     gvToKindMap(secretGVK, deploymentGVK),
				gvksFromSpecificCall: gvToKindMap(secretGVK, deploymentGVK),
			},
			expectedGVKs: []schema.GroupVersionKind{secretGVK, deploymentGVK},
		},
		{
			desc: "we never get psps from full list, but do from specific call",
			client: &fakeServerResources{
				gvksFromFullList:        gvToKindMap(secretGVK, deploymentGVK),
				returnErrorFromFullList: true,
				gvksFromSpecificCall:    gvToKindMap(secretGVK, pspGVK, deploymentGVK),
			},
			expectedGVKs: []schema.GroupVersionKind{secretGVK, pspGVK, deploymentGVK},
		},
		{
			desc: "we get only some things from the full list but can't make up the rest from specific calls",
			client: &fakeServerResources{
				gvksFromFullList:            gvToKindMap(secretGVK, deploymentGVK),
				returnErrorFromFullList:     true,
				returnErrorFromSpecificCall: true,
			},
			expectedGVKs: []schema.GroupVersionKind{secretGVK, deploymentGVK},
			errExpected:  true,
		},
		{
			desc: "we get everything but from specific calls",
			client: &fakeServerResources{
				returnErrorFromFullList: true,
				gvksFromSpecificCall:    gvToKindMap(secretGVK, pspGVK, deploymentGVK),
			},
			expectedGVKs: []schema.GroupVersionKind{secretGVK, pspGVK, deploymentGVK},
		},
		{
			desc: "we never get anything",
			client: &fakeServerResources{
				returnErrorFromFullList:     true,
				returnErrorFromSpecificCall: true,
			},
			errExpected: true,
		},
	} {
		c := testCase
		t.Run(c.desc, func(t *testing.T) {
			got, err := GetAvailableResources(c.client, expectedGVKs)
			if c.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, gvksToMap(c.expectedGVKs), got)
		})
	}
}
