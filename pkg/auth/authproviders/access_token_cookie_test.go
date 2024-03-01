package authproviders

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stretchr/testify/assert"
)

func TestIsAccessTokenCookie(t *testing.T) {
	testTokenInfo := &tokens.TokenInfo{Token: "some-token"}
	cases := map[string]struct {
		cookie func() *http.Cookie
		expect bool
	}{
		"AccessTokenCookie should be true": {
			cookie: func() *http.Cookie {
				return AccessTokenCookie(testTokenInfo)
			},
			expect: true,
		},
		"Cookie with different name should be false": {
			cookie: func() *http.Cookie {
				c := AccessTokenCookie(testTokenInfo)
				c.Name = "NotRoxAccessToken"
				return c
			},
		},
		"Cookie without secure should be false": {
			cookie: func() *http.Cookie {
				c := AccessTokenCookie(testTokenInfo)
				c.Secure = false
				return c
			},
		},
		"Cookie without same site strict should be false": {
			cookie: func() *http.Cookie {
				c := AccessTokenCookie(testTokenInfo)
				c.SameSite = http.SameSiteLaxMode
				return c
			},
		},
		"Cookie without http only should be false": {
			cookie: func() *http.Cookie {
				c := AccessTokenCookie(testTokenInfo)
				c.HttpOnly = false
				return c
			},
		},
		"Cookie with lowercase name should be false": {
			cookie: func() *http.Cookie {
				c := AccessTokenCookie(testTokenInfo)
				c.Name = strings.ToLower(AccessTokenCookieName)
				return c
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expect, IsAccessTokenCookie(tc.cookie()))
		})
	}
}
