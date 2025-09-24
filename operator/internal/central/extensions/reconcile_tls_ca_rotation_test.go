package extensions

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"maps"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/go-logr/logr"
	"github.com/stackrox/rox/generated/storage"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	commonExtensions "github.com/stackrox/rox/operator/internal/common/extensions"
	"github.com/stackrox/rox/operator/internal/common/rendercache"
	"github.com/stackrox/rox/operator/internal/types"
	"github.com/stackrox/rox/operator/internal/utils"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCentralCARotation(t *testing.T) {
	t.Setenv(envCentralCARotationEnabled, "true")
	baseCase := secretReconciliationTestCase{
		Spec: basicSpecWithScanner(true, true),
	}
	central := buildFakeCentral(baseCase)
	client := buildFakeClient(t, baseCase, central)

	// similar to reconcileCentralTLS, but injects a custom currentTime
	runAt := func(ctx context.Context, central *platform.Central, c ctrlClient.Client, d ctrlClient.Reader, statusUpdater func(updateStatusFunc),
		log logr.Logger, renderCache *rendercache.RenderCache, currentTime time.Time) error {
		run := &createCentralTLSExtensionRun{
			SecretReconciliator:   commonExtensions.NewSecretReconciliator(c, d, central),
			centralObj:            central,
			currentTime:           currentTime,
			extraIssueCertOptions: []mtls.IssueCertOption{mtls.WithValidityNotBefore(currentTime)},
			renderCache:           renderCache,
		}
		return run.Execute(ctx)
	}

	baseTime := time.Now()

	type timepoint struct {
		offset                  time.Duration
		verifyCentralFunc       secretVerifyFunc
		fixSecondaryCANotBefore bool
	}

	year := 365 * 24 * time.Hour
	timepoints := []timepoint{
		{
			// Year 1: should have one CA
			offset:            0,
			verifyCentralFunc: verifyCentralCertNoSecondaryCA,
		},
		{
			// Year 2: should have one CA still
			offset:            2*year + time.Hour,
			verifyCentralFunc: verifyCentralCertNoSecondaryCA,
		},
		{
			// Year 3: should add a secondary CA
			offset:                  3*year + time.Hour,
			verifyCentralFunc:       verifyCentralCertWithSecondaryCA,
			fixSecondaryCANotBefore: true,
		},
		{
			// Year 4: still two CAs
			offset:            4*year + time.Hour,
			verifyCentralFunc: verifyCentralCertWithSecondaryCA,
		},
		{
			// Year 5: original CA is now expired, should only have 1 CA again
			offset:            5*year + time.Hour,
			verifyCentralFunc: verifyCentralCertNoSecondaryCA,
		},
	}

	commonSecrets := map[string]secretVerifyFunc{
		"central-db-tls":         verifyCentralServiceCert(storage.ServiceType_CENTRAL_DB_SERVICE),
		"scanner-tls":            verifyCentralServiceCert(storage.ServiceType_SCANNER_SERVICE),
		"scanner-db-tls":         verifyCentralServiceCert(storage.ServiceType_SCANNER_DB_SERVICE),
		"scanner-v4-indexer-tls": verifyCentralServiceCert(storage.ServiceType_SCANNER_V4_INDEXER_SERVICE),
		"scanner-v4-matcher-tls": verifyCentralServiceCert(storage.ServiceType_SCANNER_V4_MATCHER_SERVICE),
		"scanner-v4-db-tls":      verifyCentralServiceCert(storage.ServiceType_SCANNER_V4_DB_SERVICE),
	}

	for _, tp := range timepoints {
		currentTime := baseTime.Add(tp.offset)

		tc := baseCase
		tc.ExpectedCreatedSecrets = map[string]secretVerifyFunc{
			"central-tls": tp.verifyCentralFunc,
		}
		maps.Copy(tc.ExpectedCreatedSecrets, commonSecrets)

		testSecretReconciliationAtTime(t, central, client, runAt, tc, currentTime)

		if tp.fixSecondaryCANotBefore {
			// Hack: certgen.GenerateCA() can only generate a CA starting from time.Now(), and plugging in a customized
			// NotBefore value is not straightforward.
			// This function updates the NotBefore property of CA certificate to the desired time.
			updateSecondaryCANotBefore(t, client, currentTime)
		}
	}
}

