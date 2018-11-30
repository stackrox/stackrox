package basic

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/monoclock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	challengeHandlerPath      = "challenge"
	clientStateQueryParamName = "client_state"

	defaultTTL = 24 * time.Hour

	loginTimeLimit = 5 * time.Second
)

var (
	tokenOptions = []tokens.Option{
		tokens.WithDefaultTTL(defaultTTL),
	}
)

type provider struct {
	urlPathPrefix string
	monoClock     monoclock.MonoClock
}

func (p *provider) ExchangeToken(ctx context.Context, externalRawToken, state string) (*tokens.ExternalUserClaim, []tokens.Option, string, error) {
	return nil, nil, "", status.Errorf(codes.Unimplemented, "basic auth provider does not implement ExchangeToken")
}

func (p *provider) LoginURL(clientState string) string {
	queryParams := url.Values{}
	queryParams.Set(clientStateQueryParamName, clientState)
	queryParams.Set("micro_ts", strconv.FormatInt(int64(p.monoClock.SinceEpoch()/time.Microsecond), 10))
	u := &url.URL{
		Path:     p.urlPathPrefix + challengeHandlerPath,
		RawQuery: queryParams.Encode(),
	}
	return u.String()
}

func (p *provider) RefreshURL() string {
	return ""
}

func newProvider(ctx context.Context, id string, uiEndpoint string, urlPathPrefix string, config map[string]string) (*provider, map[string]string, error) {
	provider := &provider{
		urlPathPrefix: urlPathPrefix,
		monoClock:     monoclock.New(),
	}
	return provider, nil, nil
}

func (p *provider) ProcessHTTPRequest(w http.ResponseWriter, r *http.Request) (*tokens.ExternalUserClaim, []tokens.Option, string, error) {
	restPath := strings.TrimPrefix(r.URL.Path, p.urlPathPrefix)
	if len(restPath) == len(r.URL.Path) {
		return nil, nil, "", httputil.NewError(http.StatusNotFound, "Not Found")
	}

	if restPath != challengeHandlerPath {
		return nil, nil, "", httputil.NewError(http.StatusNotFound, "Not Found")
	}
	if r.Method != http.MethodGet {
		return nil, nil, "", httputil.NewError(http.StatusBadRequest, "Bad Request")
	}

	// If logging in via basic auth, the identity extractor of the request pipeline should already have validated our
	// identity.
	identity := authn.IdentityFromContext(r.Context())
	if identity != nil {
		if basicAuthIdentity, ok := identity.(basic.Identity); ok {
			return basicAuthIdentity.AsExternalUser(), tokenOptions, r.URL.Query().Get(clientStateQueryParamName), nil
		}
	}

	// Otherwise, cause the browser to display the challenge dialog.
	microTS, err := strconv.ParseInt(r.URL.Query().Get("micro_ts"), 10, 64)
	if err != nil {
		return nil, nil, "", httputil.NewError(http.StatusInternalServerError, "Unparseable microtimestamp")
	}

	age := p.monoClock.SinceEpoch() - time.Microsecond*time.Duration(microTS)

	if age > loginTimeLimit {
		return nil, nil, "", errors.New("invalid username or password")
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="StackRox" charset="UTF-8"`)
	w.WriteHeader(http.StatusUnauthorized)
	return nil, nil, "", nil
}
