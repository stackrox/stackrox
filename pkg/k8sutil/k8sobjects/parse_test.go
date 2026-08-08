package k8sobjects

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestParseRef(t *testing.T) {
	tests := map[string]struct {
		input       string
		want        ObjectRef
		wantErr     bool
		errContains string
	}{
		"namespaced deployment with apps group": {
			input: "Deployment:apps/v1:default/nginx",
			want: ObjectRef{
				GVK: schema.GroupVersionKind{
					Group:   "apps",
					Version: "v1",
					Kind:    "Deployment",
				},
				Namespace: "default",
				Name:      "nginx",
			},
		},
		"namespaced pod with core group": {
			input: "Pod:v1:kube-system/coredns",
			want: ObjectRef{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "Pod",
				},
				Namespace: "kube-system",
				Name:      "coredns",
			},
		},
		"cluster-scoped namespace": {
			input: "Namespace:v1:default",
			want: ObjectRef{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "Namespace",
				},
				Namespace: "",
				Name:      "default",
			},
		},
		"cluster-scoped cluster role": {
			input: "ClusterRole:rbac.authorization.k8s.io/v1:admin",
			want: ObjectRef{
				GVK: schema.GroupVersionKind{
					Group:   "rbac.authorization.k8s.io",
					Version: "v1",
					Kind:    "ClusterRole",
				},
				Namespace: "",
				Name:      "admin",
			},
		},
		"custom resource with alpha version": {
			input: "MyResource:example.com/v1alpha1:prod/instance-1",
			want: ObjectRef{
				GVK: schema.GroupVersionKind{
					Group:   "example.com",
					Version: "v1alpha1",
					Kind:    "MyResource",
				},
				Namespace: "prod",
				Name:      "instance-1",
			},
		},
		"names with dashes and numbers": {
			input: "Deployment:apps/v1:test-ns-123/app-deployment-v2",
			want: ObjectRef{
				GVK: schema.GroupVersionKind{
					Group:   "apps",
					Version: "v1",
					Kind:    "Deployment",
				},
				Namespace: "test-ns-123",
				Name:      "app-deployment-v2",
			},
		},
		"empty string": {
			input:       "",
			wantErr:     true,
			errContains: "unexpected number of colons",
		},
		"no colons": {
			input:       "invalid",
			wantErr:     true,
			errContains: "unexpected number of colons",
		},
		"one colon": {
			input:       "Kind:version",
			wantErr:     true,
			errContains: "unexpected number of colons",
		},
		"too many colons": {
			input:       "Kind:group/v1:namespace:name",
			wantErr:     true,
			errContains: "unexpected number of colons",
		},
		"group/version without slash is treated as version only": {
			input: "Kind:invalid-version-format:name",
			want: ObjectRef{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "invalid-version-format",
					Kind:    "Kind",
				},
				Namespace: "",
				Name:      "name",
			},
		},
		"too many slashes in name": {
			input:       "Kind:apps/v1:namespace/sub/name",
			wantErr:     true,
			errContains: "too many slashes",
		},
		"empty parts with colons": {
			input: "::",
			want: ObjectRef{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "",
					Kind:    "",
				},
				Namespace: "",
				Name:      "",
			},
		},
		"empty kind": {
			input: ":v1:default/pod",
			want: ObjectRef{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "",
				},
				Namespace: "default",
				Name:      "pod",
			},
		},
		"empty name": {
			input: "Pod:v1:",
			want: ObjectRef{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "Pod",
				},
				Namespace: "",
				Name:      "",
			},
		},
		"namespace with empty name": {
			input: "Pod:v1:default/",
			want: ObjectRef{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "Pod",
				},
				Namespace: "default",
				Name:      "",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ParseRef(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
