package backend

import (
	"errors"
	"testing"

	"github.com/stackrox/rox/pkg/auth/tokens"
	tokensMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestIssueRoleTokenHTTPRequestScope(t *testing.T) {
	issuer := tokensMocks.NewMockIssuer(gomock.NewController(t))
	issueError := errors.New("issue failed")
	scopes := tokens.HTTPRequestScopes{"/v1/metadata": {"GET"}}
	issuer.EXPECT().Issue(gomock.Any(), tokens.RoxClaims{
		RoleNames:         []string{"Analyst"},
		Name:              "test",
		HTTPRequestScopes: tokens.HTTPRequestScopes{"/v1/metadata": {"GET"}},
	}).Return(nil, issueError)
	backend := &backendImpl{issuer: issuer}
	_, _, err := backend.IssueRoleToken(t.Context(), "test", []string{"Analyst"}, nil, scopes)
	assert.ErrorIs(t, err, issueError)

	_, _, err = backend.IssueRoleToken(t.Context(), "test", []string{"Analyst"}, nil,
		tokens.HTTPRequestScopes{"/v1/../admin": {"POST"}})
	assert.ErrorIs(t, err, errox.InvalidArgs)

	issuer.EXPECT().Issue(gomock.Any(), tokens.RoxClaims{
		RoleNames: []string{"Analyst"}, Name: "test",
		HTTPRequestScopes: tokens.HTTPRequestScopes{"/v1/metadata": {}},
	}).Return(nil, issueError)
	_, _, err = backend.IssueRoleToken(t.Context(), "test", []string{"Analyst"}, nil,
		tokens.HTTPRequestScopes{"/v1/metadata": {}})
	assert.ErrorIs(t, err, issueError)
}
