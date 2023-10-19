//go:build sql_integration

package backend

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stackrox/rox/central/clusterinit/backend/access"
	"github.com/stackrox/rox/central/clusterinit/backend/certificate/mocks"
	"github.com/stackrox/rox/central/clusterinit/store"
	pgStore "github.com/stackrox/rox/central/clusterinit/store/postgres"
	"github.com/stackrox/rox/central/clusters"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	testData = "testdata"
)

func readCertAndKey(serviceName string) (*mtls.IssuedCert, error) {
	certFile := path.Join(testData, fmt.Sprintf("%s-cert.pem", serviceName))
	keyFile := path.Join(testData, fmt.Sprintf("%s-key.pem", serviceName))

	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return nil, err
	}

	keyPEM, err := os.ReadFile(keyFile)
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
	caCertPEM, err := os.ReadFile("testdata/ca-cert.pem")
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
	db           postgres.DB
	certProvider *mocks.MockProvider
	mockCtrl     *gomock.Controller
	certBundle   clusters.CertBundle
	caCert       string
}

func (s *clusterInitBackendTestSuite) SetupTest() {
	err := s.initMockData()
	s.Require().NoError(err, "retrieving test data for mocking")
	s.mockCtrl = gomock.NewController(s.T())
	m := mocks.NewMockProvider(s.mockCtrl)

	s.db = pgtest.ForT(s.T())

	pgStore := pgStore.New(s.db)
	s.Require().NoError(err)
	s.backend = newBackend(store.NewStore(pgStore), m)
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	s.certProvider = m

	// Configure CertificateProvider mock.
	s.certProvider.EXPECT().GetCA().Return(s.caCert, nil).AnyTimes()
}

func (s *clusterInitBackendTestSuite) TestInitBundleLifecycle() {
	ctx := s.ctx

	s.certProvider.EXPECT().GetBundle().Return(s.certBundle, uuid.NewV4(), nil).AnyTimes()

	// Issue new init bundle.
	initBundle, err := s.backend.Issue(ctx, "test1")
	s.Require().NoError(err)
	id := initBundle.Meta.Id

	err = s.backend.CheckRevoked(ctx, id)
	s.Require().NoErrorf(err, "newly generated init bundle %q is revoked", id)

	caCert, err := s.certProvider.GetCA()
	s.Require().NoError(err)

	certBundle, _, err := s.certProvider.GetBundle()
	s.Require().NoError(err)

	s.Require().Equal(initBundle.CACert, caCert)
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

	// Verify properties about the generated Kubernetes secreted
	yamlBytes, err := initBundle.RenderAsK8sSecrets()
	s.Require().NoError(err)

	unstructuredObjs, err := k8sutil.UnstructuredFromYAMLMulti(string(yamlBytes))
	s.Require().NoError(err)

	for _, obj := range unstructuredObjs {
		name := obj.GetName()
		s.Require().True(stringutils.ConsumeSuffix(&name, "-tls"))
		s.Equal(initBundle.Meta.GetName(), obj.GetAnnotations()["init-bundle.stackrox.io/name"])
		s.Equal(initBundle.Meta.GetId(), obj.GetAnnotations()["init-bundle.stackrox.io/id"])
		s.Equal(initBundle.Meta.GetCreatedAt().String(), obj.GetAnnotations()["init-bundle.stackrox.io/created-at"])
		s.Equal(initBundle.Meta.GetExpiresAt().String(), obj.GetAnnotations()["init-bundle.stackrox.io/expires-at"])

		val, ok, err := unstructured.NestedString(obj.UnstructuredContent(), "stringData", "ca.pem")
		s.Require().NoError(err)
		s.Require().True(ok)
		s.Equal(caCert, val)

		var svcType storage.ServiceType
		switch name {
		case "sensor":
			svcType = storage.ServiceType_SENSOR_SERVICE
		case "collector":
			svcType = storage.ServiceType_COLLECTOR_SERVICE
		case "admission-control":
			svcType = storage.ServiceType_ADMISSION_CONTROL_SERVICE
		}
		s.Require().NotZerof(svcType, "invalid service name %s", name)

		val, ok, err = unstructured.NestedString(obj.UnstructuredContent(), "stringData", name+"-cert.pem")
		s.Require().NoError(err)
		s.Require().True(ok)
		s.Equal(string(initBundle.CertBundle[svcType].CertPEM), val)

		val, ok, err = unstructured.NestedString(obj.UnstructuredContent(), "stringData", name+"-key.pem")
		s.Require().NoError(err)
		s.Require().True(ok)
		s.Equal(string(initBundle.CertBundle[svcType].KeyPEM), val)
	}

	// Verify the newly generated bundle is listed.
	initBundleMetas, err := s.backend.GetAll(ctx)
	s.Require().NoError(err)
	oldInitBundleMetasLength := len(initBundleMetas)
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

	// Verify it can be revoked.
	err = s.backend.Revoke(ctx, id)
	s.Require().NoErrorf(err, "revoking newly generated init bundle %q", id)

	err = s.backend.CheckRevoked(ctx, id)
	s.Require().Errorf(err, "init bundle %q is not revoked", id)

	initBundleMetas, err = s.backend.GetAll(ctx)
	s.Require().NoError(err)
	s.Require().Len(initBundleMetas, oldInitBundleMetasLength-1, "unexpected number of returned init bundles")

	initBundleMeta = nil
	for _, m := range initBundleMetas {
		if m.Id == id {
			initBundleMeta = m
			break
		}
	}
	s.Require().Nilf(initBundleMeta, "revoked init bundle %q contained in listing", id)

}

