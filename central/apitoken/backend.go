package apitoken

import (
	"fmt"
	"sync"

	"github.com/stackrox/rox/central/apitoken/store"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/jwt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
)

var (
	log = logging.LoggerForModule()

	backendInstance     Backend
	initBackendInstance sync.Once
)

// BackendSingleton returns the apitoken backend singleton instance.
func BackendSingleton() Backend {
	initBackendInstance.Do(func() {
		tokenStore := store.New(globaldb.GetGlobalDB())
		var err error
		instance, err := newBackend(jwt.IssuerFactorySingleton(), tokenStore)
		if err != nil {
			log.Panicf("Could not create API tokens backend: %v", err)
		}
		backendInstance = instance
	})
	return backendInstance
}

// Backend is the backend for the API tokens component.
type Backend interface {
	GetTokenOrNil(tokenID string) (*v1.TokenMetadata, error)
	GetTokens(req *v1.GetAPITokensRequest) ([]*v1.TokenMetadata, error)
	IssueRoleToken(name string, role *v1.Role) (string, *v1.TokenMetadata, error)
	RevokeToken(tokenID string) (bool, error)
}

type backend struct {
	*source
	issuer tokens.Issuer
}

func newBackend(issuerFactory tokens.IssuerFactory, tokenStore store.Store) (*backend, error) {
	src, err := newSource(tokenStore)
	if err != nil {
		return nil, fmt.Errorf("creating source: %v", err)
	}
	issuer, err := issuerFactory.CreateIssuer(src, tokens.WithDefaultTTL(defaultTTL))
	if err != nil {
		return nil, fmt.Errorf("creating issuer: %v", err)
	}
	return &backend{
		source: src,
		issuer: issuer,
	}, nil
}

func (c *backend) IssueRoleToken(name string, role *v1.Role) (string, *v1.TokenMetadata, error) {
	tokenInfo, err := c.issuer.Issue(tokens.RoxClaims{RoleName: role.GetName()})
	if err != nil {
		return "", nil, err
	}

	md := metadataFromTokenInfo(name, tokenInfo)

	if err := c.source.AddToken(md); err != nil {
		return "", nil, err
	}

	return tokenInfo.Token, md, nil
}

func metadataFromTokenInfo(name string, info *tokens.TokenInfo) *v1.TokenMetadata {
	return &v1.TokenMetadata{
		Id:         info.ID,
		Name:       name,
		Role:       info.RoleName,
		IssuedAt:   protoconv.ConvertTimeToTimestamp(info.IssuedAt()),
		Expiration: protoconv.ConvertTimeToTimestamp(info.Expiry()),
	}
}
