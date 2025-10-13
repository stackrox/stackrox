package auth

import "golang.org/x/oauth2"

// STSTokenManager manages GCP short-living tokens.
type STSTokenManager interface {
	Start()
	Stop()
	TokenSource() oauth2.TokenSource
}
