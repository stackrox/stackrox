package tokenbased

import (
	"strings"

	"google.golang.org/grpc/metadata"
)

// ExtractToken extracts the token of the given type (e.g., "Bearer") from the given metadata.
func ExtractToken(md metadata.MD, tokenType string) string {
	authHeaders := md.Get("authorization")
	if len(authHeaders) != 1 {
		return ""
	}

	var tokenPrefix string
	if tokenType != "" {
		tokenPrefix = tokenType + " "
	}
	prefixLen := len(tokenPrefix)
	authHeader := authHeaders[0]
	if len(authHeader) < prefixLen || !strings.EqualFold(authHeader[:prefixLen], tokenPrefix) {
		return ""
	}

	return authHeader[prefixLen:]
}
