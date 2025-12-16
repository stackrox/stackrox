package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	authnv1 "k8s.io/api/authentication/v1"
	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// SARTimeout is the timeout for SubjectAccessReview requests
	SARTimeout = 5 * time.Second
)

var (
	log = logging.LoggerForModule()
)

// K8sUserInfo contains information about a Kubernetes user extracted from a token.
type K8sUserInfo struct {
	Username string
	UID      string
	Groups   []string
}

// K8sValidator validates Kubernetes RBAC permissions via SubjectAccessReview.
type K8sValidator struct {
	k8sClient kubernetes.Interface
}

// NewK8sValidator creates a new Kubernetes authorization validator.
func NewK8sValidator(k8sClient kubernetes.Interface) *K8sValidator {
	return &K8sValidator{
		k8sClient: k8sClient,
	}
}

// ValidateDeploymentAccess checks if the given user (from token) has read access
// to the specified deployment in the namespace.
//
// This performs a Kubernetes SubjectAccessReview to check if the user can:
// - get deployments in the namespace (if deployment is specified)
// - get deployments in all namespaces (if namespace is empty)
//
// Returns the user info if authorized, or an error if denied or if the check fails.
func (v *K8sValidator) ValidateDeploymentAccess(ctx context.Context, token, namespace, deployment string) (*K8sUserInfo, error) {
	// First, extract user info from the token via TokenReview
	userInfo, err := v.extractUserFromToken(ctx, token)
	if err != nil {
		log.Warnw("Failed to extract user from token", logging.Err(err))
		return nil, errors.Wrap(errox.NoCredentials, "invalid or expired token")
	}

	// Build SubjectAccessReview for deployment access
	sar := &authzv1.SubjectAccessReview{
		Spec: authzv1.SubjectAccessReviewSpec{
			User:   userInfo.Username,
			Groups: userInfo.Groups,
			UID:    userInfo.UID,
			ResourceAttributes: &authzv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      "get",
				Group:     "apps",
				Resource:  "deployments",
			},
		},
	}

	// If a specific deployment is requested, include it in the SAR
	if deployment != "" {
		sar.Spec.ResourceAttributes.Name = deployment
	}

	// Create context with timeout
	sarCtx, cancel := context.WithTimeout(ctx, SARTimeout)
	defer cancel()

	// Perform SubjectAccessReview
	result, err := v.k8sClient.AuthorizationV1().SubjectAccessReviews().Create(
		sarCtx,
		sar,
		metav1.CreateOptions{},
	)
	if err != nil {
		log.Errorw("SubjectAccessReview failed",
			logging.Err(err),
			logging.String("user", userInfo.Username),
			logging.String("namespace", namespace),
			logging.String("deployment", deployment),
		)
		return nil, errors.Wrap(errox.ServerError, "authorization check failed")
	}

	// Check if access is allowed
	if !result.Status.Allowed {
		log.Infow("Access denied by Kubernetes RBAC",
			logging.String("user", userInfo.Username),
			logging.String("namespace", namespace),
			logging.String("deployment", deployment),
			logging.String("reason", result.Status.Reason),
		)
		return nil, errox.NotAuthorized.Newf(
			"user %q does not have permission to access deployment %q in namespace %q: %s",
			userInfo.Username,
			deployment,
			namespace,
			result.Status.Reason,
		)
	}

	log.Infow("Access granted by Kubernetes RBAC",
		logging.String("user", userInfo.Username),
		logging.String("namespace", namespace),
		logging.String("deployment", deployment),
	)

	return userInfo, nil
}

// extractUserFromToken uses Kubernetes TokenReview to extract user information
// from the provided bearer token.
func (v *K8sValidator) extractUserFromToken(ctx context.Context, token string) (*K8sUserInfo, error) {
	// Create TokenReview request
	tr := &authnv1.TokenReview{
		Spec: authnv1.TokenReviewSpec{
			Token: token,
		},
	}

	// Create context with timeout
	trCtx, cancel := context.WithTimeout(ctx, SARTimeout)
	defer cancel()

	// Perform TokenReview
	result, err := v.k8sClient.AuthenticationV1().TokenReviews().Create(
		trCtx,
		tr,
		metav1.CreateOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("token review failed: %w", err)
	}

	// Check if token is authenticated
	if !result.Status.Authenticated {
		return nil, errox.NoCredentials.New("token is not authenticated")
	}

	// Extract user information
	userInfo := &K8sUserInfo{
		Username: result.Status.User.Username,
		UID:      result.Status.User.UID,
		Groups:   result.Status.User.Groups,
	}

	if userInfo.Username == "" {
		return nil, errox.NoCredentials.New("token did not provide a username")
	}

	return userInfo, nil
}
