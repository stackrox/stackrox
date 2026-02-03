package centralproxy

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/coalescer"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/grpc/authn"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"golang.org/x/sync/errgroup"
	authenticationv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	stackroxNamespaceHeader = "ACS-AUTH-NAMESPACE-SCOPE"
	defaultCacheTTL         = 3 * time.Minute
	// k8sAPITimeout is the maximum time allowed for Kubernetes API calls (TokenReview, SubjectAccessReview).
	// This ensures that API calls don't hang indefinitely when all callers have cancelled.
	k8sAPITimeout = 30 * time.Second
)

// authzCacheKey uniquely identifies an authorization check for caching.
// It includes UID and groups to avoid over-permissive cache hits when different
// tokens for the same username have different group memberships.
type authzCacheKey struct {
	uid        string
	userGroups string // Joined group names
	namespace  string
}

// String returns a string representation of the cache key for use with singleflight.
func (k authzCacheKey) String() string {
	return fmt.Sprintf("%s+%s+%s", k.uid, k.userGroups, k.namespace)
}

// authzResult wraps the authorization result for caching.
// We need a wrapper because expiringcache treats nil values as "not found".
type authzResult struct {
	err error
}

// k8sResource represents a Kubernetes resource that requires authorization checking.
type k8sResource struct {
	Resource string
	Group    string
}

// String returns the qualified resource name in "resource.group" format.
// For core API resources (empty group), it returns "resource.core".
func (r k8sResource) String() string {
	if r.Group == "" {
		return r.Resource + ".core"
	}
	return r.Resource + "." + r.Group
}

// k8sAuthorizer verifies that a bearer token has the required Kubernetes permissions.
// It validates tokens using TokenReview and checks permissions using SubjectAccessReview.
// The user is authorized if they have get and list permissions to all deployment-like
// resources.
type k8sAuthorizer struct {
	client           kubernetes.Interface
	tokenCache       expiringcache.Cache[string, *authenticationv1.UserInfo]
	authzCache       expiringcache.Cache[authzCacheKey, *authzResult]
	verbsToCheck     []string
	resourcesToCheck []k8sResource
	// tokenReviewGroup coalesces concurrent authentication requests for the same token.
	tokenReviewGroup *coalescer.Coalescer[*authenticationv1.UserInfo]
	// authzGroup coalesces concurrent authorization requests for the same user/namespace.
	authzGroup *coalescer.Coalescer[*authzResult]
}

// newK8sAuthorizer creates a new Kubernetes-based authorizer with TokenReview and
// SubjectAccessReview caching.
func newK8sAuthorizer(client kubernetes.Interface) *k8sAuthorizer {
	return &k8sAuthorizer{
		client:           client,
		tokenCache:       expiringcache.NewExpiringCache[string, *authenticationv1.UserInfo](defaultCacheTTL),
		authzCache:       expiringcache.NewExpiringCache[authzCacheKey, *authzResult](defaultCacheTTL),
		verbsToCheck:     []string{"get", "list"},
		tokenReviewGroup: coalescer.New[*authenticationv1.UserInfo](),
		authzGroup:       coalescer.New[*authzResult](),
		resourcesToCheck: []k8sResource{
			{Resource: "pods", Group: ""},
			{Resource: "replicationcontrollers", Group: ""},
			{Resource: "daemonsets", Group: "apps"},
			{Resource: "deployments", Group: "apps"},
			{Resource: "replicasets", Group: "apps"},
			{Resource: "statefulsets", Group: "apps"},
			{Resource: "cronjobs", Group: "batch"},
			{Resource: "jobs", Group: "batch"},
			{Resource: "deploymentconfigs", Group: "apps.openshift.io"},
		},
	}
}

