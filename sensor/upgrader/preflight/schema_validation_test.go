package preflight

import (
	_ "embed"
	"testing"

	openapi_v2 "github.com/google/gnostic-models/openapiv2"
	"github.com/stackrox/rox/pkg/gziputil"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// Kubernetes OpenAPI schema from server version v1.22.12-gke.1200
	// Created via
	//   kubectl get --raw /openapi/v2 | gzip -c
	//go:embed testdata/k8s-1.22.json.gz
	k8sSchemaJSONGZ1_22 []byte
)

func TestValidateObject(t *testing.T) {
	obj, err := k8sutil.UnstructuredFromYAML(`
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: test-netpol
  namespace: test-ns
`)
	require.NoError(t, err)

	k8sSchemaJSON1_22, err := gziputil.Decompress(k8sSchemaJSONGZ1_22)
	require.NoError(t, err)

	schemaDoc, err := openapi_v2.ParseDocument(k8sSchemaJSON1_22)
	require.NoError(t, err)

	validator, err := common.ValidatorFromOpenAPIDoc(schemaDoc)
	require.NoError(t, err)

	assert.NoError(t, validateObject(obj, validator))
}
