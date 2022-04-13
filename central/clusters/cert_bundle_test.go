package clusters

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/buildinfo"
	"github.com/stackrox/stackrox/pkg/mtls"
	"github.com/stretchr/testify/assert"
)

func TestCertBundle_FileMap_Nil(t *testing.T) {
	var bundle CertBundle
	fileMap := bundle.FileMap()
	assert.Empty(t, fileMap)
}

func TestCertBundle_FileMap_Empty(t *testing.T) {
	bundle := CertBundle{}
	fileMap := bundle.FileMap()
	assert.Empty(t, fileMap)
}

func TestCertBundle_FileMap(t *testing.T) {
	bundle := CertBundle{
		storage.ServiceType_SENSOR_SERVICE: &mtls.IssuedCert{
			CertPEM: []byte("sensor certificate"),
			KeyPEM:  []byte("sensor private key"),
		},
		storage.ServiceType_ADMISSION_CONTROL_SERVICE: &mtls.IssuedCert{
			CertPEM: []byte("admission control certificate"),
			KeyPEM:  []byte("admission control private key"),
		},
	}

	expectedFiles := map[string]string{
		"sensor-cert.pem":            "sensor certificate",
		"sensor-key.pem":             "sensor private key",
		"admission-control-cert.pem": "admission control certificate",
		"admission-control-key.pem":  "admission control private key",
	}

	assert.Equal(t, expectedFiles, bundle.FileMap())
}

func TestCertBundle_FileMap_WithInvalid(t *testing.T) {
	asserter := assert.Panics
	if buildinfo.ReleaseBuild {
		asserter = assert.NotPanics
	}
	asserter(t, func() {
		bundle := CertBundle{
			storage.ServiceType_SENSOR_SERVICE: &mtls.IssuedCert{
				CertPEM: []byte("sensor certificate"),
				KeyPEM:  []byte("sensor private key"),
			},
			storage.ServiceType_ADMISSION_CONTROL_SERVICE: &mtls.IssuedCert{
				CertPEM: []byte("admission control certificate"),
				KeyPEM:  []byte("admission control private key"),
			},
			storage.ServiceType(99): &mtls.IssuedCert{
				CertPEM: []byte("unknown service certificate"),
				KeyPEM:  []byte("unknown service private key"),
			},
		}

		expectedFiles := map[string]string{
			"sensor-cert.pem":            "sensor certificate",
			"sensor-key.pem":             "sensor private key",
			"admission-control-cert.pem": "admission control certificate",
			"admission-control-key.pem":  "admission control private key",
		}

		assert.Equal(t, expectedFiles, bundle.FileMap())
	})
}
