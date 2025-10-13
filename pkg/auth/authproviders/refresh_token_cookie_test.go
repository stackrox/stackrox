package authproviders

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshTokenCookieData_EncodeDecodeRoundTrip(t *testing.T) {

	data := refreshTokenCookieData{
		ProviderType: "myProvider",
		ProviderID:   "myProviderID",
		RefreshTokenData: RefreshTokenData{
			RefreshToken: "refreshToken%WithSome:Special??Characterß",
		},
	}

	encoded, err := data.Encode()
	require.NoError(t, err)

	var decoded refreshTokenCookieData
	require.NoError(t, decoded.Decode(encoded))

	assert.Equal(t, data, decoded)
}

func TestRefreshTokenCookieData_EncodeDecodeRoundTrip_WithType(t *testing.T) {

	data := refreshTokenCookieData{
		ProviderType: "myProvider",
		ProviderID:   "myProviderID",
		RefreshTokenData: RefreshTokenData{
			RefreshToken:     "refreshToken%WithSome:Special??Characterß",
			RefreshTokenType: "access_token",
		},
	}

	encoded, err := data.Encode()
	require.NoError(t, err)

	var decoded refreshTokenCookieData
	require.NoError(t, decoded.Decode(encoded))

	assert.Equal(t, data, decoded)
}

func TestRefreshTokenCookieData_TestEncode(t *testing.T) {

	data := refreshTokenCookieData{
		ProviderType: "myProvider",
		ProviderID:   "myProviderID",
		RefreshTokenData: RefreshTokenData{
			RefreshToken: "MyToken",
		},
	}

	encoded, err := data.Encode()
	require.NoError(t, err)

	validEncodings := []string{
		"providerType=myProvider&providerId=myProviderID&refreshToken=MyToken",
		"providerId=myProviderID&refreshToken=MyToken&providerType=myProvider",
		"refreshToken=MyToken&providerType=myProvider&providerId=myProviderID",
		"providerType=myProvider&refreshToken=MyToken&providerId=myProviderID",
		"refreshToken=MyToken&providerId=myProviderID&providerType=myProvider",
		"providerId=myProviderID&providerType=myProvider&refreshToken=MyToken",
	}

	assert.Contains(t, validEncodings, encoded)
}

func TestRefreshTokenCookieData_TestDecode(t *testing.T) {

	encoded := "providerType=myProvider&providerId=myProviderID&refreshToken=My%20Token"
	var decoded refreshTokenCookieData
	require.NoError(t, decoded.Decode(encoded))

	expected := refreshTokenCookieData{
		ProviderType: "myProvider",
		ProviderID:   "myProviderID",
		RefreshTokenData: RefreshTokenData{
			RefreshToken: "My Token",
		},
	}

	assert.Equal(t, expected, decoded)
}

func TestRefreshTokenCookieData_TestDecode_WithType(t *testing.T) {

	encoded := "providerType=myProvider&refreshTokenType=access_token&providerId=myProviderID&refreshToken=My%20Token"
	var decoded refreshTokenCookieData
	require.NoError(t, decoded.Decode(encoded))

	expected := refreshTokenCookieData{
		ProviderType: "myProvider",
		ProviderID:   "myProviderID",
		RefreshTokenData: RefreshTokenData{
			RefreshToken:     "My Token",
			RefreshTokenType: "access_token",
		},
	}

	assert.Equal(t, expected, decoded)
}
