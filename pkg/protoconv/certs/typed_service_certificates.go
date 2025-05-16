package protoconv

import (
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/services"
	"github.com/stackrox/rox/pkg/set"
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
func ConvertTypedServiceCertificateSetToFileMap(certSet *storage.TypedServiceCertificateSet) (map[string][]byte, error) {
	caCert := certSet.GetCaPem()
	if len(caCert) == 0 {
		return nil, errors.New("no CA certificate in typed service certificate set")
	}
	serviceCerts := certSet.GetServiceCerts()
	if len(serviceCerts) == 0 {
		return nil, errors.New("no service certificates in typed service certificate set")
	}

	fileMap := make(map[string][]byte, 1+2*len(serviceCerts)) // 1 for CA cert, and key+cert for each service
	if caCert != nil {
		fileMap[caCertKey] = caCert
	}
	for _, cert := range serviceCerts {
		serviceName := services.ServiceTypeToSlugName(cert.ServiceType)
		if serviceName == "" {
			return nil, errors.Errorf("failed to obtain slug-name for service type %v", cert.ServiceType)
		}
		fileMap[serviceName+"-cert.pem"] = cert.Cert.CertPem
		fileMap[serviceName+"-key.pem"] = cert.Cert.KeyPem
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
func ConvertFileMapToTypedServiceCertificateSet(fileMap map[string][]byte) (*storage.TypedServiceCertificateSet, []string, error) {
	var unknownServices set.Set[string]

	ca, err := mtls.LoadCAForValidation(fileMap[caCertKey])
	if err != nil {
		return nil, nil, errors.New("invalid CA certificate in file map")
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
			certPem = pemData
		} else if strings.HasSuffix(fileName, "-key.pem") {
			serviceSlugName = strings.TrimSuffix(fileName, "-key.pem")
			keyPem = pemData
		} else {
			return nil, nil, errors.Errorf("unexpected file name %q in file map", fileName)
		}
		serviceType := services.SlugNameToServiceType(serviceSlugName)
		if serviceType == storage.ServiceType_UNKNOWN_SERVICE {
			if unknownServices == nil {
				unknownServices = make(set.Set[string], len(fileMap))
			}
			unknownServices.Add(serviceSlugName)
			continue
		}
		if serviceCertMap[serviceType] == nil {
			serviceCertMap[serviceType] = &storage.ServiceCertificate{}
		}

		if certPem != nil {
			serviceCertMap[serviceType].CertPem = certPem
		}
		if keyPem != nil {
			serviceCertMap[serviceType].KeyPem = keyPem
		}
		// When certificate and key have been retrieved from the file map, validate them against the CA.
		if len(serviceCertMap[serviceType].CertPem) > 0 && len(serviceCertMap[serviceType].KeyPem) > 0 {
			keyAndCert := map[string][]byte{
				mtls.ServiceCertFileName: serviceCertMap[serviceType].CertPem,
				mtls.ServiceKeyFileName:  serviceCertMap[serviceType].KeyPem,
			}
			err = certgen.VerifyServiceCertAndKey(keyAndCert, "", ca, serviceType, nil)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "verifying service certificate for service %s", serviceType.String())
			}
		}
	}

	var typedServiceCerts []*storage.TypedServiceCertificate
	if len(serviceCertMap) == 0 {
		// We are expecting a non-zero number of services in valid `TypedServiceCertificateSet`.
		return nil, nil, errors.New("no known service certificates in file map")
	}
	typedServiceCerts = make([]*storage.TypedServiceCertificate, 0, len(serviceCertMap))
	for serviceType, serviceCert := range serviceCertMap {
		if len(serviceCert.CertPem) == 0 {
			return nil, nil, errors.Errorf("missing certificate for service %s in file map", serviceType.String())
		}
		if len(serviceCert.KeyPem) == 0 {
			return nil, nil, errors.Errorf("missing key for service %s in file map", serviceType.String())
		}
		typedServiceCerts = append(typedServiceCerts, &storage.TypedServiceCertificate{
			ServiceType: serviceType,
			Cert:        serviceCert,
		})
	}

	certSet := storage.TypedServiceCertificateSet{
		CaPem:        fileMap[caCertKey],
		ServiceCerts: typedServiceCerts,
	}

	var unknownServicesSlice []string
	if unknownServices != nil {
		unknownServicesSlice = unknownServices.AsSlice()
		slices.Sort(unknownServicesSlice)
	}

	return &certSet, unknownServicesSlice, nil
}
