package renderer

import (
	"fmt"
	"strings"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/images/defaults"
	flavorUtils "github.com/stackrox/rox/pkg/images/defaults/testutils"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/zip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestRenderCentralDBOnly(t *testing.T) {
	suite.Run(t, new(centralDBTestSuite))
}

type centralDBTestSuite struct {
	suite.Suite
	testFlavor    defaults.ImageFlavor
	testCA        mtls.CA
	centralDBCert *mtls.IssuedCert
}

func (suite *centralDBTestSuite) SetupSuite() {
	suite.T().Setenv("TEST_VERSIONS", "true")
	suite.testFlavor = flavorUtils.MakeImageFlavorForTest(suite.T())
	var err error
	suite.testCA, err = certgen.GenerateCA()
	require.NoError(suite.T(), err)
	suite.centralDBCert, err = suite.testCA.IssueCertForSubject(mtls.CentralDBSubject)
	require.NoError(suite.T(), err)
}

func (suite *centralDBTestSuite) testWithHostPath(t *testing.T, c Config, m mode) {
	log.Info("Test host path")
	c.HostPath = &HostPathPersistence{
		DB: &HostPathPersistenceInstance{
			HostPath: "/var/lib/stackrox-db",
		},
	}
	files, err := render(c, m, suite.testFlavor)
	assert.NoError(t, err)
	suite.verifyFiles(t, files, &c)

	c.HostPath = &HostPathPersistence{
		DB: &HostPathPersistenceInstance{
			HostPath:          "/var/lib/stackrox-db",
			NodeSelectorKey:   "key",
			NodeSelectorValue: "value",
		},
	}
	files, err = render(c, m, suite.testFlavor)
	assert.NoError(t, err)
	suite.verifyFiles(t, files, &c)
}

func (suite *centralDBTestSuite) verifyFiles(t *testing.T, files []*zip.File, c *Config) {
	fm := make(map[string][]unstructured.Unstructured, len(files))
	for _, f := range files {
		if f.Name == "README" || strings.HasSuffix(f.Name, ".sh") {
			assert.NotEmpty(t, f.Content)
			continue
		}
		unstructuredObjs, err := k8sutil.UnstructuredFromYAMLMulti(string(f.Content))
		require.NoError(t, err, f.Name)
		fm[strings.TrimPrefix(f.Name, "central/")] = unstructuredObjs
	}
	// Verify secrets overwrite
	verifyNestedFieldInFile(t, fm, "01-central-05-db-tls-secret.yaml", "Secret", string(suite.testCA.CertPEM()), "stringData", "ca.pem")
	verifyNestedFieldInFile(t, fm, "01-central-05-db-tls-secret.yaml", "Secret", string(suite.centralDBCert.CertPEM), "stringData", "cert.pem")
	verifyNestedFieldInFile(t, fm, "01-central-05-db-tls-secret.yaml", "Secret", string(suite.centralDBCert.KeyPEM), "stringData", "key.pem")
	// Verify top level resources
	verifyNestedFieldInFile(t, fm, "01-central-00-db-serviceaccount.yaml", "ServiceAccount", "central-db", "metadata", "name")
	verifyNestedFieldInFile(t, fm, "01-central-08-db-configmap.yaml", "ConfigMap", "central-db-config", "metadata", "name")
	verifyNestedFieldInFile(t, fm, "01-central-08-external-db-configmap.yaml", "ConfigMap", "central-external-db", "metadata", "name")
	verifyNestedFieldInFile(t, fm, "01-central-12-central-db.yaml", "Deployment", "central-db", "metadata", "name")

	if c.HasCentralDBExternal() {
		// Verify Persistent Volume Claim
		verifyNestedFieldInFile(t, fm, "01-central-11-db-pvc.yaml", "PersistentVolumeClaim", "name", "metadata", "name")
		verifyNestedFieldInFile(t, fm, "01-central-11-db-pvc.yaml", "PersistentVolumeClaim", "name", "metadata", "name")
	} else if c.HasCentralDBHostPath() {
		// Verify Hostpath
		contents := getMatchedMapInFile(t, fm, "01-central-12-central-db.yaml", "Deployment")
		vals, ok, err := unstructured.NestedSlice(contents, "spec", "template", "spec", "volumes")
		assert.NoError(t, err)
		assert.True(t, ok)
		for _, val := range vals {
			vol := val.(map[string]interface{})
			if vol["name"] == "disk" {
				verifyNestedString(t, vol, "/var/lib/stackrox-db", "hostPath", "path")
				break
			}
		}
		if c.HostPath.DB.NodeSelectorKey != "" {
			verifyNestedFieldInFile(t, fm, "01-central-12-central-db.yaml", "Deployment", c.HostPath.DB.NodeSelectorValue, "spec", "template", "spec", "nodeSelector", c.HostPath.DB.NodeSelectorKey)
		}
	} else {
		assert.NotContains(t, files, "01-central-11-db-pvc.yaml")
	}
}

func verifyNestedString(t *testing.T, objMap map[string]interface{}, value string, fields ...string) {
	val, ok, err := unstructured.NestedString(objMap, fields...)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, value, val)
}