func TestCentralCARotationAddSecondaryAndPromote(t *testing.T) {
	// This test simulates an upgrade from an ACS version that does not have CA rotation enabled to a version that does.
	// The Central CA is 4+ years old, and so a new secondary CA should be added and simultaneously promoted to primary.
	t.Setenv(envCentralCARotationEnabled, "true")

	oldCA, err := certgen.GenerateCA()
	require.NoError(t, err)

	// Update the CA's NotBefore to be 4+ years ago to simulate an old deployment
	privateKey, err := helpers.ParsePrivateKeyPEM(oldCA.KeyPEM())
	require.NoError(t, err)
	oldCATime := time.Now().Add(-4*365*24*time.Hour - 24*time.Hour)
	oldCACertBytes, err := updateCertificateNotBefore(oldCA.Certificate(), privateKey, oldCATime)
	require.NoError(t, err)
	oldCAPEM, err := mtls.LoadCAForSigning(oldCACertBytes, oldCA.KeyPEM())
	require.NoError(t, err)

	centralCert, err := oldCAPEM.IssueCertForSubject(mtls.CentralSubject, mtls.WithNamespace(testutils.TestNamespace), mtls.WithValidityNotBefore(oldCATime))
	require.NoError(t, err)

	fileMap := map[string][]byte{
		mtls.CACertFileName:      oldCAPEM.CertPEM(),
		mtls.CAKeyFileName:       oldCAPEM.KeyPEM(),
		mtls.ServiceCertFileName: centralCert.CertPEM,
		mtls.ServiceKeyFileName:  centralCert.KeyPEM,
	}

	testCase := secretReconciliationTestCase{
		Spec: basicSpecWithScanner(true, true),
		ExistingManaged: []*coreV1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "central-tls",
					Namespace: testutils.TestNamespace,
				},
				Data: fileMap,
			},
		},
	}

	central := buildFakeCentral(testCase)
	client := buildFakeClient(t, testCase, central)
	run := &createCentralTLSExtensionRun{
		SecretReconciliator: commonExtensions.NewSecretReconciliator(client, client, central),
		centralObj:          central,
		currentTime:         time.Now(),
	}

	err = run.Execute(context.Background())
	require.NoError(t, err)

	secret := &coreV1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: testutils.TestNamespace, Name: "central-tls"}
	err = client.Get(context.Background(), key, secret)
	require.NoError(t, err)

	// Verify secondary CA exists and is the old primary CA
	secondaryCA, err := certgen.LoadSecondaryCAFromFileMap(secret.Data)
	require.NoError(t, err, "secondary CA should exist after AddSecondaryAndPromote")
	require.Equal(t, oldCAPEM.CertPEM(), secondaryCA.CertPEM(), "secondary CA should be the old primary CA")
	require.Equal(t, oldCAPEM.KeyPEM(), secondaryCA.KeyPEM(), "secondary CA key should be the old primary CA key")

	// Verify primary CA has changed
	newPrimaryCA, err := certgen.LoadCAFromFileMap(secret.Data)
	require.NoError(t, err, "primary CA should exist")
	require.NotEqual(t, oldCAPEM.CertPEM(), newPrimaryCA.CertPEM(), "primary CA should have changed to a new CA")
	require.NotEqual(t, oldCAPEM.KeyPEM(), newPrimaryCA.KeyPEM(), "primary CA key should have changed to a new CA key")
}

func verifyCentralCertNoSecondaryCA(t *testing.T, fileMap types.SecretDataMap, atTime *time.Time) {
	t.Helper()
	verifyCentralCert(t, fileMap, atTime)
	_, err := certgen.LoadSecondaryCAFromFileMap(fileMap)
	require.Error(t, err)
}

func verifyCentralCertWithSecondaryCA(t *testing.T, fileMap types.SecretDataMap, atTime *time.Time) {
	t.Helper()
	verifyCentralCert(t, fileMap, atTime)
	_, err := certgen.LoadSecondaryCAFromFileMap(fileMap)
	require.NoError(t, err)
}

func updateSecondaryCANotBefore(t *testing.T, client ctrlClient.Client, currentTime time.Time) {
	t.Helper()

	secret := &coreV1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: testutils.TestNamespace, Name: "central-tls"}
	err := utils.GetWithFallbackToUncached(context.Background(), client, client, key, secret)
	require.NoError(t, err)

	secondaryCA, err := certgen.LoadSecondaryCAFromFileMap(secret.Data)
	require.NoError(t, err)

	privateKey, err := helpers.ParsePrivateKeyPEM(secondaryCA.KeyPEM())
	require.NoError(t, err)

	updatedCert, err := updateCertificateNotBefore(secondaryCA.Certificate(), privateKey, currentTime)
	require.NoError(t, err)

	updatedCA, err := mtls.LoadCAForSigning(updatedCert, secondaryCA.KeyPEM())
	require.NoError(t, err)

	certgen.AddSecondaryCAToFileMap(secret.Data, updatedCA)
	err = client.Update(context.Background(), secret)
	require.NoError(t, err)
}

// updateCertificateNotBefore copies a CA certificate, updates the NotBefore and NotAfter fields,
// and then re-signs the certificate.
func updateCertificateNotBefore(ca *x509.Certificate, priv crypto.Signer, notBefore time.Time) (cert []byte, err error) {
	copy, err := x509.ParseCertificate(ca.Raw)
	if err != nil {
		return
	}

	validityDuration := ca.NotAfter.Sub(ca.NotBefore)
	copy.NotBefore = notBefore.Add(-5 * time.Minute)
	copy.NotAfter = copy.NotBefore.Add(validityDuration)
	cert, err = x509.CreateCertificate(rand.Reader, copy, copy, priv.Public(), priv)
	if err != nil {
		return
	}

	cert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
	return
}
