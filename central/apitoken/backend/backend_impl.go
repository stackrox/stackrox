package backend

import (
	"context"

	"github.com/stackrox/rox/central/apitoken/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timeutil"
)

type backendImpl struct {
	tokenStore datastore.DataStore
	issuer     tokens.Issuer
	source     *sourceImpl
}

func (c *backendImpl) GetTokenOrNil(ctx context.Context, tokenID string) (*storage.TokenMetadata, error) {
	return c.tokenStore.GetTokenOrNil(ctx, tokenID)
}

func (c *backendImpl) GetTokens(ctx context.Context, req *v1.GetAPITokensRequest) ([]*storage.TokenMetadata, error) {
	return c.tokenStore.GetTokens(ctx, req)
}

func (c *backendImpl) IssueRoleToken(ctx context.Context, name string, role *storage.Role) (string, *storage.TokenMetadata, error) {
	tokenInfo, err := c.issuer.Issue(tokens.RoxClaims{RoleName: role.GetName()})
	if err != nil {
		return "", nil, err
	}

	md := metadataFromTokenInfo(name, tokenInfo)

	if err := c.tokenStore.AddToken(ctx, md); err != nil {
		return "", nil, err
	}

	return tokenInfo.Token, md, nil
}

func (c *backendImpl) RevokeToken(ctx context.Context, tokenID string) (bool, error) {
	t, err := c.tokenStore.GetTokenOrNil(ctx, tokenID)
	if err != nil {
		return false, err
	}
	if t == nil {
		return false, nil
	}
	if t.Revoked {
		return true, nil
	}

	expiry := protoconv.ConvertTimestampToTimeOrDefault(t.GetExpiration(), timeutil.Max)
	exists, err := c.tokenStore.RevokeToken(ctx, tokenID)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	c.source.Revoke(tokenID, expiry)
	return true, nil
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
