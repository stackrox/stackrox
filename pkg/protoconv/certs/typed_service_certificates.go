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

// ConvertTypedServiceCertificateSetToFileMap converts a TypedServiceCertificateSet into a map
// of the shape
//
//	{
//	   "ca-cert.pem": "<PEM encoded CA certificate>",
//	   "<service>-cert.pem": "<PEM encoded service certificate>",
//	   "<service>-key.pem": "<PEM encoded service key>",
//	   ...
//	}
//
// It returns error in case a service type contained in the input failed to be converted into
// its associated slug-name representation.
func ConvertTypedServiceCertificateSetToFileMap(certSet *storage.TypedServiceCertificateSet) (map[string]string, error) {
	serviceCerts := certSet.GetServiceCerts()
	caCert := certSet.GetCaPem()
	fileMap := make(map[string]string, 1+2*len(serviceCerts)) // 1 for CA cert, and key+cert for each service
	if caCert != nil {
		fileMap[caCertKey] = string(caCert)
	}
	for _, cert := range certSet.GetServiceCerts() {
		serviceName := services.ServiceTypeToSlugName(cert.ServiceType)
		if serviceName == "" {
			return nil, errors.Errorf("failed to obtain slug-name for service type %v", cert.ServiceType)
		}
		fileMap[serviceName+"-cert.pem"] = string(cert.Cert.CertPem)
		fileMap[serviceName+"-key.pem"] = string(cert.Cert.KeyPem)
	}
	return fileMap, nil
}

// ConvertFileMapToTypedServiceCertificateSet is the inverse for ConvertTypedServiceCertificateSetToFileMap.
// It converts a map of the form
//
//	{
//	   "ca-cert.pem": "<PEM encoded CA certificate>",
//	   "<service>-cert.pem": "<PEM encoded service certificate>",
//	   "<service>-key.pem": "<PEM encoded service key>",
//	   ...
//	}
//
// into a TypedServiceCertificateSet.
//
// It returns error in case the input map contains keys of unexpected shape or in case it was
// not possible to derive proper service types from the respective file name.
func ConvertFileMapToTypedServiceCertificateSet(fileMap map[string]string) (*storage.TypedServiceCertificateSet, error) {
	var caPem []byte
	if caCert := fileMap[caCertKey]; caCert != "" {
		caPem = []byte(caCert)
	}

	serviceCertMap := make(map[storage.ServiceType]*storage.ServiceCertificate)

	for fileName, pemData := range fileMap {
		if fileName == caCertKey {
			// We handle the CA special and don't process it as part of this loop.
			continue
		}
		var serviceSlugName string
		var certPem, keyPem []byte

		if strings.HasSuffix(fileName, "-cert.pem") {
			serviceSlugName = strings.TrimSuffix(fileName, "-cert.pem")
			certPem = []byte(pemData)
		} else if strings.HasSuffix(fileName, "-key.pem") {
			serviceSlugName = strings.TrimSuffix(fileName, "-key.pem")
			keyPem = []byte(pemData)
		} else {
			utils.Should(errors.Errorf("unexpected file name %q in file map while trying to convert to typed service certificate set", fileName))
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
