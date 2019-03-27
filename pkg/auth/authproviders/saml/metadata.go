package saml

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"time"

	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"
	dsig "github.com/russellhaering/goxmldsig"
)

func configureIDPFromMetadataURL(ctx context.Context, sp *saml2.SAMLServiceProvider, metadataURL string) error {
	entityID, descriptor, err := fetchIDPMetadata(ctx, metadataURL)
	if err != nil {
		return fmt.Errorf("fetching IdP metadata: %v", err)
	}
	sp.IdentityProviderIssuer = entityID
	return configureIDPFromDescriptor(sp, descriptor)
}

func fetchIDPMetadata(ctx context.Context, url string) (string, *types.IDPSSODescriptor, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", nil, fmt.Errorf("could not create HTTP request: %v", err)
	}
	resp, err := http.DefaultClient.Do(request.WithContext(ctx))
	if err != nil {
		return "", nil, fmt.Errorf("fetching metadata: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	var desc types.EntityDescriptor
	if err := xml.NewDecoder(resp.Body).Decode(&desc); err != nil {
		return "", nil, fmt.Errorf("parsing metadata XML: %v", err)
	}
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
				return fmt.Errorf("could not decode X.509 certificate data: %v", err)
			}
			parsedCert, err := x509.ParseCertificate(rawData)
			if err != nil {
				return fmt.Errorf("could not parse X.509 certificate: %v", err)
			}
			certStore.Roots = append(certStore.Roots, parsedCert)
		}
	}
	sp.IDPCertificateStore = certStore
	return nil
}
