package auth

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// UserInfo represents information about an authenticated user
type UserInfo struct {
	Username string   `json:"username"`
	UID      string   `json:"uid"`
	Groups   []string `json:"groups"`
}

// Validator handles OpenShift token validation
type Validator struct {
	httpClient *http.Client
	cache      *userInfoCache
}

// NewValidator creates a new OpenShift token validator
func NewValidator() *Validator {
	// Create HTTP client for Kubernetes API calls
	// Note: In production, this should use proper CA validation
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // TODO: Use proper CA validation
			},
		},
	}

	return &Validator{
		httpClient: client,
		cache:      newUserInfoCache(),
	}
}

// ValidateToken validates an OpenShift access token and returns user information
func (v *Validator) ValidateToken(ctx context.Context, token string) (*UserInfo, error) {
	// Check cache first
	if userInfo := v.cache.get(token); userInfo != nil {
		log.Debug("Using cached user info")
		return userInfo, nil
	}

	// Call OpenShift API to validate token and get user info
	userInfo, err := v.validateTokenWithAPI(ctx, token)
	if err != nil {
		return nil, err
	}

	// Cache the result
	v.cache.set(token, userInfo)

	return userInfo, nil
}

// validateTokenWithAPI calls the OpenShift API to validate the token
func (v *Validator) validateTokenWithAPI(ctx context.Context, token string) (*UserInfo, error) {
	// Call the Kubernetes API to get user information
	// This endpoint returns information about the user associated with the token
	req, err := http.NewRequestWithContext(ctx, "GET", "https://kubernetes.default.svc/api/v1/users/~", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call Kubernetes API")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("Kubernetes API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var k8sUser struct {
		Metadata struct {
			Name string `json:"name"`
			UID  string `json:"uid"`
		} `json:"metadata"`
		Groups []string `json:"groups"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&k8sUser); err != nil {
		return nil, errors.Wrap(err, "failed to decode user response")
	}

	userInfo := &UserInfo{
		Username: k8sUser.Metadata.Name,
		UID:      k8sUser.Metadata.UID,
		Groups:   k8sUser.Groups,
	}

	log.Debugf("Successfully validated token for user: %s", userInfo.Username)
	return userInfo, nil
}

// userInfoCache provides simple in-memory caching for user information
type userInfoCache struct {
	cache map[string]*cacheEntry
	mutex sync.RWMutex
}

type cacheEntry struct {
	userInfo  *UserInfo
	expiresAt time.Time
}

func newUserInfoCache() *userInfoCache {
	c := &userInfoCache{
		cache: make(map[string]*cacheEntry),
	}
	
	// Start cleanup goroutine
	go c.cleanup()
	
	return c
}

func (c *userInfoCache) get(token string) *UserInfo {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.cache[token]
	if !exists || time.Now().After(entry.expiresAt) {
		return nil
	}

	return entry.userInfo
}

func (c *userInfoCache) set(token string, userInfo *UserInfo) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Cache for 5 minutes
	c.cache[token] = &cacheEntry{
		userInfo:  userInfo,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
}

func (c *userInfoCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mutex.Lock()
		now := time.Now()
		for token, entry := range c.cache {
			if now.After(entry.expiresAt) {
				delete(c.cache, token)
			}
		}
		c.mutex.Unlock()
	}
}