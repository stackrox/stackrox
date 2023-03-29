package validation

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	errAdditionalCANameRequired         = "must specify a name for the additional CA"
	errAdditionalCAContentRequired      = "must specify a certificate for the additional CA"
	errAdditionalCAContentNoCertFound   = "no certificates found in the provided CA cert content"
	errAdditionalCAContentParsingFailed = "failed to parse the provided CA cert content"
	errFailedToVerifyCertificateChain   = "failed to verify the certificate chain"
)

// ValidateCentral validates the Central CR.
//
// This method will return a useful/meaningful error message for
// each field that does not pass validation, as well as the field path, so the user can easily
// identify the source of the error.
//
// Currently, this only validates the TLS.additionalCAs field, but is intended to be
// expanded to validate the entire CR.
//
// See: TODO(ROX-7683)
func ValidateCentral(central *v1alpha1.Central) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, validateCentralSpec(field.NewPath("spec"), central.Spec)...)
	return errs
}

func validateCentralSpec(path *field.Path, spec v1alpha1.CentralSpec) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, validateCentralTLSConfig(path.Child("tls"), spec.TLS)...)
	return errs
}

func validateCentralTLSConfig(path *field.Path, tlsConfig *v1alpha1.TLSConfig) field.ErrorList {
	var errs field.ErrorList
	if tlsConfig == nil {
		return nil
	}
	errs = append(errs, validateCentralTLSConfigAdditionalCAs(path.Child("additionalCAs"), tlsConfig.AdditionalCAs)...)
	return errs
}

func validateCentralTLSConfigAdditionalCAs(path *field.Path, additionalCAs []v1alpha1.AdditionalCA) field.ErrorList {
	var errs field.ErrorList
	var seenAdditionalCANames = sets.NewString()
	for i, additionalCA := range additionalCAs {
		itemPath := path.Index(i)
		if seenAdditionalCANames.Has(additionalCA.Name) {
			errs = append(errs, field.Duplicate(itemPath.Child("name"), additionalCA.Name))
		}
		seenAdditionalCANames.Insert(additionalCA.Name)
		errs = append(errs, validateCentralTLSConfigAdditionalCA(itemPath, additionalCA)...)
	}
	return errs
}

func validateCentralTLSConfigAdditionalCA(path *field.Path, additionalCA v1alpha1.AdditionalCA) field.ErrorList {
	var errs field.ErrorList
	if additionalCA.Name == "" {
		errs = append(errs, field.Required(path.Child("name"), errAdditionalCANameRequired))
	}
	errs = append(errs, validateCentralTLSConfigAdditionalCACertificate(path.Child("content"), additionalCA.Content)...)
	return errs
}

func validateCentralTLSConfigAdditionalCACertificate(path *field.Path, content string) field.ErrorList {
	var errs field.ErrorList
	if len(content) == 0 {
		errs = append(errs, field.Required(path, errAdditionalCAContentRequired))
		return errs
	}
	certificates, err := parseCertificateChainFromPEM([]byte(content))
	if err != nil {
		errs = append(errs, field.Invalid(path, content, fmt.Sprintf("%s: %s", errAdditionalCAContentParsingFailed, err.Error())))
		return errs
	}
	if len(certificates) == 0 {
		errs = append(errs, field.Invalid(path, content, errAdditionalCAContentNoCertFound))
		return errs
	}

	if err := validateCertificateChain(certificates); err != nil {
		errs = append(errs, field.Invalid(path, content, err.Error()))
		return errs
	}

	return errs
}

func validateCertificateChain(certificates []*x509.Certificate) error {
	if len(certificates) == 1 {
		// No need to validate the certificate chain when there is only one certificate
		return nil
	}
	rootCertificate := certificates[len(certificates)-1]
	certPool := x509.NewCertPool()
	certPool.AddCert(rootCertificate)

	var intermediates = x509.NewCertPool()
	for i := len(certificates) - 1; i >= 0; i-- {
		position := prettyPosition(i + 1)
		intermediateCertificate := certificates[i]
		opts := x509.VerifyOptions{
			Roots:         certPool,
			Intermediates: intermediates,
		}
		if _, err := intermediateCertificate.Verify(opts); err != nil {
			return fmt.Errorf("%s: %s: %s",
				errFailedToVerifyCertificateChain,
				"could not verify the "+position+" certificate in the chain",
				err.Error(),
			)
		}
		intermediates.AddCert(certificates[i])
	}

	return nil
}

func parseCertificateChainFromPEM(data []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	var i = 0
	for {

		if len(data) == 0 {
			break
		}

		position := prettyPosition(i)

		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			return nil, fmt.Errorf(fmt.Sprintf("failed to parse %s certificate in chain: no PEM data found", position))
		}
		if block.Type != "CERTIFICATE" {
			return nil, fmt.Errorf("failed to parse %s certificate in chain: unexpected PEM type '%s'", position, block.Type)
		}
		if len(block.Headers) != 0 {
			return nil, fmt.Errorf("failed to parse %s certificate in chain: unexpected PEM optional headers", position)
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s certificate in chain: %s", position, err.Error())
		}
		certs = append(certs, cert)
		i++
	}
	return certs, nil
}

func prettyPosition(i int) string {
	var position string
	if i == 0 {
		position = "1st"
	} else if i == 1 {
		position = "2nd"
	} else if i == 2 {
		position = "3rd"
	} else {
		position = fmt.Sprintf("%dth", i+1)
	}
	return position
}
