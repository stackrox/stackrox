package protoconv

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/services"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	caCertKey = "ca-cert.pem"
)

// ConvertTypedServiceCertificateSetToFileMap ...
func ConvertTypedServiceCertificateSetToFileMap(certSet *storage.TypedServiceCertificateSet) map[string]string {
	serviceCerts := certSet.GetServiceCerts()
	caCert := certSet.GetCaPem()
	fileMap := make(map[string]string, 1+2*len(serviceCerts)) // 1 for CA cert, and key+cert for each service
	if caCert != nil {
		fileMap[caCertKey] = string(caCert)
	}
	for _, cert := range certSet.GetServiceCerts() {
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

// ConvertFileMapToTypedServiceCertificateSet ...
func ConvertFileMapToTypedServiceCertificateSet(fileMap map[string]string) *storage.TypedServiceCertificateSet {
	var caPem []byte
	if caCert := fileMap[caCertKey]; caCert != "" {
		caPem = []byte(caCert)
	}
	delete(fileMap, caCertKey)

	serviceCertMap := make(map[storage.ServiceType]*storage.ServiceCertificate)

	for fileName, pemData := range fileMap {
		var serviceSlugName string
		var certPem, keyPem []byte

		if strings.HasSuffix(fileName, "-cert.pem") {
			serviceSlugName = strings.TrimSuffix(fileName, "-cert.pem")
			certPem = []byte(pemData)
		} else if strings.HasSuffix(fileName, "-key.pem") {
			serviceSlugName = strings.TrimSuffix(fileName, "-key.pem")
			keyPem = []byte(pemData)
		} else {
			// TODO?
			continue
		}
		serviceType := services.SlugNameToServiceType(serviceSlugName)
		if serviceCertMap[serviceType] == nil {
			serviceCertMap[serviceType] = &storage.ServiceCertificate{}
		}

		if certPem != nil {
			serviceCertMap[serviceType].CertPem = certPem
		}
		if keyPem != nil {
			serviceCertMap[serviceType].KeyPem = keyPem
		}
	}

	typedServiceCerts := make([]*storage.TypedServiceCertificate, 0, len(serviceCertMap))
	for serviceType, serviceCert := range serviceCertMap {
		typedServiceCerts = append(typedServiceCerts, &storage.TypedServiceCertificate{
			ServiceType: serviceType,
			Cert:        serviceCert,
		})
	}

	certSet := storage.TypedServiceCertificateSet{
		CaPem:        caPem,
		ServiceCerts: typedServiceCerts,
	}

	return &certSet
}
