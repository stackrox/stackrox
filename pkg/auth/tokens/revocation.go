package tokens

import (
	"context"
	"errors"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timeutil"
)

// RevocationLayer is a source layer that deals with token revocation.
type RevocationLayer interface {
	SourceLayer

	// Revoke records a token as revoked. expiry should be the original expiry time of the token.
	Revoke(tokenID string, expiry time.Time)
	// IsRevoked checks if the given token is revoked.
	IsRevoked(tokenID string) bool
}

// NewRevocationLayer creates a new RevocationLayer instance.
func NewRevocationLayer() RevocationLayer {
	return &revocationLayer{
		nextExpiry: timeutil.Max,
		revoked:    make(map[string]time.Time),
	}
}

type revocationLayer struct {
	nextExpiry   time.Time
	revoked      map[string]time.Time
	revokedMutex sync.RWMutex
}

func (l *revocationLayer) Validate(_ context.Context, claims *Claims) error {
	if claims.ID == "" {
		return errors.New("token has no ID")
	}

	if l.IsRevoked(claims.ID) {
		return errors.New("token has been revoked")
	}
	return nil
}

func (l *revocationLayer) Revoke(tokenID string, expiry time.Time) {
	if time.Now().After(expiry) {
		return
	}

	l.revokedMutex.Lock()
	defer l.revokedMutex.Unlock()

	if currentExpiry, found := l.revoked[tokenID]; !found || expiry.After(currentExpiry) {
		l.revoked[tokenID] = expiry
		if expiry.Before(l.nextExpiry) {
			l.nextExpiry = expiry
		}
	}
	if time.Now().After(l.nextExpiry) {
		l.cleanupNoLock()
	}
}

func (l *revocationLayer) checkRevoked(tokenID string) (revoked bool, needsCleanup bool) {
	l.revokedMutex.RLock()
	defer l.revokedMutex.RUnlock()

	_, revoked = l.revoked[tokenID]
	needsCleanup = time.Now().After(l.nextExpiry)
	return
}

func (l *revocationLayer) cleanup() {
	l.revokedMutex.Lock()
	defer l.revokedMutex.Unlock()

	l.cleanupNoLock()
}

func (l *revocationLayer) cleanupNoLock() {
	now := time.Now()
	nextExpiration := timeutil.Max
	for tokenID, expiration := range l.revoked {
		if now.After(expiration) {
			delete(l.revoked, tokenID)
		} else if expiration.Before(nextExpiration) {
			nextExpiration = expiration
		}
	}
	l.nextExpiry = nextExpiration
}

func (l *revocationLayer) IsRevoked(tokenID string) bool {
	revoked, needsCleanup := l.checkRevoked(tokenID)
	if needsCleanup {
		l.cleanup()
	}
	return revoked
}
