package basic

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/monoclock"
)

const (
	challengeHandlerPath      = "challenge"
	clientStateQueryParamName = "client_state"

	defaultTTL = 24 * time.Hour

	loginTimeLimit = 5 * time.Second
)

type backendImpl struct {
	urlPathPrefix string
	monoClock     monoclock.MonoClock

	basicAuthMgr *basic.Manager
}

func (p *backendImpl) OnEnable(_ authproviders.Provider) {
}

func (p *backendImpl) OnDisable(_ authproviders.Provider) {
}

func (p *backendImpl) ExchangeToken(ctx context.Context, externalRawToken, state string) (*authproviders.AuthResponse, string, error) {
	urlValues, err := url.ParseQuery(externalRawToken)
	if err != nil {
		return nil, "", authproviders.CreateError("failed to parse credentials form data", err)
	}
	username, password := urlValues.Get("username"), urlValues.Get("password")
	id, err := p.basicAuthMgr.IdentityForCreds(ctx, username, password, nil)
	if err != nil {
		return nil, "", authproviders.CreateError("failed to authenticate", err)
	}

	return &authproviders.AuthResponse{
		Claims:     id.AsExternalUser(),
		Expiration: time.Now().Add(defaultTTL),
	}, state, nil
}

func (p *backendImpl) LoginURL(clientState string, _ *requestinfo.RequestInfo) (string, error) {
	queryParams := url.Values{}
	queryParams.Set(clientStateQueryParamName, clientState)
	queryParams.Set("micro_ts", strconv.FormatInt(int64(p.monoClock.SinceEpoch()/time.Microsecond), 10))
	u := &url.URL{
		Path:     p.urlPathPrefix + challengeHandlerPath,
		RawQuery: queryParams.Encode(),
	}
	return u.String(), nil
}

func (p *backendImpl) Config() map[string]string {
	return nil
}

func (p *backendImpl) RefreshURL() string {
	return ""
}

func newBackend(urlPathPrefix string, basicAuthMgr *basic.Manager) (*backendImpl, error) {
	backendImpl := &backendImpl{
		urlPathPrefix: urlPathPrefix,
		monoClock:     monoclock.New(),
		basicAuthMgr:  basicAuthMgr,
	}
	return backendImpl, nil
}

func (p *backendImpl) ProcessHTTPRequest(w http.ResponseWriter, r *http.Request) (*authproviders.AuthResponse, error) {
	restPath := strings.TrimPrefix(r.URL.Path, p.urlPathPrefix)
	if len(restPath) == len(r.URL.Path) {
		return nil, httputil.NewError(http.StatusNotFound, "Not Found")
	}

	if restPath != challengeHandlerPath {
		return nil, httputil.NewError(http.StatusNotFound, "Not Found")
	}
	if r.Method != http.MethodGet {
		return nil, httputil.NewError(http.StatusBadRequest, "Bad Request")
	}

	// If logging in via basic auth, the identity extractor of the request pipeline should already have validated our
	// identity.
	identity := authn.IdentityFromContextOrNil(r.Context())
	if identity != nil {
		if basicAuthIdentity, ok := identity.(basic.Identity); ok {
			authResp := &authproviders.AuthResponse{
				Claims:     basicAuthIdentity.AsExternalUser(),
				Expiration: time.Now().Add(defaultTTL),
			}
			return authResp, nil
		}
	}

	// Otherwise, cause the browser to display the challenge dialog.
	microTS, err := strconv.ParseInt(r.URL.Query().Get("micro_ts"), 10, 64)
	if err != nil {
		return nil, httputil.NewError(http.StatusInternalServerError, "Unparseable microtimestamp")
	}

	age := p.monoClock.SinceEpoch() - time.Microsecond*time.Duration(microTS)

	if age > loginTimeLimit {
		return nil, errors.New("invalid username or password")
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="StackRox" charset="UTF-8"`)
	w.WriteHeader(http.StatusUnauthorized)
	return nil, nil
}

func (p *backendImpl) Validate(_ context.Context, _ *tokens.Claims) error {
	return nil
}