// Tests if attempt to issue two init bundles with the same name fails as expected.
func (s *clusterInitBackendTestSuite) TestIssuingWithDuplicateName() {
	ctx := s.ctx

	s.certProvider.EXPECT().GetBundle().Return(s.certBundle, uuid.NewV4(), nil)
	_, err := s.backend.Issue(ctx, "test2")
	s.Require().NoError(err)

	s.certProvider.EXPECT().GetBundle().Return(s.certBundle, uuid.NewV4(), nil)
	_, err = s.backend.Issue(ctx, "test2")
	s.Require().Error(err, "issuing two init bundles with the same name")
}

func (s *clusterInitBackendTestSuite) TestValidateClientCertificateEmptyChain() {
	ctx := s.ctx

	err := s.backend.ValidateClientCertificate(ctx, nil)
	s.Require().Error(err)
	s.Equal("empty cert chain passed", err.Error())
}

func (s *clusterInitBackendTestSuite) TestValidateClientCertificateNotFound() {
	ctx := s.ctx
	id := uuid.NewV4()
	certs := []mtls.CertInfo{
		{Subject: pkix.Name{Organization: []string{id.String()}}},
	}

	err := s.backend.ValidateClientCertificate(ctx, certs)
	s.Require().Error(err)
	s.Equal(fmt.Sprintf("failed checking init bundle status %[1]q: retrieving init bundle %[1]q: init bundle not found", id), err.Error())
}

func (s *clusterInitBackendTestSuite) TestValidateClientCertificateEphemeralInitBundle() {
	ctx := s.ctx
	id := uuid.NewV4()
	certs := []mtls.CertInfo{
		{Subject: pkix.Name{
			CommonName:   centralsensor.EphemeralInitCertClusterID,
			Organization: []string{id.String()},
		}},
	}

	err := s.backend.ValidateClientCertificate(ctx, certs)
	s.Require().NoError(err)
}

