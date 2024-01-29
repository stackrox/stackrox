package basic

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
)

// Extractor is the identity extractor for the basic auth identity.
type Extractor struct {
	manager      *Manager
	authProvider authproviders.Provider
}

func parseBasicAuthToken(basicAuthToken string) (string, string, error) {
	decoded, err := base64.StdEncoding.DecodeString(basicAuthToken)
	if err != nil {
		return "", "", err
	}

	decodedStr := string(decoded)
	parts := strings.SplitN(decodedStr, ":", 2)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("malformed basic auth token: %q", decodedStr)
	}
	return parts[0], parts[1], nil
}

func getExtractorError(msg string, err error) *authn.ExtractorError {
	return authn.NewExtractorError("basic", msg, err)
}

// IdentityForRequest returns an identity for the given request if it contains valid basic auth credentials.
// If non-nil, the returned identity implements `basic.Identity`.
func (e *Extractor) IdentityForRequest(ctx context.Context, ri requestinfo.RequestInfo) (authn.Identity, *authn.ExtractorError) {
	md := metautils.NiceMD(ri.Metadata)
	authHeader := md.Get("Authorization")
	if authHeader == "" {
		return nil, nil
	}

	basicAuthToken := strings.TrimPrefix(authHeader, "Basic ")
	if len(basicAuthToken) == len(authHeader) {
		return nil, nil // not basic auth
	}

	username, password, err := parseBasicAuthToken(basicAuthToken)
	if err != nil {
		return nil, getExtractorError("failed to parse basic auth token", err)
	}

	id, err := e.manager.IdentityForCreds(ctx, username, password, e.authProvider)
	if err != nil {
		return nil, getExtractorError(fmt.Sprintf("failed to identify user with username %q", username), err)
	}

	return id, nil
}

// NewExtractor returns a new identity extractor for basic auth.
func NewExtractor(mgr *Manager, authProvider authproviders.Provider) (*Extractor, error) {
	return &Extractor{
		manager:      mgr,
		authProvider: authProvider,
	}, nil
}
