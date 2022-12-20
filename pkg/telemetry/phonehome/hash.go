package phonehome

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/stackrox/rox/pkg/grpc/authn"
)

func hash(id string) string {
	sha := sha256.Sum256([]byte(id))
	return base64.StdEncoding.EncodeToString(sha[:])
}

// HashUserID anonymizes user ID so that it can be sent to the external
// telemetry storage for product data analysis.
func (cfg *Config) HashUserID(userID, authProviderID string) string {
	clientID := "unknown"
	if cfg != nil {
		clientID = cfg.ClientID
	}
	if userID == "" {
		userID = "unauthenticated"
	}
	if authProviderID == "" {
		authProviderID = "unknown"
	}
	return hash(fmt.Sprintf("%s:%s:%s", clientID, authProviderID, userID))
}

// HashUserAuthID extracts the user and auth provider from the provided identity
// and anonymizes the couple so that it can be sent to the external telemetry
// storage for product data analysis.
func (cfg *Config) HashUserAuthID(id authn.Identity) string {
	var userID, providerID string
	if id != nil {
		userID = id.UID()
		if provider := id.ExternalAuthProvider(); provider != nil {
			providerID = provider.ID()
		}
	}
	return cfg.HashUserID(userID, providerID)
}
