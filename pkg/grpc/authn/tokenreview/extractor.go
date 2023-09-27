package tokenreview

import (
	"context"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	v1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	authenticationV1 "k8s.io/client-go/kubernetes/typed/authentication/v1"
)

// NewExtractor returns a new token-based identity extractor.
func NewExtractor(roleStore permissions.RoleStore) authn.IdentityExtractor {
	return &extractor{
		roleStore: roleStore,
	}
}

type extractor struct {
	roleStore permissions.RoleStore
	validator tokens.Validator
}

func (e *extractor) IdentityForRequest(ctx context.Context, ri requestinfo.RequestInfo) (authn.Identity, error) {
	rawToken := authn.ExtractToken(ri.Metadata, "K8sToken")
	if rawToken == "" {
		logging.GetRateLimitedLogger().Warn("No K8sToken header")
		return nil, nil
	}
	config, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		logging.GetRateLimitedLogger().Warnf("Error getting k8s in-cluster config: %w", err)
		return nil, err
	}
	authClient, err := authenticationV1.NewForConfig(config)
	if err != nil {
		logging.GetRateLimitedLogger().Warnf("Error getting k8s auth client: %w", err)
		return nil, err
	}
	reviewResult, err := authClient.TokenReviews().Create(ctx, &v1.TokenReview{
		Spec: v1.TokenReviewSpec{
			Token: rawToken,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		logging.GetRateLimitedLogger().Warnf("Error getting token review: %w", err)
		return nil, err
	}
	if reviewResult.Status.Error != "" {
		logging.GetRateLimitedLogger().Warnf("Error getting token review: %s", reviewResult.Status.Error)
		return nil, nil
	}
	if !reviewResult.Status.Authenticated {
		logging.GetRateLimitedLogger().Warnf("Is not authenticated")
		return nil, nil
	}
	roles := make([]permissions.ResolvedRole, 0)
	if hasClusterAdminsGroup(reviewResult) {
		adminRole, err := e.roleStore.GetAndResolveRole(ctx, "Admin")
		if err != nil {
			logging.GetRateLimitedLogger().Warnf("Error getting admin role: %w", err)
			return nil, err
		}
		roles = append(roles, adminRole)
	}
	if isServiceAccount(reviewResult) {
		analystRole, err := e.roleStore.GetAndResolveRole(ctx, "Analyst")
		if err != nil {
			logging.GetRateLimitedLogger().Warnf("Error getting analyst role: %w", err)
			return nil, err
		}
		roles = append(roles, analystRole)
	}
	return &k8sBasedIdentity{
		uid:           reviewResult.Status.User.UID,
		username:      reviewResult.Status.User.Username,
		resolvedRoles: roles,
		attributes:    extractAttributes(reviewResult),
	}, nil
}

func extractAttributes(result *v1.TokenReview) map[string][]string {
	output := map[string][]string{}
	output[authproviders.UseridAttribute] = []string{result.Status.User.UID}
	output[authproviders.NameAttribute] = []string{result.Status.User.Username}
	output[authproviders.GroupsAttribute] = result.Status.User.Groups
	return output
}

func hasClusterAdminsGroup(result *v1.TokenReview) bool {
	for _, group := range result.Status.User.Groups {
		if group == "system:cluster-admins" {
			return true
		}
	}
	return false
}

func isServiceAccount(result *v1.TokenReview) bool {
	for _, group := range result.Status.User.Groups {
		if group == "system:serviceaccounts" {
			return true
		}
	}
	return false
}
