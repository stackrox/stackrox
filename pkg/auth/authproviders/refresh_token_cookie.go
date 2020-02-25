package authproviders

import (
	"net/http"
	"net/url"

	"github.com/gorilla/schema"
	"github.com/pkg/errors"
)

const (
	refreshTokenCookieName = "RoxRefreshToken"
)

var (
	schemaEncoder = schema.NewEncoder()
	schemaDecoder = schema.NewDecoder()
)

type refreshTokenCookieData struct {
	ProviderType string `schema:"providerType,required"`
	ProviderID   string `schema:"providerId,required"`
	RefreshToken string `schema:"refreshToken,required"`
}

func cookieDataFromRequest(req *http.Request) (*refreshTokenCookieData, error) {
	cookie, err := req.Cookie(refreshTokenCookieName)
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
	vals := make(url.Values)
	if err := schemaEncoder.Encode(r, vals); err != nil {
		return "", err
	}
	return vals.Encode(), nil
}

func (r *refreshTokenCookieData) Decode(encoded string) error {
	vals, err := url.ParseQuery(encoded)
	if err != nil {
		return errors.Wrap(err, "parsing encoded cookie data")
	}
	return schemaDecoder.Decode(r, vals)
}
