package saml

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/stackrox/rox/pkg/stringutils"
)

func configureIDPFromMetadataURL(ctx context.Context, sp *saml2.SAMLServiceProvider, metadataURL string) error {
	entityID, descriptor, err := fetchIDPMetadata(ctx, metadataURL)
	if err != nil {
		return errors.Wrap(err, "fetching IdP metadata")
	}
	sp.IdentityProviderIssuer = entityID
	return configureIDPFromDescriptor(sp, descriptor)
}

func fetchIDPMetadata(ctx context.Context, url string) (string, *types.IDPSSODescriptor, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", nil, errors.Wrap(err, "could not create HTTP request")
	}

	httpClient := http.DefaultClient
	if stringutils.ConsumeSuffix(&request.URL.Scheme, "+insecure") {
		httpClient = insecureHTTPClient
	}

	resp, err := httpClient.Do(request.WithContext(ctx))
	if err != nil {
		return "", nil, errors.Wrap(err, "fetching metadata")
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	var descriptors entityDescriptors
	if err := xml.NewDecoder(resp.Body).Decode(&descriptors); err != nil {
		return "", nil, errors.Wrap(err, "parsing metadata XML")
	}
	if len(descriptors) != 1 {
		return "", nil, errors.Errorf("invalid number of entity descriptors in metadata response: expected exactly one, got %d", len(descriptors))
	}
	desc := descriptors[0]
	if desc.IDPSSODescriptor == nil {
		return "", nil, errors.New("metadata contains no IdP SSO descriptor")
	}
	if !desc.ValidUntil.IsZero() && !desc.ValidUntil.After(time.Now()) {
		return "", nil, fmt.Errorf("IdP metadata has expired at %v", desc.ValidUntil)
	}
	return desc.EntityID, desc.IDPSSODescriptor, nil
}

func configureIDPFromDescriptor(sp *saml2.SAMLServiceProvider, descriptor *types.IDPSSODescriptor) error {
	if descriptor.WantAuthnRequestsSigned {
		return errors.New("the IdP wants signed authentication requests, which are currently not supported")
	}

	var redirectLoginURL string
	for _, ssoService := range descriptor.SingleSignOnServices {
		if ssoService.Binding == saml2.BindingHttpRedirect {
			redirectLoginURL = ssoService.Location
			break
		}
	}
	if redirectLoginURL == "" {
		return errors.New("could not determine location for the HTTP-Redirect binding")
	}
	sp.IdentityProviderSSOURL = redirectLoginURL
	certStore := &dsig.MemoryX509CertificateStore{}
	for _, keyDesc := range descriptor.KeyDescriptors {
		if keyDesc.Use != "signing" {
			continue
		}
		for _, cert := range keyDesc.KeyInfo.X509Data.X509Certificates {
			rawData, err := base64.StdEncoding.DecodeString(cert.Data)
			if err != nil {
				return errors.Wrap(err, "could not decode X.509 certificate data")
			}
			parsedCert, err := x509.ParseCertificate(rawData)
			if err != nil {
				return errors.Wrap(err, "could not parse X.509 certificate")
			}
			certStore.Roots = append(certStore.Roots, parsedCert)
		}
	}
	sp.IDPCertificateStore = certStore
	return nil
}
