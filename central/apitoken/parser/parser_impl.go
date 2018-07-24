package parser

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/apitoken"
	"bitbucket.org/stack-rox/apollo/central/apitoken/signer"
	"bitbucket.org/stack-rox/apollo/central/role/store"
	"bitbucket.org/stack-rox/apollo/pkg/auth/tokenbased"
	"bitbucket.org/stack-rox/apollo/pkg/jwt"
)

type parserImpl struct {
	signer    signer.Signer
	roleStore store.Store
}

// The RoleMapper is not required by this implementation of IdentityParser because API tokens encode their role
// in the subject claim.
func (p *parserImpl) Parse(headers map[string][]string, _ tokenbased.RoleMapper) (tokenbased.Identity, error) {
	validator := jwt.NewRS256Validator(p.signer, apitoken.Issuer, apitoken.Audience)
	_, claims, err := validator.Validate(headers)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %s", err)
	}
	role, exists := p.roleStore.GetRole(claims.Subject)
	if !exists {
		return nil, fmt.Errorf("subject %s does not correspond to an existing role", claims.Subject)
	}
	return tokenbased.NewIdentity(claims.ID, role, claims.Expiry.Time()), nil
}
