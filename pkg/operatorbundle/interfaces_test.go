package operatorbundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepositoryFromReference(t *testing.T) {
	tests := map[string]struct {
		reference string
		want      string
	}{
		"digest stripped": {
			reference: "registry.redhat.io/albo/controller-rhel9@sha256:abc",
			want:      "registry.redhat.io/albo/controller-rhel9",
		},
		"tag stripped": {
			reference: "registry.redhat.io/albo/controller-rhel9:1.2",
			want:      "registry.redhat.io/albo/controller-rhel9",
		},
		"registry port preserved": {
			reference: "localhost:5000/albo/controller:1.2",
			want:      "localhost:5000/albo/controller",
		},
		"registry port preserved without tag": {
			reference: "localhost:5000/albo/controller",
			want:      "localhost:5000/albo/controller",
		},
		"tag and digest": {
			reference: "registry.redhat.io/albo/controller-rhel9:1.2@sha256:abc",
			want:      "registry.redhat.io/albo/controller-rhel9",
		},
		"plain repository": {
			reference: "registry.redhat.io/albo/controller-rhel9",
			want:      "registry.redhat.io/albo/controller-rhel9",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, RepositoryFromReference(tc.reference))
		})
	}
}
