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
	Actor                string `json:"actor"`
	ActorID              string `json:"actor_id"`
	BaseRef              string `json:"base_ref"`
	Environment          string `json:"environment"`
	EventName            string `json:"event_name"`
	GitRef               string `json:"ref"`
	HeadRef              string `json:"head_ref"`
	JobWorkflowRef       string `json:"job_workflow_ref"`
	JobWorkflowSHA       string `json:"job_workflow_sha"`
	RefType              string `json:"ref_type"`
	Repository           string `json:"repository"`
	RepositoryID         string `json:"repository_id"`
	RepositoryOwner      string `json:"repository_owner"`
	RepositoryOwnerID    string `json:"repository_owner_id"`
	RepositoryVisibility string `json:"repository_visibility"`
	RunID                string `json:"run_id"`
	RunNumber            string `json:"run_number"`
	RunAttempt           string `json:"run_attempt"`
	RunnerEnvironment    string `json:"runner_environment"`
	Workflow             string `json:"workflow"`
	WorkflowRef          string `json:"workflow_ref"`
	WorkflowSHA          string `json:"workflow_sha"`
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
			"sub": {subject},
			"aud": audiences,

			"actor":                 {claims.Actor},
			"actor_id":              {claims.ActorID},
			"base_ref":              {claims.BaseRef},
			"environment":           {claims.Environment},
			"event_name":            {claims.EventName},
			"head_ref":              {claims.HeadRef},
			"job_workflow_ref":      {claims.JobWorkflowRef},
			"job_workflow_sha":      {claims.JobWorkflowSHA},
			"ref":                   {claims.GitRef},
			"ref_type":              {claims.RefType},
			"repository":            {claims.Repository},
			"repository_id":         {claims.RepositoryID},
			"repository_owner":      {claims.RepositoryOwner},
			"repository_owner_id":   {claims.RepositoryOwnerID},
			"repository_visibility": {claims.RepositoryVisibility},
			"run_id":                {claims.RunID},
			"run_number":            {claims.RunNumber},
			"run_attempt":           {claims.RunAttempt},
			"runner_environment":    {claims.RunnerEnvironment},
			"workflow":              {claims.Workflow},
			"workflow_ref":          {claims.WorkflowRef},
			"workflow_sha":          {claims.WorkflowSHA},
		},
	}

	return tokens.RoxClaims{
		ExternalUser: userClaims,
		Name:         actorWithID,
	}
}
