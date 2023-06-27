package overlays

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type dummy struct {
	Kind       string    `json:"kind,omitempty"`
	APIVersion string    `json:"apiVersion,omitempty"`
	Spec       DummySpec `json:"spec,omitempty"`
}

type DummySpec struct {
	Overlays []*v1alpha1.K8sObjectOverlay `json:"overlays,omitempty"`
}

var manifestBase = `
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
  namespace: test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: test
        image: test
        ports:
        - containerPort: 8080
`

var manifestBytes []byte

func init() {
	manifestBytes = []byte(manifestBase)
}

func TestPostRenderer(t *testing.T) {

	cr := &dummy{
		APIVersion: "blah.com/v1",
		Kind:       "dummy",
		Spec: DummySpec{
			Overlays: []*v1alpha1.K8sObjectOverlay{
				{
					Kind:       "Deployment",
					Name:       "test",
					APIVersion: "apps/v1",
					Patches: []*v1alpha1.K8sObjectOverlayPatch{
						{
							Path:  "metadata.annotations",
							Value: `test: test`,
						},
					},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(cr)
	require.NoError(t, err)

	obj := &unstructured.Unstructured{}
	require.NoError(t, err)
	require.NoError(t, obj.UnmarshalJSON(jsonBytes))

	r := OverlayPostRenderer{
		obj:              obj,
		defaultNamespace: "test",
	}

	got, err := r.Run(bytes.NewBuffer(manifestBytes))
	require.NoError(t, err)

	want := `apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    test: test
  name: test
  namespace: test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
        - image: test
          name: test
          ports:
            - containerPort: 8080

---
`
	assert.Equal(t, want, got.String())

}
