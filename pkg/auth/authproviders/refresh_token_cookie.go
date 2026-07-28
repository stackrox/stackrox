package authproviders

import (
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

const (
	// RefreshTokenCookieName is the name of the cookie containing the refresh token.
	RefreshTokenCookieName = "RoxRefreshToken"
)

// RefreshTokenData encapsulates data relevant to refresh tokens.
type RefreshTokenData struct {
	RefreshToken     string `schema:"refreshToken,required"`
	RefreshTokenType string `schema:"refreshTokenType,omitempty"`
}

// Type returns the inferred type of the refresh token stored in this type.
func (d *RefreshTokenData) Type() string {
	if d.RefreshTokenType != "" {
		return d.RefreshTokenType
	}
	return "refresh_token"
}

type refreshTokenCookieData struct {
	ProviderType string `schema:"providerType,required"`
	ProviderID   string `schema:"providerId,required"`
	RefreshTokenData
}

func cookieDataFromRequest(req *http.Request) (*refreshTokenCookieData, error) {
	cookie, err := req.Cookie(RefreshTokenCookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			return nil, nil
		}
		return nil, err
	}

	var data refreshTokenCookieData
	if err := data.Decode(cookie.Value); err != nil {
		return nil, errors.Wrap(err, "decoding cookie value")
	}

	return &data, nil
}

func (r *refreshTokenCookieData) Encode() (string, error) {
	vals := url.Values{
		"providerType": {r.ProviderType},
		"providerId":   {r.ProviderID},
		"refreshToken": {r.RefreshToken},
	}
	if r.RefreshTokenType != "" {
		vals.Set("refreshTokenType", r.RefreshTokenType)
	}
	return vals.Encode(), nil
}

func (r *refreshTokenCookieData) Decode(encoded string) error {
	vals, err := url.ParseQuery(encoded)
	if err != nil {
		return errors.Wrap(err, "parsing encoded cookie data")
	}
	r.ProviderType = vals.Get("providerType")
	r.ProviderID = vals.Get("providerId")
	r.RefreshToken = vals.Get("refreshToken")
	r.RefreshTokenType = vals.Get("refreshTokenType")
	return nil
}
