package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// CacheKey represents a composite key for token cache lookups.
// The key is based on the user identifier, namespace, and deployment.
// This ensures that tokens are scoped per-user and per-resource.
type CacheKey struct {
	UserID     string
	Namespace  string
	Deployment string
}

// Key generates a deterministic string representation of the cache key.
// It uses SHA-256 hashing to ensure a consistent length and avoid
// issues with special characters in user identifiers.
func (k CacheKey) Key() string {
	// Create a consistent string representation
	raw := fmt.Sprintf("user=%s|ns=%s|deploy=%s", k.UserID, k.Namespace, k.Deployment)

	// Hash to ensure consistent length and avoid character issues
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

// String returns a human-readable representation of the key (for logging).
func (k CacheKey) String() string {
	if k.Deployment != "" {
		return fmt.Sprintf("user=%s, namespace=%s, deployment=%s", k.UserID, k.Namespace, k.Deployment)
	}
	if k.Namespace != "" {
		return fmt.Sprintf("user=%s, namespace=%s (all deployments)", k.UserID, k.Namespace)
	}
	return fmt.Sprintf("user=%s (all namespaces)", k.UserID)
}

// NewCacheKey creates a new cache key from the given parameters.
func NewCacheKey(userID, namespace, deployment string) CacheKey {
	return CacheKey{
		UserID:     userID,
		Namespace:  namespace,
		Deployment: deployment,
	}
}
