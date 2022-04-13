package clusters

import (
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/services"
	"github.com/stackrox/stackrox/pkg/utils"
)

// FileMap returns a map[string]string that maps individual file names for service certificates to their PEM-encoded
// contents.
// The file name is derived from the slug-case version of the service type, e.g., for the ADMISSION_CONTROL_SERVICE
// service type, the respective files are `admission-control-cert.pem` and `admission-control-key.pem`.
func (b CertBundle) FileMap() map[string]string {
	files := make(map[string]string, 2*len(b))
	for svcType, cert := range b {
		serviceName := services.ServiceTypeToSlugName(svcType)
		if serviceName == "" {
			utils.Should(errors.Errorf("invalid service type %v when creating certificate bundle to file map", svcType))
			continue // ignore
		}
		files[serviceName+"-cert.pem"] = string(cert.CertPEM)
		files[serviceName+"-key.pem"] = string(cert.KeyPEM)
	}

	return files
}
