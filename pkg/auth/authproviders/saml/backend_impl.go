package saml

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	saml2 "github.com/russellhaering/gosaml2"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/idputil"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/stringutils"
)

// All configuration keys the auth provider exposes within the auth provider config map.
const (
	SpIssuerConfigKey        = "sp_issuer"
	IDPMetadataURLConfigKey  = "idp_metadata_url"
	IDPIssuerConfigKey       = "idp_issuer"
	IDPCertPemConfigKey      = "idp_cert_pem"
	IDPSSOUrlConfigKey       = "idp_sso_url"
	IDPNameIDFormatConfigKey = "idp_nameid_format"
)

type backendImpl struct {
	factory    *factory
	acsURLPath string
	sp         saml2.SAMLServiceProvider
	id         string

	config map[string]string
}

func (p *backendImpl) OnEnable(_ authproviders.Provider) {
	p.factory.RegisterBackend(p)
}

func (p *backendImpl) OnDisable(_ authproviders.Provider) {
	p.factory.UnregisterBackend(p)
}

func (p *backendImpl) loginURL(clientState string) (string, error) {
	doc, err := p.sp.BuildAuthRequestDocument()
	if err != nil {
		return "", errors.Wrap(err, "could not construct auth request")
	}
	authURL, err := p.sp.BuildAuthURLRedirect(idputil.MakeState(p.id, clientState), doc)
	if err != nil {
		return "", errors.Wrap(err, "could not construct auth URL")
	}
	return authURL, nil
}

func newBackend(ctx context.Context, acsURLPath string, id string, uiEndpoints []string, config map[string]string) (*backendImpl, error) {
	if len(uiEndpoints) != 1 {
		return nil, errors.New("SAML requires exactly one UI endpoint")
	}
	p := &backendImpl{
		acsURLPath: acsURLPath,
		id:         id,
	}

	acsURL := &url.URL{
		Scheme: "https",
		Host:   uiEndpoints[0],
		Path:   acsURLPath,
	}
	p.sp.AssertionConsumerServiceURL = acsURL.String()

	spIssuer := config[SpIssuerConfigKey]
	if spIssuer == "" {
		return nil, errors.New("no ServiceProvider issuer specified")
	}
	p.sp.ServiceProviderIssuer = spIssuer

	effectiveConfig := map[string]string{
		SpIssuerConfigKey: spIssuer,
	}

	if config[IDPMetadataURLConfigKey] != "" {
		if !stringutils.AllEmpty(config[IDPIssuerConfigKey], config[IDPCertPemConfigKey], config[IDPSSOUrlConfigKey], config[IDPNameIDFormatConfigKey]) {
			return nil, errors.New("if IdP metadata URL is set, IdP issuer, SSO URL, certificate data and Name/ID format must be left blank")
		}
		if err := configureIDPFromMetadataURL(ctx, &p.sp, config[IDPMetadataURLConfigKey]); err != nil {
			return nil, errors.Wrap(err, "could not configure auth provider from IdP metadata URL")
		}
		effectiveConfig[IDPMetadataURLConfigKey] = config[IDPMetadataURLConfigKey]
	} else {
		if !stringutils.AllNotEmpty(config[IDPIssuerConfigKey], config[IDPSSOUrlConfigKey], config[IDPCertPemConfigKey]) {
			return nil, errors.New("if IdP metadata URL is not set, IdP issuer, SSO URL, and certificate data must be specified")
		}
		if err := configureIDPFromSettings(&p.sp, config[IDPIssuerConfigKey], config[IDPSSOUrlConfigKey], config[IDPCertPemConfigKey], config[IDPNameIDFormatConfigKey]); err != nil {
			return nil, errors.Wrap(err, "could not configure auth provider from settings")
		}
		effectiveConfig[IDPIssuerConfigKey] = config[IDPIssuerConfigKey]
		effectiveConfig[IDPSSOUrlConfigKey] = config[IDPSSOUrlConfigKey]
		effectiveConfig[IDPCertPemConfigKey] = config[IDPCertPemConfigKey]
		effectiveConfig[IDPNameIDFormatConfigKey] = config[IDPNameIDFormatConfigKey]
	}

	p.config = effectiveConfig

	return p, nil
}

