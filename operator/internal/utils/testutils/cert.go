package testutils

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func GenerateTestCertWithValidity(t *testing.T, notBeforeStr, notAfterStr string) *x509.Certificate {
	t.Helper()
	if notBeforeStr == "" && notAfterStr == "" {
		return nil
	}
	notBefore, err := time.Parse(time.RFC3339, notBeforeStr)
	require.NoError(t, err)
	notAfter, err := time.Parse(time.RFC3339, notAfterStr)
	require.NoError(t, err)
	return &x509.Certificate{
		NotBefore: notBefore,
		NotAfter:  notAfter,
	}
}
