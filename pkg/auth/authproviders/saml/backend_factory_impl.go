package saml

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	saml2 "github.com/russellhaering/gosaml2"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/idputil"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	acsRelativePath = "acs"

	// TypeName is the standard type name for SAML auth providers
	TypeName = "saml"
)

var _ authproviders.BackendFactory = (*factory)(nil)

type factory struct {
	urlPathPrefix string

	backendsByIssuer      map[string]map[*backendImpl]struct{}
	backendsByIssuerMutex sync.Mutex
}

// NewFactory creates a new SAML auth provider factory.
func NewFactory(urlPathPrefix string) authproviders.BackendFactory {
	urlPathPrefix = strings.TrimRight(urlPathPrefix, "/") + "/"
	f := &factory{
		urlPathPrefix:    urlPathPrefix,
		backendsByIssuer: make(map[string]map[*backendImpl]struct{}),
	}

	return f
}

func (f *factory) CreateBackend(ctx context.Context, id string, uiEndpoints []string, config map[string]string, _ map[string]string) (authproviders.Backend, error) {
	be, err := newBackend(ctx, f.urlPathPrefix+acsRelativePath, id, uiEndpoints, config)
	if err != nil {
		return nil, err
	}
	be.factory = f
	return be, nil
}

func (f *factory) processACSRequest(r *http.Request) (string, error) {
	if r.Method != http.MethodPost {
		return "", httputil.NewError(http.StatusMethodNotAllowed, "only POST requests are allowed to this URL")
	}
	if err := r.ParseForm(); err != nil {
		return "", httputil.Errorf(http.StatusBadRequest, "could not parse form data: %v", err)
	}

	state := r.FormValue("RelayState")
	providerID, _ := idputil.SplitState(state)
	if providerID != "" {
		// Preferred option: specified via relay state
		return providerID, nil
	}
	return f.autoRouteACSRequest(r)
}

func (f *factory) getBackendsByIssuer(issuerName string) []*backendImpl {
	f.backendsByIssuerMutex.Lock()
	defer f.backendsByIssuerMutex.Unlock()

	backendSet := f.backendsByIssuer[issuerName]
	if len(backendSet) == 0 {
		return nil
	}

	backendSlice := make([]*backendImpl, 0, len(backendSet))
	for be := range backendSet {
		backendSlice = append(backendSlice, be)
	}
	return backendSlice
}

func (f *factory) RegisterBackend(be *backendImpl) {
	issuerName := be.sp.IdentityProviderIssuer

	f.backendsByIssuerMutex.Lock()
	defer f.backendsByIssuerMutex.Unlock()
	beSet := f.backendsByIssuer[issuerName]
	if beSet == nil {
		beSet = make(map[*backendImpl]struct{})
		f.backendsByIssuer[issuerName] = beSet
	}
	beSet[be] = struct{}{}
}

func (f *factory) UnregisterBackend(be *backendImpl) {
	issuerName := be.sp.IdentityProviderIssuer

	f.backendsByIssuerMutex.Lock()
	defer f.backendsByIssuerMutex.Unlock()
	beSet := f.backendsByIssuer[issuerName]
	if beSet == nil {
		return
	}
	delete(beSet, be)
	if len(beSet) == 0 {
		delete(f.backendsByIssuer, issuerName)
	}
}

func (f *factory) ProcessHTTPRequest(_ http.ResponseWriter, r *http.Request) (string, string, error) {
	if !strings.HasPrefix(r.URL.Path, f.urlPathPrefix) {
		return "", "", httputil.NewError(http.StatusInternalServerError, "received invalid request")
	}

	relayState := r.FormValue("RelayState")
	_, clientState := idputil.SplitState(relayState)

	relativePath := r.URL.Path[len(f.urlPathPrefix):]
	if relativePath == acsRelativePath {
		providerID, err := f.processACSRequest(r)
		return providerID, clientState, err
	}

	return "", clientState, httputil.NewError(http.StatusNotFound, "Not Found")
}

func (f *factory) ResolveProviderAndClientState(state string) (string, string, error) {
	providerID, clientState := idputil.SplitState(state)
	if providerID == "" {
		return "", clientState, fmt.Errorf("malformed state %q", state)
	}
	return providerID, clientState, nil
}

func (f *factory) autoRouteACSRequest(req *http.Request) (string, error) {
	// Heuristically try to deduce the target backend from (a) IdP issuer name and (b) referer/origin URL.
	// Note: heuristically is fine - signature checking still takes place, so even if we reroute this to the wrong
	// backend, there are no adverse security implications.
	resp, err := saml2.DecodeUnverifiedBaseResponse(req.FormValue("SAMLResponse"))
	if err != nil {
		return "", httputil.Errorf(http.StatusBadRequest, "no relay state specified, and not able to parse SAML response: %v", err)
	}
	issuerName := ""
	if resp.Issuer != nil {
		issuerName = resp.Issuer.Value
	}
	backends := f.getBackendsByIssuer(issuerName)
	if len(backends) > 1 {
		backends = filterBackendsByOrigin(req, backends)
	}

	switch l := len(backends); {
	case l == 0:
		return "", httputil.Errorf(http.StatusBadRequest, "no relay state specified, and no backend registered for IdP issuer %q", issuerName)
	case l == 1:
		return backends[0].id, nil
	default:
		return "", httputil.Errorf(http.StatusBadRequest,
			"Multiple active auth provider backends exist for IdP issuer %q.\n"+
				"Please set the `Default RelayState` field in your IdP config per the instructions on the configuration "+
				"page, or delete all but one auth provider for this issuer", issuerName)
	}
}

func (f *factory) RedactConfig(config map[string]string) map[string]string {
	return config
}

func (f *factory) MergeConfig(newCfg, _ map[string]string) map[string]string {
	return newCfg
}

func (f *factory) GetSuggestedAttributes() []string {
	return []string{authproviders.UseridAttribute}
}

func filterBackendsByOrigin(req *http.Request, backends []*backendImpl) []*backendImpl {
	var baseURLs []string

	// Try both referer and origin
	for _, ref := range []string{req.Referer(), req.Header.Get("Origin")} {
		if ref == "" {
			continue
		}

		if u, err := url.Parse(ref); err == nil {
			baseURL := (&url.URL{
				Scheme: stringutils.OrDefault(u.Scheme, "https"),
				Host:   u.Host,
				Path:   "/",
			}).String()
			baseURLs = append(baseURLs, baseURL)
		}
	}

	if len(baseURLs) == 0 {
		return backends
	}

	var filtered []*backendImpl
	for _, be := range backends {
		for _, baseURL := range baseURLs {
			if strings.HasPrefix(be.sp.IdentityProviderSSOURL, baseURL) {
				filtered = append(filtered, be)
				break
			}
		}
	}

	if len(filtered) > 0 {
		return filtered
	}
	return backends
}
