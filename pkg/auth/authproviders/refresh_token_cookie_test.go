package authproviders

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshTokenCookieData_EncodeDecodeRoundTrip(t *testing.T) {
	t.Parallel()

	data := refreshTokenCookieData{
		ProviderType: "myProvider",
		ProviderID:   "myProviderID",
		RefreshToken: "refreshToken%WithSome:Special??Characterß",
	}

	encoded, err := data.Encode()
	require.NoError(t, err)

	var decoded refreshTokenCookieData
	require.NoError(t, decoded.Decode(encoded))

	assert.Equal(t, data, decoded)
}

func TestRefreshTokenCookieData_TestEncode(t *testing.T) {
	t.Parallel()

	data := refreshTokenCookieData{
		ProviderType: "myProvider",
		ProviderID:   "myProviderID",
		RefreshToken: "MyToken",
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
	t.Parallel()

	encoded := "providerType=myProvider&providerId=myProviderID&refreshToken=My%20Token"
	var decoded refreshTokenCookieData
	require.NoError(t, decoded.Decode(encoded))

	expected := refreshTokenCookieData{
		ProviderType: "myProvider",
		ProviderID:   "myProviderID",
		RefreshToken: "My Token",
	}

	assert.Equal(t, expected, decoded)
}
