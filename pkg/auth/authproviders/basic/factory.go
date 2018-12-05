package basic

import (
	"context"
	"net/http"
	"strings"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// TypeName is the standard type name for basic auth provider.
	TypeName = "basic"
)

var (
	log = logging.LoggerForModule()
)

type factory struct {
	urlPathPrefix string
}

// NewFactory creates a new factory for Basic authprovider backends.
func NewFactory(urlPathPrefix string) authproviders.BackendFactory {
	urlPathPrefix = strings.TrimRight(urlPathPrefix, "/") + "/"
	return &factory{
		urlPathPrefix: urlPathPrefix,
	}
}

func (f *factory) CreateAuthProviderBackend(ctx context.Context, id string, uiEndpoints []string, config map[string]string) (authproviders.AuthProviderBackend, map[string]string, error) {
	providerURLPathPrefix := f.urlPathPrefix + id + "/"
	return newProvider(ctx, id, uiEndpoints, providerURLPathPrefix, config)
}

func (f *factory) ProcessHTTPRequest(w http.ResponseWriter, r *http.Request) (string, error) {
	restPath := strings.TrimPrefix(r.URL.Path, f.urlPathPrefix)
	if len(restPath) == len(r.URL.Path) {
		return "", httputil.NewError(http.StatusNotFound, "Not Found")
	}
	if restPath == "" {
		return "", httputil.NewError(http.StatusForbidden, "Forbidden")
	}
	pathComponents := strings.SplitN(restPath, "/", 2)
	return pathComponents[0], nil
}

func (f *factory) ResolveProvider(state string) (string, error) {
	return "", status.Errorf(codes.Unimplemented, "not implemented")
}
