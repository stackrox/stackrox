package basic

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync/atomic"
	"unsafe"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/htpasswd"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
)

// Extractor is the identity extractor for the basic auth identity.
type Extractor struct {
	hashFilePtr  unsafe.Pointer
	userRole     *storage.Role
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

func (e *Extractor) hashFile() *htpasswd.HashFile {
	return (*htpasswd.HashFile)(atomic.LoadPointer(&e.hashFilePtr))
}

// SetHashFile sets the hash file to be used for basic auth.
func (e *Extractor) SetHashFile(hashFile *htpasswd.HashFile) {
	atomic.StorePointer(&e.hashFilePtr, unsafe.Pointer(hashFile))
}

// IdentityForRequest returns an identity for the given request if it contains valid basic auth credentials.
// If non-nil, the returned identity implements `basic.Identity`.
func (e *Extractor) IdentityForRequest(_ context.Context, ri requestinfo.RequestInfo) (authn.Identity, error) {
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
		return nil, err
	}

	if !e.hashFile().Check(username, password) {
		return nil, errors.New("invalid username and/or password")
	}

	return identity{
		username:     username,
		role:         e.userRole,
		authProvider: e.authProvider,
	}, nil
}

// NewExtractor returns a new identity extractor for basic auth.
func NewExtractor(hashFile *htpasswd.HashFile, userRole *storage.Role, authProvider authproviders.Provider) (*Extractor, error) {
	return &Extractor{
		hashFilePtr:  unsafe.Pointer(hashFile),
		userRole:     userRole,
		authProvider: authProvider,
	}, nil
}
