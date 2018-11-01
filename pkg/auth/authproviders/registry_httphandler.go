package authproviders

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/httputil"
)

const (
	providersPath = "providers"
	loginPath     = "login"
)

func (r *storeBackedRegistry) URLPathPrefix() string {
	return r.urlPathPrefix
}

func (r *storeBackedRegistry) errorURL(err error, typ string, clientState string) *url.URL {
	return &url.URL{
		Path: r.redirectURL,
		Fragment: url.Values{
			"error": {err.Error()},
			"type":  {typ},
			"state": {clientState},
		}.Encode(),
	}
}

func (r *storeBackedRegistry) tokenURL(rawToken string, typ string, clientState string) *url.URL {
	return &url.URL{
		Path: r.redirectURL,
		Fragment: url.Values{
			"token": {rawToken},
			"type":  {typ},
			"state": {clientState},
		}.Encode(),
	}
}

func (r *storeBackedRegistry) providersURLPrefix() string {
	return path.Join(r.urlPathPrefix, providersPath) + "/"
}

func (r *storeBackedRegistry) loginURLPrefix() string {
	return path.Join(r.urlPathPrefix, loginPath) + "/"
}

func (r *storeBackedRegistry) initHTTPMux() {
	r.HandleFunc(r.providersURLPrefix(), r.providersHTTPHandler)
	r.HandleFunc(r.loginURLPrefix(), r.loginHTTPHandler)
}

func (r *storeBackedRegistry) loginHTTPHandler(w http.ResponseWriter, req *http.Request) {
	prefix := r.loginURLPrefix()
	if !strings.HasPrefix(req.URL.Path, prefix) {
		log.Errorf("UNEXPECTED: received HTTP request for invalid URL %v", req.URL)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	providerID := req.URL.Path[len(prefix):]
	clientState := req.URL.Query().Get("clientState")

	provider := r.getAuthProvider(providerID)
	if provider == nil {
		http.Error(w, fmt.Sprintf("Unknown auth provider ID %q", providerID), http.StatusNotFound)
		return
	}

	loginURL := provider.Backend().LoginURL(clientState)
	if loginURL == "" {
		http.Error(w, "could not get login URL", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", loginURL)
	w.WriteHeader(http.StatusSeeOther)
}

func (r *storeBackedRegistry) loginURL(providerID string) string {
	return path.Join(r.loginURLPrefix(), providerID)
}

func (r *storeBackedRegistry) providersHTTPHandler(w http.ResponseWriter, req *http.Request) {
	prefix := r.providersURLPrefix()
	if !strings.HasPrefix(req.URL.Path, prefix) {
		log.Errorf("UNEXPECTED: received HTTP request for invalid URL %v", req.URL)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	relativePath := req.URL.Path[len(prefix):]
	parts := strings.SplitN(relativePath, "/", 2)
	if len(parts) == 0 {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	typ := parts[0]

	factory := r.getFactory(typ)
	if factory == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	providerID, err := factory.ProcessHTTPRequest(w, req)
	var provider *authProvider
	if err == nil {
		provider = r.getAuthProvider(providerID)
		if provider == nil {
			err = fmt.Errorf("invalid auth provider ID %q", providerID)
		} else if provider.Type() != parts[0] {
			err = fmt.Errorf("auth provider %s is of invalid type %s", provider.Name(), provider.Type())
		}
	}
	if err != nil {
		if httpErr, ok := err.(httputil.HTTPError); ok {
			http.Error(w, httpErr.Error(), httpErr.HTTPStatusCode())
			return
		}
		w.Header().Set("Location", r.errorURL(err, typ, "").String())
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	claim, opts, clientState, err := provider.Backend().ProcessHTTPRequest(w, req)
	var tokenInfo *tokens.TokenInfo

	if err == nil && claim != nil {
		tokenInfo, err = provider.issuer.Issue(tokens.RoxClaims{ExternalUser: claim}, opts...)
	}

	if err != nil {
		if httpErr, ok := err.(httputil.HTTPError); ok {
			http.Error(w, httpErr.Error(), httpErr.HTTPStatusCode())
			return
		}
		w.Header().Set("Location", r.errorURL(err, typ, clientState).String())
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	if tokenInfo == nil {
		// Assume the ProcessHTTPRequest already took care of writing a response.
		return
	}

	w.Header().Set("Location", r.tokenURL(tokenInfo.Token, typ, clientState).String())
	w.WriteHeader(http.StatusSeeOther)
}
