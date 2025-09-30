package m2m

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/tokens"
)

type githubClaimExtractor struct{}

var _ claimExtractor = (*githubClaimExtractor)(nil)

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

func (g *githubClaimExtractor) ExtractRoxClaims(claims map[string][]string) (tokens.RoxClaims, error) {
	// This is in-line with the user ID we use for other auth providers, where a mix of username + ID wil be used.
	// In general, "|" is used as a separator for auth attributes.
	actorWithID := fmt.Sprintf("%s|%s", claims["actor_id"][0], claims["actor"][0])

	userClaims := &tokens.ExternalUserClaim{
		UserID:     actorWithID,
		FullName:   claims["actor"][0],
		Attributes: claims,
	}
	return tokens.RoxClaims{
		ExternalUser: userClaims,
		Name:         actorWithID,
	}, nil
}

func (g *githubClaimExtractor) ExtractClaims(idToken *IDToken) (map[string][]string, error) {
	// OIDC tokens issued for GitHub Actions have special claims, we'll reuse them.
	var claims githubActionClaims
	if err := idToken.Claims(&claims); err != nil {
		return nil, errors.Wrap(err, "extracting GitHub Actions claims")
	}
	return map[string][]string{
		"sub": {idToken.Subject},
		"aud": idToken.Audience,

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
	}, nil
}