func (s *clusterInitBackendTestSuite) TestValidateClientCertificate() {
	// To access the revoke check a token should be passed without any access rights.
	ctxWithoutSAC := context.Background()

	s.certProvider.EXPECT().GetBundle().Return(s.certBundle, uuid.NewV4(), nil)

	meta, err := s.backend.Issue(s.ctx, "revoke-check")
	s.Require().NoError(err)

	certs := []mtls.CertInfo{
		{Subject: pkix.Name{Organization: []string{meta.Meta.Id}}},
	}

	// Success for valid init bundles
	err = s.backend.ValidateClientCertificate(ctxWithoutSAC, certs)
	s.Require().NoError(err)

	err = s.backend.Revoke(s.ctx, meta.Meta.Id)
	s.Require().NoError(err)

	// Fail for a revoked init bundles
	err = s.backend.ValidateClientCertificate(ctxWithoutSAC, certs)
	s.Require().Error(err)
	s.Contains(err.Error(), "init bundle is revoked")
}

func (s *clusterInitBackendTestSuite) TestValidateClientCertificateShouldIgnoreNonInitBundles() {
	// To access the revoke check a token should be passed without any access rights.
	ctxWithoutSAC := context.Background()

	certs := []mtls.CertInfo{
		{Subject: pkix.Name{Organization: []string{}}},
	}

	err := s.backend.ValidateClientCertificate(ctxWithoutSAC, certs)
	s.Require().NoError(err)
}

// Tests if names can be reused after revoking.
func (s *clusterInitBackendTestSuite) TestIssuingAfterRevoking() {
	name := "test3"
	ctx := s.ctx

	s.certProvider.EXPECT().GetBundle().Return(s.certBundle, uuid.NewV4(), nil)
	initBundle, err := s.backend.Issue(ctx, name)
	id := initBundle.Meta.GetId()
	s.Require().NoError(err)

	s.certProvider.EXPECT().GetBundle().Return(s.certBundle, uuid.NewV4(), nil)
	_, err = s.backend.Issue(ctx, name)
	s.Require().Error(err, "issuing two init bundles with the same name")

	err = s.backend.Revoke(ctx, id)
	s.Require().NoErrorf(err, "revoking init bundle %q", id)

	s.certProvider.EXPECT().GetBundle().Return(s.certBundle, uuid.NewV4(), nil)
	_, err = s.backend.Issue(ctx, name)
	s.Require().NoError(err)
}

func (s *clusterInitBackendTestSuite) TestCheckAccess() {
	cases := map[string]struct {
		ctx         context.Context
		access      storage.Access
		shouldFail  bool
		expectedErr error
	}{
		"read access to both Administration and Integration should allow read access": {
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Administration, resources.Integration))),
			access: storage.Access_READ_ACCESS,
		},
		"read access to both Administration and Integration should not allow write access": {
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Administration, resources.Integration))),
			access:      storage.Access_READ_WRITE_ACCESS,
			shouldFail:  true,
			expectedErr: errox.NotAuthorized,
		},
		"read access to only Administration should not allow read access": {
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Administration))),
			access:      storage.Access_READ_ACCESS,
			shouldFail:  true,
			expectedErr: errox.NotAuthorized,
		},
		"read access to only Integration should not allow read access": {
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Integration))),
			access:      storage.Access_READ_ACCESS,
			shouldFail:  true,
			expectedErr: errox.NotAuthorized,
		},
		"write access to both should allow write access": {
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Administration, resources.Integration))),
			access: storage.Access_READ_WRITE_ACCESS,
		},
		"write access to only Administration should not allow write access": {
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Administration))),
			access:      storage.Access_READ_WRITE_ACCESS,
			shouldFail:  true,
			expectedErr: errox.NotAuthorized,
		},
		"write access to only Integration should not allow write access": {
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Integration))),
			access:      storage.Access_READ_WRITE_ACCESS,
			shouldFail:  true,
			expectedErr: errox.NotAuthorized,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			err := access.CheckAccess(c.ctx, c.access)
			if c.shouldFail {
				s.Require().Error(err)
				s.ErrorIs(err, c.expectedErr)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *clusterInitBackendTestSuite) TearDownTest() {
	s.db.Close()
}