func (p *backendImpl) Config() map[string]string {
	return p.config
}

func (p *backendImpl) consumeSAMLResponse(samlResponse string) (*authproviders.AuthResponse, error) {
	ai, err := p.sp.RetrieveAssertionInfo(samlResponse)
	if err != nil {
		return nil, errors.Wrap(err, "error in saml response")
	}

	var expiry time.Time
	if ai.SessionNotOnOrAfter != nil {
		expiry = *ai.SessionNotOnOrAfter
	}

	claim := saml2AssertionInfoToExternalClaim(ai)
	return &authproviders.AuthResponse{
		Claims:     claim,
		Expiration: expiry,
	}, nil
}

func (p *backendImpl) ProcessHTTPRequest(_ http.ResponseWriter, r *http.Request) (*authproviders.AuthResponse, error) {
	if r.URL.Path != p.acsURLPath {
		return nil, httputil.NewError(http.StatusNotFound, "Not Found")
	}
	if r.Method != http.MethodPost {
		return nil, httputil.NewError(http.StatusMethodNotAllowed, "Method Not Allowed")
	}

	samlResponse := r.FormValue("SAMLResponse")
	if samlResponse == "" {
		return nil, httputil.NewError(http.StatusBadRequest, "no SAML response transmitted")
	}

	return p.consumeSAMLResponse(samlResponse)
}

func (p *backendImpl) ExchangeToken(_ context.Context, _, _ string) (*authproviders.AuthResponse, string, error) {
	return nil, "", errors.New("not implemented")
}

func (p *backendImpl) RefreshURL() string {
	return ""
}

func (p *backendImpl) LoginURL(clientState string, _ *requestinfo.RequestInfo) (string, error) {
	return p.loginURL(clientState)
}

func (p *backendImpl) Validate(_ context.Context, _ *tokens.Claims) error {
	return nil
}

// Helpers
//////////

func getAttribute(assertionInfo *saml2.AssertionInfo, keys ...string) string {
	for _, key := range keys {
		if val := assertionInfo.Values.Get(key); val != "" {
			return val
		}
	}
	return ""
}

var emailAttributes = []string{
	"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
	"urn:oid:0.9.2342.19200300.100.1.3",
	"email",
	"Email",
	"emailaddress",
	"mail",
}

var fullNameAttributes = []string{
	"http://schemas.microsoft.com/identity/claims/displayname",
	"urn:oid:2.16.840.1.113730.3.1.241",
	"displayName",
	"urn:oid:2.5.4.3",
	"commonName",
}

var givenNameAttributes = []string{
	"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname",
	"urn:oid:2.5.4.42",
	"givenName",
}

var surnameAttributes = []string{
	"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname",
	"urn:oid:2.5.4.4",
	"surname",
}

func saml2AssertionInfoToExternalClaim(assertionInfo *saml2.AssertionInfo) *tokens.ExternalUserClaim {
	claim := &tokens.ExternalUserClaim{
		UserID: assertionInfo.NameID,
		Email:  getAttribute(assertionInfo, emailAttributes...),
		FullName: stringutils.FirstNonEmpty(
			getAttribute(assertionInfo, fullNameAttributes...),
			stringutils.JoinNonEmpty(
				" ",
				getAttribute(assertionInfo, givenNameAttributes...),
				getAttribute(assertionInfo, surnameAttributes...))),
	}
	claim.Attributes = make(map[string][]string)
	claim.Attributes[authproviders.UseridAttribute] = []string{claim.UserID}

	// We store claims as both friendly name and name for easy of use.
	for _, value := range assertionInfo.Values {
		for _, innerValue := range value.Values {
			if value.Name != "" {
				claim.Attributes[value.Name] = append(claim.Attributes[value.Name], innerValue.Value)
			}
			if value.FriendlyName != "" {
				claim.Attributes[value.FriendlyName] = append(claim.Attributes[value.FriendlyName], innerValue.Value)
			}
		}
	}
	return claim
}
