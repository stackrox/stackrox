package apitoken

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/apitoken/store"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/jwt"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sync"
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
	GetTokenOrNil(tokenID string) (*storage.TokenMetadata, error)
	GetTokens(req *v1.GetAPITokensRequest) ([]*storage.TokenMetadata, error)
	IssueRoleToken(name string, role *storage.Role) (string, *storage.TokenMetadata, error)
	RevokeToken(tokenID string) (bool, error)
}

type backend struct {
	*source
	issuer tokens.Issuer
}

func newBackend(issuerFactory tokens.IssuerFactory, tokenStore store.Store) (*backend, error) {
	src, err := newSource(tokenStore)
	if err != nil {
		return nil, errors.Wrap(err, "creating source")
	}
	issuer, err := issuerFactory.CreateIssuer(src, tokens.WithDefaultTTL(defaultTTL))
	if err != nil {
		return nil, errors.Wrap(err, "creating issuer")
	}
	return &backend{
		source: src,
		issuer: issuer,
	}, nil
}

func (c *backend) IssueRoleToken(name string, role *storage.Role) (string, *storage.TokenMetadata, error) {
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

func metadataFromTokenInfo(name string, info *tokens.TokenInfo) *storage.TokenMetadata {
	return &storage.TokenMetadata{
		Id:         info.ID,
		Name:       name,
		Role:       info.RoleName,
		IssuedAt:   protoconv.ConvertTimeToTimestamp(info.IssuedAt()),
		Expiration: protoconv.ConvertTimeToTimestamp(info.Expiry()),
	}
}
