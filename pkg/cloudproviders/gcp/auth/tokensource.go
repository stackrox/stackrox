package auth

import (
	"context"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

// CredentialManagerTokenSource provides tokens provided by the credential manager.
type CredentialManagerTokenSource struct {
	credManager CredentialsManager
}

var _ oauth2.TokenSource = &CredentialManagerTokenSource{}

// Token returns a managed token.
func (t *CredentialManagerTokenSource) Token() (*oauth2.Token, error) {
	creds, err := t.credManager.GetCredentials(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get credentials")
	}
	return creds.TokenSource.Token()
}
