package rbac

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/centralproxy/pkg/auth"
	"github.com/stackrox/rox/pkg/logging"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	log = logging.LoggerForModule()
)

const (
	// Virtual API group for StackRox security resources
	securityAPIGroup = "security.stackrox.io"
	securityAPIVersion = "v1"
)

// Checker handles Kubernetes RBAC permission checks
type Checker struct {
	k8sClient kubernetes.Interface
}

// NewChecker creates a new RBAC checker
func NewChecker(k8sClient kubernetes.Interface) *Checker {
	return &Checker{
		k8sClient: k8sClient,
	}
}

// CheckAccess verifies if a user has permission to access a specific resource
func (c *Checker) CheckAccess(ctx context.Context, userInfo *auth.UserInfo, resource, verb string) bool {
	// Create SubjectAccessReview to check permissions
	sar := &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			User:   userInfo.Username,
			Groups: userInfo.Groups,
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Group:    securityAPIGroup,
				Version:  securityAPIVersion,
				Resource: resource,
				Verb:     verb,
			},
		},
	}

	// Add timeout to the context
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Perform the access review
	result, err := c.k8sClient.AuthorizationV1().
		SubjectAccessReviews().
		Create(ctx, sar, metav1.CreateOptions{})

	if err != nil {
		log.Errorf("Failed to perform SubjectAccessReview: %v", err)
		return false
	}

	allowed := result.Status.Allowed
	if !allowed {
		log.Debugf("Access denied for user %s: %s:%s (reason: %s)", 
			userInfo.Username, resource, verb, result.Status.Reason)
	} else {
		log.Debugf("Access granted for user %s: %s:%s", 
			userInfo.Username, resource, verb)
	}

	return allowed
}

// CheckNamespacedAccess verifies access to namespace-scoped resources
func (c *Checker) CheckNamespacedAccess(ctx context.Context, userInfo *auth.UserInfo, resource, verb, namespace string) bool {
	sar := &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			User:   userInfo.Username,
			Groups: userInfo.Groups,
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Group:     securityAPIGroup,
				Version:   securityAPIVersion,
				Resource:  resource,
				Verb:      verb,
				Namespace: namespace,
			},
		},
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := c.k8sClient.AuthorizationV1().
		SubjectAccessReviews().
		Create(ctx, sar, metav1.CreateOptions{})

	if err != nil {
		log.Errorf("Failed to perform SubjectAccessReview: %v", err)
		return false
	}

	allowed := result.Status.Allowed
	if !allowed {
		log.Debugf("Access denied for user %s: %s:%s in namespace %s (reason: %s)", 
			userInfo.Username, resource, verb, namespace, result.Status.Reason)
	} else {
		log.Debugf("Access granted for user %s: %s:%s in namespace %s", 
			userInfo.Username, resource, verb, namespace)
	}

	return allowed
}

// CheckMultipleAccess checks multiple permissions at once and returns which ones are allowed
func (c *Checker) CheckMultipleAccess(ctx context.Context, userInfo *auth.UserInfo, permissions []Permission) (map[Permission]bool, error) {
	results := make(map[Permission]bool)

	for _, perm := range permissions {
		var allowed bool
		if perm.Namespace != "" {
			allowed = c.CheckNamespacedAccess(ctx, userInfo, perm.Resource, perm.Verb, perm.Namespace)
		} else {
			allowed = c.CheckAccess(ctx, userInfo, perm.Resource, perm.Verb)
		}
		results[perm] = allowed
	}

	return results, nil
}

// Permission represents a required RBAC permission
type Permission struct {
	Resource  string
	Verb      string
	Namespace string // Optional for namespace-scoped permissions
}

// VirtualGVRMapping defines how GraphQL fields map to virtual GVR permissions
var VirtualGVRMapping = map[string][]Permission{
	"images": {
		{Resource: "images", Verb: "list"},
	},
	"vulnerabilities": {
		{Resource: "vulnerabilities", Verb: "list"},
	},
	"imageVulnerabilities": {
		// Hierarchical: images permission includes vulnerability data
		{Resource: "images", Verb: "list"},
	},
	"policies": {
		{Resource: "policies", Verb: "list"},
	},
	"violations": {
		{Resource: "violations", Verb: "list"},
	},
}