// formatForbiddenErr creates a consistent forbidden error message for authorization failures.
func formatForbiddenErr(user, verb string, resource k8sResource, namespace string) error {
	// Uppercase the verb for readability.
	verb = strings.ToUpper(verb)

	if namespace == FullClusterAccessScope {
		return pkghttputil.Errorf(
			http.StatusForbidden,
			"user %q lacks cluster-wide %s permission for resource %q",
			user, verb, resource.String(),
		)
	}
	return pkghttputil.Errorf(
		http.StatusForbidden,
		"user %q lacks %s permission for resource %q in namespace %q",
		user, verb, resource.String(), namespace,
	)
}

// authenticate validates the bearer token using TokenReview and returns user information.
// Successful authentications are cached and concurrent requests are coalesced to reduce load
// on the Kubernetes API server.
func (a *k8sAuthorizer) authenticate(ctx context.Context, r *http.Request) (*authenticationv1.UserInfo, error) {
	token, err := extractBearerToken(r)
	if err != nil {
		return nil, err
	}

	// Fast path: check cache first.
	if userInfo, ok := a.tokenCache.Get(token); ok {
		return userInfo, nil
	}

	// Slow path: coalesce concurrent authentication requests for the same token.
	return a.tokenReviewGroup.Coalesce(ctx, token, func() (*authenticationv1.UserInfo, error) { //nolint:wrapcheck
		// Double-check cache inside coalesce to avoid redundant API calls.
		if userInfo, ok := a.tokenCache.Get(token); ok {
			return userInfo, nil
		}

		// Use a background context with timeout to ensure the shared function is independent
		// of the initial request context while still having a bounded lifetime.
		ctx, cancel := context.WithTimeout(context.Background(), k8sAPITimeout)
		defer cancel()
		userInfo, err := a.validateToken(ctx, token)
		if err != nil {
			return nil, err
		}

		a.tokenCache.Add(token, userInfo)
		return userInfo, nil
	})
}

func extractBearerToken(r *http.Request) (string, error) {
	headers := phonehome.Headers(r.Header)
	token := authn.ExtractToken(&headers, "Bearer")
	if token == "" {
		return "", pkghttputil.NewError(http.StatusUnauthorized, "missing or invalid bearer token")
	}
	return token, nil
}

// validateToken validates the bearer token using TokenReview and returns user information.
func (a *k8sAuthorizer) validateToken(ctx context.Context, token string) (*authenticationv1.UserInfo, error) {
	tokenReview := &authenticationv1.TokenReview{
		Spec: authenticationv1.TokenReviewSpec{
			Token: token,
		},
	}

	result, err := a.client.AuthenticationV1().TokenReviews().Create(ctx, tokenReview, metav1.CreateOptions{})
	if err != nil {
		return nil, pkghttputil.Errorf(http.StatusInternalServerError, "performing token review: %v", err)
	}

	if result.Status.Error != "" {
		return nil, pkghttputil.Errorf(http.StatusUnauthorized, "token validation error: %s", result.Status.Error)
	}

	if !result.Status.Authenticated {
		return nil, pkghttputil.NewError(http.StatusUnauthorized, "token authentication failed")
	}

	return &result.Status.User, nil
}

