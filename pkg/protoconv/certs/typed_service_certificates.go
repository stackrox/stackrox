package protoconv

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/services"
	"github.com/stackrox/rox/pkg/utils"
)

// ConvertTypedServiceCertificateSetToFileMap...
func ConvertTypedServiceCertificateSetToFileMap(certSet *storage.TypedServiceCertificateSet) map[string]string {
	fileMap := make(map[string]string)
	for _, cert := range certSet.ServiceCerts {
		serviceName := services.ServiceTypeToSlugName(cert.ServiceType)
		if serviceName == "" {
			utils.Should(errors.Errorf("invalid service type %v when creating certificate bundle to file map", cert.ServiceType))
			continue // ignore
		}
		fileMap[serviceName+"-cert.pem"] = string(cert.Cert.CertPem)
		fileMap[serviceName+"-key.pem"] = string(cert.Cert.KeyPem)
	}
	return fileMap
}