func getMatchedMapInFile(t *testing.T, fileMap map[string][]unstructured.Unstructured, fileName string, kind string) map[string]interface{} {
	objs, ok := fileMap[fileName]
	require.True(t, ok, "%s not found", fileName)
	require.GreaterOrEqual(t, len(objs), 1)
	for _, obj := range objs {
		val, ok, err := unstructured.NestedString(obj.UnstructuredContent(), "kind")
		require.NoError(t, err)
		require.True(t, ok)
		if val == kind {
			return obj.UnstructuredContent()
		}
	}
	assert.Failf(t, "Cannot find kind", kind)
	return nil
}

func verifyNestedFieldInFile(t *testing.T, fileMap map[string][]unstructured.Unstructured, fileName string, kind string, value string, fields ...string) {
	contents := getMatchedMapInFile(t, fileMap, fileName, kind)
	verifyNestedString(t, contents, value, fields...)
}

func (suite *centralDBTestSuite) testWithPV(t *testing.T, c Config, m mode) {
	log.Info("Test PV")
	c.External = &ExternalPersistence{
		DB: &ExternalPersistenceInstance{
			Name: "name",
		},
	}
	files, err := render(c, m, suite.testFlavor)
	assert.NoError(t, err)
	suite.verifyFiles(t, files, &c)

	c.External = &ExternalPersistence{
		DB: &ExternalPersistenceInstance{
			Name:         "name",
			StorageClass: "storageClass",
		},
	}
	files, err = render(c, m, suite.testFlavor)
	assert.NoError(t, err)
	suite.verifyFiles(t, files, &c)
}

func (suite *centralDBTestSuite) TestRenderCentralDBBundle() {
	for _, orch := range []storage.ClusterType{storage.ClusterType_KUBERNETES_CLUSTER, storage.ClusterType_OPENSHIFT_CLUSTER, storage.ClusterType_OPENSHIFT4_CLUSTER} {
		suite.T().Run(fmt.Sprintf("DbBundle-%s", orch), func(t *testing.T) {
			centralFileMap := make(map[string][]byte, 4)
			centralFileMap["central-db-password"] = []byte("Apassword")
			centralFileMap["central-db-cert.pem"] = suite.centralDBCert.CertPEM
			centralFileMap["central-db-key.pem"] = suite.centralDBCert.KeyPEM
			centralFileMap[mtls.CACertFileName] = suite.testCA.CertPEM()

			conf := Config{
				ClusterType: storage.ClusterType_KUBERNETES_CLUSTER,
				K8sConfig: &K8sConfig{
					CommonConfig: CommonConfig{
						CentralDBImage: "stackrox/central-db:2.2.11.0-57-g392c0f5bed-dirty",
					},
					EnableCentralDB: true,
				},
				SecretsByteMap: centralFileMap,
			}
			conf.K8sConfig.DeploymentFormat = v1.DeploymentFormat_KUBECTL
			conf.ClusterType = orch
			suite.testWithHostPath(t, conf, centralDBOnly)
			suite.testWithPV(t, conf, centralDBOnly)
		})
	}
}
