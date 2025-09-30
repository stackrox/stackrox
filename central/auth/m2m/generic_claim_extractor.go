package m2m

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
)

var _ claimExtractor = (*genericClaimExtractor)(nil)

type genericClaimExtractor struct{}

func (g *genericClaimExtractor) ExtractRoxClaims(claims map[string][]string) (tokens.RoxClaims, error) {
	friendlyName := getFriendlyName(claims)
	subject := strings.Join(claims["sub"], ":")
	userID := utils.IfThenElse(friendlyName == "", subject,
		fmt.Sprintf("%s|%s", subject, friendlyName))

	userClaims := &tokens.ExternalUserClaim{
		UserID:     userID,
		FullName:   stringutils.FirstNonEmpty(friendlyName, userID),
		Attributes: claims,
	}

	return tokens.RoxClaims{
		ExternalUser: userClaims,
		Name:         userID,
	}, nil
}

func getFriendlyName(claims map[string][]string) string {
	// These are some sample claims that typically have the user's name or email.
	userNameClaims := []string{
		"email",
		"preferred_username",
		"full_name",
	}

	for _, userNameClaim := range userNameClaims {
		if value, ok := claims[userNameClaim]; ok && len(value) == 1 {
			return value[0]
		}
	}

	return ""
}

func (g *genericClaimExtractor) ExtractClaims(idToken *IDToken) (map[string][]string, error) {
	var unstructured map[string]interface{}
	if err := idToken.Claims(&unstructured); err != nil {
		return nil, errors.Wrap(err, "extracting claims")
	}

	claims := make(map[string][]string, len(unstructured)+2)
	claims["sub"] = []string{idToken.Subject}
	claims["aud"] = idToken.Audience
	for key, value := range unstructured {
		switch value := value.(type) {
		case string:
			claims[key] = []string{value}
		case []string:
			claims[key] = value
		case []any:
			for _, v := range value {
				if s, ok := v.(string); ok {
					claims[key] = append(claims[key], s)
				}
			}
		default:
			log.Debugf("Dropping value %v for claim %s since its a nested claim or a non-string type %T", value, key, value)
		}
	}
	return claims, nil
}
