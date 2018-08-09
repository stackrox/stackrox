package cachedstore

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

const (
	fakeID   = "FAKEID"
	fakeRole = "FAKEROLE"
)

type mockStore struct {
	tokens map[string]*v1.TokenMetadata
}

func (m *mockStore) AddToken(token *v1.TokenMetadata) error {
	m.tokens[token.GetId()] = token
	return nil
}

func (m *mockStore) GetToken(id string) (token *v1.TokenMetadata, exists bool, err error) {
	token, exists = m.tokens[token.GetId()]
	return
}

func (m *mockStore) GetTokens(*v1.GetAPITokensRequest) ([]*v1.TokenMetadata, error) {
	tokens := make([]*v1.TokenMetadata, 0, len(m.tokens))
	for _, token := range m.tokens {
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func (m *mockStore) RevokeToken(id string) (exists bool, err error) {
	token := m.tokens[id]
	token.Revoked = true
	m.tokens[id] = token
	return true, nil
}

type CachedStoreTestSuite struct {
	suite.Suite
	cachedStore CachedStore
}

func (suite *CachedStoreTestSuite) mustCreateCachedStore(initialTokens ...*v1.TokenMetadata) CachedStore {
	tokens := make(map[string]*v1.TokenMetadata)
	for _, token := range initialTokens {
		tokens[token.GetId()] = token
	}

	s := &mockStore{tokens: tokens}
	cachedStore, err := New(s)
	suite.Require().NoError(err)
	return cachedStore
}

func (suite *CachedStoreTestSuite) TestRevocation() {
	cachedStore := suite.mustCreateCachedStore()
	suite.Require().NoError(cachedStore.CheckTokenRevocation(fakeID))

	err := cachedStore.AddToken(&v1.TokenMetadata{Id: fakeID})
	suite.Require().NoError(err)

	suite.Require().NoError(cachedStore.CheckTokenRevocation(fakeID))

	exists, err := cachedStore.RevokeToken(fakeID)
	suite.Require().True(exists)
	suite.Require().NoError(err)
	suite.Error(cachedStore.CheckTokenRevocation(fakeID))
	suite.NoError(cachedStore.CheckTokenRevocation("ARBITRARY"))
}

func (suite *CachedStoreTestSuite) TestWorksWhenLoadingFromStore() {
	const nonRevoked = "ARBITRARY"

	cachedStore := suite.mustCreateCachedStore([]*v1.TokenMetadata{
		{Id: fakeID, Revoked: true},
		{Id: nonRevoked},
	}...)

	suite.Error(cachedStore.CheckTokenRevocation(fakeID))
	suite.NoError(cachedStore.CheckTokenRevocation(nonRevoked))
}

func TestCachedStore(t *testing.T) {
	suite.Run(t, new(CachedStoreTestSuite))
}
