package m2m

import (
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	_ claimExtractor = (*githubClaimExtractor)(nil)
	_ claimExtractor = (*genericClaimExtractor)(nil)
)

type claimExtractor interface {
	ExtractRoxClaims(idToken *oidc.IDToken) (tokens.RoxClaims, error)
}

func newClaimExtractorFromConfig(config *storage.AuthMachineToMachineConfig) claimExtractor {
	if config.GetType() == storage.AuthMachineToMachineConfig_GENERIC {
		return &genericClaimExtractor{configID: config.GetId()}
	}

	return &githubClaimExtractor{configID: config.GetId()}
}

type genericClaimExtractor struct {
	configID string
}

func (g *genericClaimExtractor) ExtractRoxClaims(idToken *oidc.IDToken) (tokens.RoxClaims, error) {
	var unstructured map[string]interface{}
	if err := idToken.Claims(&unstructured); err != nil {
		return tokens.RoxClaims{}, errors.Wrap(err, "extracting claims")
	}

	return createRoxClaimsFromGenericClaims(idToken.Subject, unstructured), nil
}

func createRoxClaimsFromGenericClaims(subject string, unstructured map[string]interface{}) tokens.RoxClaims {
	stringClaims := mapToStringClaims(unstructured)

	friendlyName := getFriendlyName(stringClaims)

	userID := utils.IfThenElse(friendlyName == "", subject,
		fmt.Sprintf("%s|%s", subject, friendlyName))

	userClaims := &tokens.ExternalUserClaim{
		UserID:     userID,
		FullName:   stringutils.FirstNonEmpty(friendlyName, userID),
		Attributes: mapToStringClaims(unstructured),
	}

	return tokens.RoxClaims{
		ExternalUser: userClaims,
		Name:         userID,
	}
}

func getFriendlyName(claims map[string][]string) string {
	// These are some sample claims that typically have the user's name or email.
	userNameClaims := []string{
		"email",
		"preferred_username",
		"full_name",
	}

	for _, userNameClaim := range userNameClaims {
		if value, ok := claims[userNameClaim]; ok && len(value) == 1 {
			return value[0]
		}
	}

	return ""
}

type githubClaimExtractor struct {
	configID string
}

// Claims of the ID token issued for github actions.
// See: https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect#understanding-the-oidc-token
type githubActionClaims struct {
	Actor           string `json:"actor"`
	ActorID         string `json:"actor_id"`
	Environment     string `json:"environment"`
	EventName       string `json:"event_name"`
	GitRef          string `json:"ref"`
	Repository      string `json:"repository"`
	RepositoryOwner string `json:"repository_owner"`
}

func (g *githubClaimExtractor) ExtractRoxClaims(idToken *oidc.IDToken) (tokens.RoxClaims, error) {
	// OIDC tokens issued for GitHub Actions have special claims, we'll reuse them.
	var claims githubActionClaims
	if err := idToken.Claims(&claims); err != nil {
		return tokens.RoxClaims{}, errors.Wrap(err, "extracting GitHub Actions claims")
	}

	return createRoxClaimsFromGitHubClaims(idToken.Subject, idToken.Audience, claims), nil
}

func createRoxClaimsFromGitHubClaims(subject string, audiences []string, claims githubActionClaims) tokens.RoxClaims {
	// This is in-line with the user ID we use for other auth providers, where a mix of username + ID wil be used.
	// In general, "|" is used as a separator for auth attributes.
	actorWithID := fmt.Sprintf("%s|%s", claims.ActorID, claims.Actor)

	userClaims := &tokens.ExternalUserClaim{
		UserID:   actorWithID,
		FullName: claims.Actor,
		Attributes: map[string][]string{
			"actor":            {claims.Actor},
			"actor_id":         {claims.ActorID},
			"repository":       {claims.Repository},
			"repository_owner": {claims.RepositoryOwner},
			"environment":      {claims.Environment},
			"event_name":       {claims.EventName},
			"ref":              {claims.GitRef},
			"sub":              {subject},
			"aud":              audiences,
		},
	}

	return tokens.RoxClaims{
		ExternalUser: userClaims,
		Name:         actorWithID,
	}
}
