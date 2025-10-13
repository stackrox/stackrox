package certgen

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func checkCertAndKey(t *testing.T, fileMap map[string][]byte, expectedCert, expectedKey []byte, certFileName, keyFileName string) {
	t.Helper()
	if !bytes.Equal(fileMap[certFileName], expectedCert) {
		t.Errorf("Expected cert for %s to be %q, got %q", certFileName, expectedCert, fileMap[certFileName])
	}
	if !bytes.Equal(fileMap[keyFileName], expectedKey) {
		t.Errorf("Expected key for %s to be %q, got %q", keyFileName, expectedKey, fileMap[keyFileName])
	}
}

func TestGenerateCA(t *testing.T) {
	ca, err := GenerateCA()
	assert.NoError(t, err)

	certPEM := ca.CertPEM()
	block, _ := pem.Decode(certPEM)
	assert.NotNil(t, block)

	cert, err := x509.ParseCertificate(block.Bytes)
	assert.NoError(t, err)

	const fiveYears = 5 * 365 * 24 * time.Hour
	validity := cert.NotAfter.Sub(cert.NotBefore)
	assert.InDelta(t, fiveYears, validity, float64(time.Hour), "cert validity should be ~5 years")
}

func TestPromoteSecondaryCA(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	primaryCert := []byte("primary-cert")
	primaryKey := []byte("primary-key")
	secondaryCert := []byte("secondary-cert")
	secondaryKey := []byte("secondary-key")

	primaryCA := mocks.NewMockCA(ctrl)
	primaryCA.EXPECT().CertPEM().Return(primaryCert).AnyTimes()
	primaryCA.EXPECT().KeyPEM().Return(primaryKey).AnyTimes()

	secondaryCA := mocks.NewMockCA(ctrl)
	secondaryCA.EXPECT().CertPEM().Return(secondaryCert).AnyTimes()
	secondaryCA.EXPECT().KeyPEM().Return(secondaryKey).AnyTimes()

	fileMap := make(map[string][]byte)
	AddCAToFileMap(fileMap, primaryCA)
	AddSecondaryCAToFileMap(fileMap, secondaryCA)

	checkCertAndKey(t, fileMap, primaryCert, primaryKey, mtls.CACertFileName, mtls.CAKeyFileName)
	checkCertAndKey(t, fileMap, secondaryCert, secondaryKey, mtls.SecondaryCACertFileName, mtls.SecondaryCAKeyFileName)

	PromoteSecondaryCA(fileMap)

	checkCertAndKey(t, fileMap, secondaryCert, secondaryKey, mtls.CACertFileName, mtls.CAKeyFileName)
	checkCertAndKey(t, fileMap, primaryCert, primaryKey, mtls.SecondaryCACertFileName, mtls.SecondaryCAKeyFileName)

	RemoveSecondaryCA(fileMap)

	checkCertAndKey(t, fileMap, secondaryCert, secondaryKey, mtls.CACertFileName, mtls.CAKeyFileName)
	checkCertAndKey(t, fileMap, nil, nil, mtls.SecondaryCACertFileName, mtls.SecondaryCAKeyFileName)
}
