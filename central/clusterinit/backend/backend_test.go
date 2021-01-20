package backend

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/clusterinit/backend/mocks"
	rocksdbStore "github.com/stackrox/rox/central/clusterinit/store/rocksdb"
	"github.com/stackrox/rox/central/clusters"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

const (
	testData = "testdata"
)

func readCertAndKey(serviceName string) (*mtls.IssuedCert, error) {
	certFile := path.Join(testData, fmt.Sprintf("%s-cert.pem", serviceName))
	keyFile := path.Join(testData, fmt.Sprintf("%s-key.pem", serviceName))

	certPEM, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, err
	}

	keyPEM, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, err
	}

	return &mtls.IssuedCert{
		CertPEM:  certPEM,
		KeyPEM:   keyPEM,
		X509Cert: x509Cert,
		ID:       nil, // For testing we get away without filling this in.
	}, nil
}

func (s *clusterInitBackendTestSuite) initMockData() error {
	caCertPEM, err := ioutil.ReadFile("testdata/ca-cert.pem")
	if err != nil {
		return err
	}

	sensorIssuedCert, err := readCertAndKey("sensor")
	if err != nil {
		return err
	}
	collectorIssuedCert, err := readCertAndKey("collector")
	if err != nil {
		return err
	}
	admissionControlIssuedCert, err := readCertAndKey("admission-control")
	if err != nil {
		return err
	}

	s.certBundle = map[storage.ServiceType]*mtls.IssuedCert{
		storage.ServiceType_SENSOR_SERVICE:            sensorIssuedCert,
		storage.ServiceType_COLLECTOR_SERVICE:         collectorIssuedCert,
		storage.ServiceType_ADMISSION_CONTROL_SERVICE: admissionControlIssuedCert,
	}
	s.caCert = string(caCertPEM)

	return nil
}

func TestClusterInitBackend(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(clusterInitBackendTestSuite))
}

type clusterInitBackendTestSuite struct {
	suite.Suite
	backend      Backend
	ctx          context.Context
	rocksDB      *rocksdb.RocksDB
	certProvider *mocks.MockCertificateProvider
	mockCtrl     *gomock.Controller
	certBundle   clusters.CertBundle
	caCert       string
}

func (s *clusterInitBackendTestSuite) SetupTest() {
	err := s.initMockData()
	s.Require().NoError(err, "retrieving test data for mocking")
	s.mockCtrl = gomock.NewController(s.T())
	m := mocks.NewMockCertificateProvider(s.mockCtrl)
	s.rocksDB = rocksdbtest.RocksDBForT(s.T())
	store, err := rocksdbStore.NewStore(s.rocksDB)
	s.Require().NoError(err)
	s.backend = newBackend(store, m)
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	s.certProvider = m

	// Configure CertificateProvider mock.
	s.certProvider.EXPECT().GetCA().Return(s.caCert, nil).AnyTimes()
	s.certProvider.EXPECT().GetBundle().Return(s.certBundle, uuid.NewV4(), nil).AnyTimes()
}

func (s *clusterInitBackendTestSuite) TestIssuing() {
	ctx := s.ctx

	// Issue new init bundle.
	initBundle, err := s.backend.Issue(ctx, "test1")
	s.Require().NoError(err)
	id := initBundle.Meta.Id

	caCert, err := s.certProvider.GetCA()
	s.Require().NoError(err)

	certBundle, _, err := s.certProvider.GetBundle()
	s.Require().NoError(err)

	s.Require().Equal(initBundle.CaCert, caCert)
	s.Require().Equal(initBundle.CertBundle, certBundle)

	// Verify YAML-rendered init bundle looks as expected.
	expected := map[string]interface{}{
		"ca": map[string]interface{}{
			"cert": caCert,
		},
		"sensor": map[string]interface{}{
			"serviceTLS": map[string]interface{}{
				"cert": string(certBundle[storage.ServiceType_SENSOR_SERVICE].CertPEM),
				"key":  string(certBundle[storage.ServiceType_SENSOR_SERVICE].KeyPEM),
			},
		},
		"collector": map[string]interface{}{
			"serviceTLS": map[string]interface{}{
				"cert": string(certBundle[storage.ServiceType_COLLECTOR_SERVICE].CertPEM),
				"key":  string(certBundle[storage.ServiceType_COLLECTOR_SERVICE].KeyPEM),
			},
		},
		"admissionControl": map[string]interface{}{
			"serviceTLS": map[string]interface{}{
				"cert": string(certBundle[storage.ServiceType_ADMISSION_CONTROL_SERVICE].CertPEM),
				"key":  string(certBundle[storage.ServiceType_ADMISSION_CONTROL_SERVICE].KeyPEM),
			},
		},
	}

	initBundleYAML, err := initBundle.RenderAsYAML()
	s.Require().NoError(err)

	// Compute diff.
	var parsed map[string]interface{}
	err = yaml.Unmarshal(initBundleYAML, &parsed)
	s.Require().NoError(err)

	diff := maputil.DiffGenericMap(expected, parsed)
	if diff != nil {
		fmt.Fprintln(os.Stderr, "Init bundle diff:")
		prettyDiff, err := json.MarshalIndent(diff, "", "  ")
		s.Require().NoError(err, "failed to serialize diff as JSON")
		fmt.Fprintf(os.Stderr, "%s\n", prettyDiff)
	}
	s.Require().Nil(diff)

	// Verify the newly generated bundle is listed.
	initBundleMetas, err := s.backend.GetAll(ctx)
	s.Require().NoError(err)
	var initBundleMeta *storage.InitBundleMeta
	for _, m := range initBundleMetas {
		if m.Id == id {
			initBundleMeta = m
			break
		}
	}
	s.Require().NotNilf(initBundleMeta, "failed to find newly generated init bundle with ID %s in listing", id)
	s.Require().Equal(initBundle.Meta, initBundleMeta, "init bundle meta data changed between generation and listing")

	// Verify it is not revoked.
	s.Require().False(initBundleMeta.IsRevoked, "newly generated init bundle is revoked")
}

// Tests if attempt to issue two init bundles with the same name fails as expected.
func (s *clusterInitBackendTestSuite) TestIssuingWithDuplicateName() {
	ctx := s.ctx
	_, err := s.backend.Issue(ctx, "test2")
	s.Require().NoError(err)
	_, err = s.backend.Issue(ctx, "test2")
	s.Require().Error(err, "issuing two init bundles with the same name")
}

func (s *clusterInitBackendTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.rocksDB)
}