// authorize checks if the authenticated user has required permissions.
// Authorization behavior is determined by the namespace header:
//   - Empty: No SubjectAccessReview and minimal rox token with empty access scope.
//   - Specific namespace: SubjectAccessReview for the namespace and rox token with
//     access scope limited to the namespace.
//   - FullClusterAccessScope ("*"): SubjectAccessReview for all namespaces and rox token
//     with cluster-wide access scope.
//
// Successful authorizations are cached and concurrent requests are coalesced to reduce load
// on the Kubernetes API server.
func (a *k8sAuthorizer) authorize(ctx context.Context, userInfo *authenticationv1.UserInfo, r *http.Request) error {
	namespace := r.Header.Get(stackroxNamespaceHeader)
	// Skip authorization if the namespace header is empty or not set.
	if namespace == "" {
		return nil
	}

	// Fast path: check cache first.
	cacheKey := a.buildAuthzCacheKey(userInfo, namespace)
	if cached, ok := a.authzCache.Get(cacheKey); ok {
		return cached.err
	}

	// Slow path: coalesce concurrent authorization requests for the same user/namespace.
	cached, err := a.authzGroup.Coalesce(ctx, cacheKey.String(), func() (*authzResult, error) {
		// Double-check cache inside coalesce to avoid redundant API calls.
		if cached, ok := a.authzCache.Get(cacheKey); ok {
			return cached, nil
		}

		log.Debugf("Authorization cache miss for user %q (uid=%q) in namespace %q", userInfo.Username, userInfo.UID, namespace)

		// Use a background context with timeout to ensure the shared function is independent
		// of the initial request context while still having a bounded lifetime.
		ctx, cancel := context.WithTimeout(context.Background(), k8sAPITimeout)
		defer cancel()
		result := a.checkAllPermissions(ctx, userInfo, namespace)
		// Only cache successful authorizations and permission denials (403 Forbidden).
		// Transient errors should not be cached so callers can retry.
		if result.err == nil || pkghttputil.StatusFromError(result.err) == http.StatusForbidden {
			a.authzCache.Add(cacheKey, result)
		}
		return result, nil
	})
	if err != nil {
		return err //nolint:wrapcheck
	}
	return cached.err
}

// buildAuthzCacheKey creates a cache key for authorization based on user identity and namespace.
func (a *k8sAuthorizer) buildAuthzCacheKey(userInfo *authenticationv1.UserInfo, namespace string) authzCacheKey {
	// Sort groups to make the cache key order-independent.
	sortedGroups := append([]string(nil), userInfo.Groups...)
	slices.Sort(sortedGroups)

	return authzCacheKey{
		uid:        userInfo.UID,
		userGroups: strings.Join(sortedGroups, "|"),
		namespace:  namespace,
	}
}

// checkAllPermissions runs all SubjectAccessReview checks in parallel.
func (a *k8sAuthorizer) checkAllPermissions(ctx context.Context, userInfo *authenticationv1.UserInfo, namespace string) *authzResult {
	// Use errgroup with context cancellation to short-circuit on first error/denial.
	g, groupCtx := errgroup.WithContext(ctx)

	for _, resource := range a.resourcesToCheck {
		for _, verb := range a.verbsToCheck {
			resource := resource

			g.Go(func() error {
				allowed, err := a.performSubjectAccessReview(groupCtx, userInfo, verb, namespace, resource)
				if err != nil {
					return pkghttputil.Errorf(http.StatusInternalServerError,
						"checking %s permission for %q: %v", verb, resource, err)
				}
				if !allowed {
					return formatForbiddenErr(userInfo.Username, verb, resource, namespace)
				}
				return nil
			})
		}
	}

	return &authzResult{err: g.Wait()}
}

// performSubjectAccessReview performs a SubjectAccessReview API call.
func (a *k8sAuthorizer) performSubjectAccessReview(ctx context.Context, userInfo *authenticationv1.UserInfo, verb, namespace string, resource k8sResource) (bool, error) {
	// In SubjectAccessReview an empty namespace means full cluster access.
	namespaceScope := namespace
	if namespace == FullClusterAccessScope {
		namespaceScope = ""
	}

	sar := &authv1.SubjectAccessReview{
		Spec: authv1.SubjectAccessReviewSpec{
			User:   userInfo.Username,
			Groups: userInfo.Groups,
			UID:    userInfo.UID,
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace: namespaceScope,
				Verb:      verb,
				Resource:  resource.Resource,
				Group:     resource.Group,
			},
		},
	}

	result, err := a.client.AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
	if err != nil {
		return false, errors.Wrap(err, "performing subject access review")
	}

	if result.Status.EvaluationError != "" {
		return false, errors.Errorf("authorization evaluation error: %s", result.Status.EvaluationError)
	}

	return result.Status.Allowed, nil
}
