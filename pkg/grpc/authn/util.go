package authn

import (
	"context"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/stringutils"
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

// UserFromContext creates *storage.SlimUser object.
func UserFromContext(ctx context.Context) *storage.SlimUser {
	identity := IdentityFromContextOrNil(ctx)
	if identity == nil {
		return nil
	}
	return &storage.SlimUser{
		Id:   identity.UID(),
		Name: stringutils.FirstNonEmpty(identity.FullName(), identity.FriendlyName()),
	}
}
