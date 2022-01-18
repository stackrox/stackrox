package authproviders

import (
	"net/http"
	"net/url"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil"
)

// Some common user identity attribute keys.
const (
	GroupsAttribute = "groups"
	EmailAttribute  = "email"
	UseridAttribute = "userid"
	NameAttribute   = "name"
)

// AllUIEndpoints returns all UI endpoints for a given auth provider, with the default UI endpoint first.
func AllUIEndpoints(providerProto *storage.AuthProvider) []string {
	if providerProto.GetUiEndpoint() == "" {
		return nil
	}
	return append([]string{providerProto.GetUiEndpoint()}, providerProto.GetExtraUiEndpoints()...)
}

// ExtractURLValuesFromRequest extracts url.Values from GET and POST requests.
func ExtractURLValuesFromRequest(r *http.Request) (url.Values, error) {
	switch r.Method {
	case http.MethodGet:
		return r.URL.Query(), nil
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			return nil, httputil.Errorf(http.StatusBadRequest, "could not parse form data: %v", err)
		}
		return r.Form, nil
	default:
		return nil, httputil.Errorf(http.StatusMethodNotAllowed, "method %s is not supported for this URL", r.Method)
	}
}
