package authproviders

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/httputil"
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

func (r *storeBackedRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !strings.HasPrefix(req.URL.Path, r.urlPathPrefix) {
		log.Errorf("UNEXPECTED: received HTTP request for invalid URL %v", req.URL)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	parts := strings.SplitN(req.URL.Path[len(r.urlPathPrefix):], "/", 2)
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
