package parser

import (
	"fmt"

	"github.com/stackrox/rox/central/apitoken"
	"github.com/stackrox/rox/central/apitoken/signer"
	"github.com/stackrox/rox/central/role/store"
	"github.com/stackrox/rox/pkg/auth/tokenbased"
	pkgJWT "github.com/stackrox/rox/pkg/jwt"
	"gopkg.in/square/go-jose.v2/jwt"
)

type parserImpl struct {
	signer            signer.Signer
	roleStore         store.Store
	revocationChecker tokenRevocationChecker
}

func (p *parserImpl) validator() pkgJWT.Validator {
	return pkgJWT.NewRS256Validator(p.signer, apitoken.Issuer, apitoken.Audience)
}

func (p *parserImpl) parseClaims(claims *jwt.Claims) (tokenbased.Identity, error) {
	role, exists := p.roleStore.GetRole(claims.Subject)
	if !exists {
		return nil, fmt.Errorf("subject %s does not correspond to an existing role", claims.Subject)
	}
	if err := p.revocationChecker.CheckTokenRevocation(claims.ID); err != nil {
		return nil, fmt.Errorf("token is revoked: %s", err)
	}
	return tokenbased.NewIdentity(claims.ID, role, claims.Expiry.Time()), nil
}

// The RoleMapper is not required by this implementation of IdentityParser because API tokens encode their role
// in the subject claim.
func (p *parserImpl) Parse(headers map[string][]string, _ tokenbased.RoleMapper) (tokenbased.Identity, error) {
	_, claims, err := p.validator().ValidateFromHeaders(headers)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %s", err)
	}
	return p.parseClaims(claims)
}

func (p *parserImpl) ParseToken(token string) (tokenbased.Identity, error) {
	claims, err := p.validator().Validate(token)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %s", err)
	}
	return p.parseClaims(claims)
}
