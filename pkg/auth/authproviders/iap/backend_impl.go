package iap

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/auth/authproviders"
	"github.com/stackrox/stackrox/pkg/auth/tokens"
	"github.com/stackrox/stackrox/pkg/grpc/requestinfo"
	"github.com/stackrox/stackrox/pkg/jwt"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/sliceutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	joseJwt "gopkg.in/square/go-jose.v2/jwt"
)

var (
	log                   = logging.LoggerForModule()
	errFingerPrintChanged = errors.New("IAP token fingerprint changed, please log in again")
)

const (
	jwtAssertion = "X-Goog-IAP-JWT-Assertion"
	issuerID     = "https://cloud.google.com/iap"
	publicKeyURL = "https://www.gstatic.com/iap/verify/public_key-jwk"

	refreshToken = "iap_refresh"
)

func newBackend(audience string, loginURL string) (authproviders.Backend, error) {
	validator, err := jwt.CreateES256Validator(issuerID, joseJwt.Audience{audience}, publicKeyURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create auth provider backend")
	}
	return &backendImpl{
		audience:  audience,
		issuerID:  issuerID,
		validator: validator,
		loginURL:  loginURL,
	}, nil
}

type googleClaims struct {
	AccessLevels []string `json:"access_levels,omitempty"`
}
type extraClaims struct {
	Email  string       `json:"email,omitempty"`
	Hd     string       `json:"hd,omitempty"`
	Google googleClaims `json:"google,omitempty"`
}

type backendImpl struct {
	audience  string
	issuerID  string
	validator jwt.Validator
	loginURL  string
}

func (p *backendImpl) Config() map[string]string {
	return map[string]string{
		AudienceConfigKey: p.audience,
	}
}

func (p *backendImpl) OnEnable(provider authproviders.Provider) {
}

func (p *backendImpl) OnDisable(provider authproviders.Provider) {
}

func (p *backendImpl) LoginURL(clientState string, ri *requestinfo.RequestInfo) (string, error) {
	return p.loginURL, nil
}

func (p *backendImpl) RefreshURL() string {
	return ""
}

func (p *backendImpl) ProcessHTTPRequest(w http.ResponseWriter, r *http.Request) (*authproviders.AuthResponse, error) {
	token := r.Header.Get(jwtAssertion)
	if token == "" {
		return nil, errors.New("invalid request, expected assertion not found")
	}
	return p.getAuthResponse(token)
}

func (p *backendImpl) ExchangeToken(ctx context.Context, externalToken, state string) (*authproviders.AuthResponse, string, error) {
	return nil, "", status.Errorf(codes.Unimplemented, "ExchangeToken not implemented for provider type %q", TypeName)
}

func (p *backendImpl) Validate(ctx context.Context, roxClaims *tokens.Claims) error {
	ri := requestinfo.FromContext(ctx)
	token := ri.HTTPRequest.Headers.Get(jwtAssertion)

	var jwtClaims joseJwt.Claims
	var extraClaims extraClaims
	err := p.validator.Validate(token, &jwtClaims, &extraClaims)

	if err != nil {
		return errors.Wrap(err, "invalid request token")
	}

	if jwtClaims.Subject != roxClaims.ExternalUser.UserID {
		return errFingerPrintChanged
	}

	if extraClaims.Email != roxClaims.ExternalUser.Email {
		return errFingerPrintChanged
	}

	if !sliceutils.StringEqual([]string{extraClaims.Hd}, []string{roxClaims.ExternalUser.Attributes["hd"][0]}) {
		return errFingerPrintChanged
	}

	return nil
}

func (p *backendImpl) RefreshAccessToken(ctx context.Context, _ authproviders.RefreshTokenData) (*authproviders.AuthResponse, error) {
	ri := requestinfo.FromContext(ctx)
	token := ri.HTTPRequest.Headers.Get(jwtAssertion)

	return p.getAuthResponse(token)
}

func (p *backendImpl) RevokeRefreshToken(ctx context.Context, _ authproviders.RefreshTokenData) error {
	// Not required to be implemented for this provider
	return nil
}

func (p *backendImpl) getAuthResponse(token string) (*authproviders.AuthResponse, error) {
	var claims joseJwt.Claims
	var extraClaims extraClaims
	err := p.validator.Validate(token, &claims, &extraClaims)

	if err != nil {
		return nil, errors.Wrap(err, "invalid token")
	}

	authResp := &authproviders.AuthResponse{
		Claims: &tokens.ExternalUserClaim{
			UserID:   claims.Subject,
			FullName: extraClaims.Email,
			Email:    extraClaims.Email,
			Attributes: map[string][]string{
				authproviders.UseridAttribute: {claims.Subject},
				authproviders.EmailAttribute:  {extraClaims.Email},
				"hd":                          {extraClaims.Hd},
				"access_levels":               extraClaims.Google.AccessLevels,
			},
		},
		Expiration: claims.Expiry.Time(),
		RefreshTokenData: authproviders.RefreshTokenData{
			RefreshToken: refreshToken,
		},
	}
	return authResp, nil
}
