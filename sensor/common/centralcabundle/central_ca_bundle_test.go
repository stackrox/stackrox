package centralcabundle

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetAndGet(t *testing.T) {
	Set(nil)

	result := Get()
	assert.Empty(t, result)

	certs := []*x509.Certificate{
		{Subject: pkix.Name{CommonName: "CA1"}},
		{Subject: pkix.Name{CommonName: "CA2"}},
	}
	Set(certs)

	result = Get()
	assert.Len(t, result, 2)
	assert.Equal(t, "CA1", result[0].Subject.CommonName)
	assert.Equal(t, "CA2", result[1].Subject.CommonName)
}

func TestSetNilClearsStore(t *testing.T) {
	Set([]*x509.Certificate{
		{Subject: pkix.Name{CommonName: "CA1"}},
	})

	assert.Len(t, Get(), 1)
	Set(nil)

	assert.Empty(t, Get())
}
