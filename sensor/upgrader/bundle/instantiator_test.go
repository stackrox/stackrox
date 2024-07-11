package bundle

import (
	"io"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type contextForTesting struct {
	yamls []string
}

func (c *contextForTesting) ParseAndValidateObject(data []byte) (*unstructured.Unstructured, error) {
	yamlStr := string(data)
	c.yamls = append(c.yamls, yamlStr)
	return k8sutil.UnstructuredFromYAML(yamlStr)
}

func (c *contextForTesting) InCertRotationMode() bool {
	return false
}

func (c *contextForTesting) IsPodSecurityEnabled() bool {
	return false
}

func TestInstantiator_LoadObjectsFromYAML(t *testing.T) {
	cases := map[string]struct {
		inputDoc     string
		expectedObjs int
		expectedErr  string
	}{
		"empty input": {
			inputDoc: "",
		},
		"only empty documents": {
			inputDoc: `
---
# this only has comments
---
`,
		},
		"object surrounded by empty documents": {
			inputDoc: `
---
# a document with only comments
---
apiVersion: v1
kind: Secret
metadata:
  name: foo
  namespace: bar
---
`,
			expectedObjs: 1,
		},
		"empty documents with invalid YAML": {
			inputDoc: `
---
# only comments
---
---
{
---
`,
			expectedErr: "invalid YAML",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			testingCtx := &contextForTesting{}
			inst := &instantiator{ctx: testingCtx}
			objs, err := inst.loadObjectsFromYAML(func() (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader(c.inputDoc)), nil
			})
			assert.Len(t, objs, c.expectedObjs)
			if c.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), c.expectedErr)
			}
		})
	}
}
