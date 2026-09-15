package auth

import (
	"context"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const earlyExpiry = 5 * time.Minute

// CreateTokenSourceFromConfig creates a token source based on the config.
func CreateTokenSourceFromConfig(ctx context.Context,
	credsJSON []byte, wifEnabled bool, scopes ...string,
) (oauth2.TokenSource, error) {
	if wifEnabled {
		creds, err := google.FindDefaultCredentials(ctx, scopes...)
		if err != nil {
			return nil, err
		}
		return oauth2.ReuseTokenSourceWithExpiry(nil, creds.TokenSource, earlyExpiry), nil
	}
	creds, err := google.CredentialsFromJSON(ctx, credsJSON, scopes...)
	if err != nil {
		return nil, err
	}
	return oauth2.ReuseTokenSourceWithExpiry(nil, creds.TokenSource, earlyExpiry), nil
}

// CreateTokenSourceFromConfigWithManager creates a token source based on the config.
func CreateTokenSourceFromConfigWithManager(ctx context.Context, manager STSTokenManager,
	credsJSON []byte, wifEnabled bool, scopes ...string,
) (oauth2.TokenSource, error) {
	if wifEnabled {
		return manager.TokenSource(), nil
	}
	creds, err := google.CredentialsFromJSON(ctx, credsJSON, scopes...)
	if err != nil {
		return nil, err
	}
	return oauth2.ReuseTokenSourceWithExpiry(nil, creds.TokenSource, earlyExpiry), nil
}
