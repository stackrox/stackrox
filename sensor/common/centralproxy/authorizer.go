package centralproxy

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/grpc/authn"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	authenticationv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"golang.org/x/sync/errgroup"
)

const (
	stackroxNamespaceHeader = "ACS-AUTH-NAMESPACE-SCOPE"
	defaultCacheTTL         = 3 * time.Minute
)

// sarCacheKey uniquely identifies a SubjectAccessReview check for caching.
// It includes UID and groups to avoid over-permissive cache hits when different
// tokens for the same username have different group memberships.
type sarCacheKey struct {
	uid        string
	userGroups string // Joined group names
	namespace  string
	verb       string
	resource   string
	group      string
}

// k8sAuthorizer verifies that a bearer token has the required Kubernetes permissions.
// It validates tokens using TokenReview and checks permissions using SubjectAccessReview.
// The user is authorized if they have get and list permissions to all deployment-like
// resources.
type k8sAuthorizer struct {
	client           kubernetes.Interface
	tokenCache       expiringcache.Cache[string, *authenticationv1.UserInfo]
	sarCache         expiringcache.Cache[sarCacheKey, bool]
	verbsToCheck     []string
	resourcesToCheck []struct {
		Resource string
		Group    string
	}
}

// newK8sAuthorizer creates a new Kubernetes-based authorizer with TokenReview and
// SubjectAccessReview caching.
func newK8sAuthorizer(client kubernetes.Interface) *k8sAuthorizer {
	return &k8sAuthorizer{
		client:       client,
		tokenCache:   expiringcache.NewExpiringCache[string, *authenticationv1.UserInfo](defaultCacheTTL),
		sarCache:     expiringcache.NewExpiringCache[sarCacheKey, bool](defaultCacheTTL),
		verbsToCheck: []string{"get", "list"},
		resourcesToCheck: []struct {
			Resource string
			Group    string
		}{
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
func formatForbiddenErr(user, verb, resource, group, namespace string) error {
	// Uppercase the verb for readability.
	verb = strings.ToUpper(verb)

	// Format as resource.group using "core" for empty group.
	qualifiedResource := resource + "." + group
	if group == "" {
		qualifiedResource = resource + ".core"
	}

	if namespace == FullClusterAccessScope {
		return pkghttputil.Errorf(
			http.StatusForbidden,
			"user %q lacks cluster-wide %s permission for resource %q",
			user, verb, qualifiedResource,
		)
	}
	return pkghttputil.Errorf(
		http.StatusForbidden,
		"user %q lacks %s permission for resource %q in namespace %q",
		user, verb, qualifiedResource, namespace,
	)
}

// authenticate validates the bearer token using TokenReview and returns user information.
func (a *k8sAuthorizer) authenticate(ctx context.Context, r *http.Request) (*authenticationv1.UserInfo, error) {
	token, err := extractBearerToken(r)
	if err != nil {
		return nil, err
	}
	return a.validateToken(ctx, token)
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
// Successful authentications are cached to reduce API calls to the Kubernetes API server.
func (a *k8sAuthorizer) validateToken(ctx context.Context, token string) (*authenticationv1.UserInfo, error) {
	if userInfo, ok := a.tokenCache.Get(token); ok {
		return userInfo, nil
	}

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

	a.tokenCache.Add(token, &result.Status.User)
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
// SAR checks are performed in parallel to reduce latency.
func (a *k8sAuthorizer) authorize(ctx context.Context, userInfo *authenticationv1.UserInfo, r *http.Request) error {
	namespace := r.Header.Get(stackroxNamespaceHeader)
	// Skip authorization if the namespace header is empty or not set.
	if namespace == "" {
		return nil
	}

	// Use errgroup with context cancellation to short-circuit on first error/denial.
	g, ctx := errgroup.WithContext(ctx)

	// Track the first authorization failure for a consistent error message.
	var (
		firstDenial     error
		firstDenialOnce sync.Once
	)

	for _, resource := range a.resourcesToCheck {
		for _, verb := range a.verbsToCheck {
			// Capture loop variables for the goroutine.
			resource := resource
			verb := verb

			g.Go(func() error {
				allowed, err := a.performSubjectAccessReview(ctx, userInfo, verb, namespace, resource)
				if err != nil {
					return pkghttputil.Errorf(http.StatusInternalServerError, "checking %s permission for %s: %v", verb, resource.Resource, err)
				}
				if !allowed {
					denial := formatForbiddenErr(userInfo.Username, verb, resource.Resource, resource.Group, namespace)
					firstDenialOnce.Do(func() {
						firstDenial = denial
					})
					return denial
				}
				return nil
			})
		}
	}

	if err := g.Wait(); err != nil {
		// Return the first denial if available (more user-friendly), otherwise the error.
		if firstDenial != nil {
			return firstDenial
		}
		return err
	}

	return nil
}

func (a *k8sAuthorizer) performSubjectAccessReview(ctx context.Context, userInfo *authenticationv1.UserInfo, verb, namespace string, resource struct {
	Resource string
	Group    string
},
) (bool, error) {
	// Sort groups to make the cache key order-independent.
	sortedGroups := append([]string(nil), userInfo.Groups...)
	slices.Sort(sortedGroups)

	cacheKey := sarCacheKey{
		uid:        userInfo.UID,
		userGroups: strings.Join(sortedGroups, "|"),
		namespace:  namespace,
		verb:       verb,
		resource:   resource.Resource,
		group:      resource.Group,
	}
	if allowed, ok := a.sarCache.Get(cacheKey); ok {
		return allowed, nil
	}
	log.Debugf(
		"Cache miss for subject access review to perform %s on %s.%s in %s (user=%q, uid=%q, userGroups=%q)",
		cacheKey.verb, cacheKey.group, cacheKey.resource, cacheKey.namespace, userInfo.Username, cacheKey.uid, cacheKey.userGroups,
	)

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
		return false, pkghttputil.Errorf(http.StatusInternalServerError, "performing subject access review: %v", err)
	}

	if result.Status.EvaluationError != "" {
		return false, pkghttputil.Errorf(http.StatusInternalServerError, "authorization evaluation error: %s", result.Status.EvaluationError)
	}

	a.sarCache.Add(cacheKey, result.Status.Allowed)
	return result.Status.Allowed, nil
}
