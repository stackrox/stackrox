package renderer

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/images/defaults"
	flavorUtils "github.com/stackrox/rox/pkg/images/defaults/testutils"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/zip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestRenderCentralDBOnly(t *testing.T) {
	suite.Run(t, new(centralDBTestSuite))
}

type centralDBTestSuite struct {
	suite.Suite
	restorer      *testbuildinfo.TestBuildTimestampRestorer
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

func (suite *centralDBTestSuite) TearDownSuite() {
	suite.restorer.Restore()
}

func (suite *centralDBTestSuite) testWithHostPath(t *testing.T, c Config, m mode) {
	log.Info("Test host path")
	c.HostPath = &HostPathPersistence{
		DB: &HostPathPersistenceInstance{
			HostPath: "/var/lib/stackrox",
		},
	}
	files, err := render(c, m, suite.testFlavor)
	assert.NoError(t, err)
	suite.verifyFiles(t, files, &c, "hostpath")

	c.HostPath = &HostPathPersistence{
		DB: &HostPathPersistenceInstance{
			HostPath:          "/var/lib/stackrox-db",
			NodeSelectorKey:   "key",
			NodeSelectorValue: "value",
		},
	}
	files, err = render(c, m, suite.testFlavor)
	assert.NoError(t, err)
	suite.verifyFiles(t, files, &c, "hostpath")

	obj := getObj(suite.T(), files, "central/01-central-12-central-db.yaml")
	centralDepoyment := obj.(*appsv1.Deployment)
	require.NotNil(t, centralDepoyment)
	require.NotNil(t, centralDepoyment.Spec)
	require.NotNil(t, centralDepoyment.Spec.Template)
	require.NotNil(t, centralDepoyment.Spec.Template.Spec)
	require.NotNil(t, centralDepoyment.Spec.Template.Spec.NodeSelector)
	require.Equal(t, map[string]string{"key": "value"}, centralDepoyment.Spec.Template.Spec.NodeSelector)
}

func getObj(t *testing.T, files []*zip.File, filePath string) runtime.Object {
	var f *zip.File
	for _, f = range files {
		if f.Name == filePath {
			break
		}
	}
	assert.Equal(t, filePath, f.Name)
	decode := scheme.Codecs.UniversalDeserializer().Decode
	reader := yaml.NewYAMLReader(bufio.NewReader(bytes.NewBuffer(f.Content)))

	yamlBytes, err := reader.Read()
	assert.NoError(t, err)

	obj, _, err := decode(yamlBytes, nil, nil)
	assert.NoError(t, err)
	return obj
}

func (suite *centralDBTestSuite) verifyFiles(t *testing.T, files []*zip.File, c *Config, storage string) {
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
	suite.verifyFile(t, fm, "01-central-05-db-tls-secret.yaml", "Secret", string(suite.testCA.CertPEM()), "stringData", "ca.pem")
	suite.verifyFile(t, fm, "01-central-05-db-tls-secret.yaml", "Secret", string(suite.centralDBCert.CertPEM), "stringData", "cert.pem")
	suite.verifyFile(t, fm, "01-central-05-db-tls-secret.yaml", "Secret", string(suite.centralDBCert.KeyPEM), "stringData", "key.pem")
	// Verify top level resources
	suite.verifyFile(t, fm, "01-central-00-db-serviceaccount.yaml", "ServiceAccount", "central-db", "metadata", "name")
	suite.verifyFile(t, fm, "01-central-08-db-configmap.yaml", "ConfigMap", "central-db-config", "metadata", "name")
	suite.verifyFile(t, fm, "01-central-08-external-db-configmap.yaml", "ConfigMap", "central-external-db", "metadata", "name")
	suite.verifyFile(t, fm, "01-central-12-central-db.yaml", "Deployment", "central-db", "metadata", "name")

	switch storage {
	case "pvc":
		// Verify Persistent Volume Claim
		suite.verifyFile(t, fm, "01-central-11-db-pvc.yaml", "PersistentVolumeClaim", "name", "metadata", "name")
		suite.verifyFile(t, fm, "01-central-11-db-pvc.yaml", "PersistentVolumeClaim", "name", "metadata", "name")
	case "hostpath":
		// Verify Hostpath
		suite.verifyFile(t, fm, "01-central-12-central-db.yaml", "Deployment", "value", "spec", "name")
	default:
		assert.NotContains(t, files, "01-central-11-db-pvc.yaml")
	}
}

func (suite *centralDBTestSuite) verifyFile(t *testing.T, fileMap map[string][]unstructured.Unstructured, fileName string, kind string, value string, fields ...string) {
	objs, ok := fileMap[fileName]
	require.True(t, ok, "%s not found", fileName)
	require.GreaterOrEqual(t, len(objs), 1)
	for _, obj := range objs {
		val, ok, err := unstructured.NestedString(obj.UnstructuredContent(), "kind")
		require.NoError(t, err)
		require.True(t, ok)
		if val == kind {
			val, ok, err := unstructured.NestedString(obj.UnstructuredContent(), fields...)
			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, val, value)
			return
		}
	}
	assert.Failf(t, "Cannot find kind", kind)
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
	suite.verifyFiles(t, files, &c, "pvc")

	c.External = &ExternalPersistence{
		DB: &ExternalPersistenceInstance{
			Name:         "name",
			StorageClass: "storageClass",
		},
	}
	files, err = render(c, m, suite.testFlavor)
	assert.NoError(t, err)
	suite.verifyFiles(t, files, &c, "pvc")
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
